package repair_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/generate"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/repair"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

func TestRepairRecomputesTotals(t *testing.T) {
	// Document non conforme (totaux cassés) via le générateur.
	d, _ := generate.Generate(2, generate.Options{Scenario: generate.ScenarioSimple, Seed: 5, Invalid: true})
	if !rules.Validate(d, "en16931").HasErrors() {
		t.Fatal("le document de départ devrait être non conforme")
	}
	changes := repair.Repair(d, repair.Options{})
	if len(changes) == 0 {
		t.Error("repair aurait dû appliquer au moins un changement")
	}
	if rules.Validate(d, "en16931").HasErrors() {
		t.Errorf("le document réparé reste non conforme : %+v", rules.Validate(d, "en16931").Findings)
	}
}

func TestRepairTrimsIdentifiers(t *testing.T) {
	d := model.NewDocument(model.KindInvoice)
	d.CurrencyCode = "EUR"
	d.Seller.VATID = "FR 12 501234567 "
	changes := repair.Repair(d, repair.Options{Fixes: []repair.Fix{repair.FixTrimIdentifiers}})
	if d.Seller.VATID != "FR12501234567" {
		t.Errorf("TVA non nettoyée : %q", d.Seller.VATID)
	}
	if len(changes) != 1 || changes[0].Fix != string(repair.FixTrimIdentifiers) {
		t.Errorf("changement non journalisé correctement : %+v", changes)
	}
}

func TestRepairNeverInventsData(t *testing.T) {
	// Un SIREN manquant ne doit jamais être fabriqué (§8.5 MUST).
	d := model.NewDocument(model.KindInvoice)
	d.CurrencyCode = "EUR"
	d.Seller.Name = "SAS Sans Siren"
	repair.Repair(d, repair.Options{})
	if d.Seller.LegalID != "" {
		t.Errorf("repair a inventé un identifiant légal : %q", d.Seller.LegalID)
	}
}

func TestRepairIdempotent(t *testing.T) {
	d, _ := generate.Generate(1, generate.Options{Scenario: generate.ScenarioMultiTVA, Seed: 3})
	repair.Repair(d, repair.Options{})
	// Un document déjà conforme ne doit plus produire de changement.
	changes := repair.Repair(d, repair.Options{})
	if len(changes) != 0 {
		t.Errorf("repair sur document sain devrait être sans effet : %+v", changes)
	}
}
