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

// TestRepairRecomputeIncludesDocumentCharges reproduit le cas réel (facture 834514) : une
// charge document taxée (éco-contribution) doit être intégrée à la base taxable de son taux et
// au HT (BR-CO-13). Avant correctif, recompute-totals l'ignorait et faisait perdre son montant
// (HT/TVA/TTC recalculés en dessous du TTC réellement dû par le client).
func TestRepairRecomputeIncludesDocumentCharges(t *testing.T) {
	rate := model.MustParseDecimal("20")
	d := model.NewDocument(model.KindInvoice)
	d.CurrencyCode = "EUR"
	d.Lines = []model.Line{
		{
			ID: "1", Name: "Article", Quantity: model.DecimalFromInt(1), UnitCode: model.UnitPiece,
			NetPrice: model.MustParseAmount("6225.05", "EUR"), TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("6225.05", "EUR"),
		},
	}
	charge := model.MustParseAmount("43.62", "EUR")
	d.AllowanceCharges = []model.AllowanceCharge{
		{IsCharge: true, Amount: charge, Reason: "Éco-contribution", TaxCategory: model.TaxStandard, TaxRate: &rate},
	}
	// Totaux volontairement faux pour vérifier que repair les corrige correctement.
	d.Totals.LineExtensionAmount = model.MustParseAmount("0", "EUR")
	d.Totals.TaxExclusiveAmount = model.MustParseAmount("0", "EUR")
	d.Totals.TaxAmount = model.MustParseAmount("0", "EUR")
	d.Totals.TaxInclusiveAmount = model.MustParseAmount("0", "EUR")
	d.Totals.DuePayableAmount = model.MustParseAmount("0", "EUR")

	repair.Repair(d, repair.Options{Fixes: []repair.Fix{repair.FixRecomputeTotals}})

	if got := d.Totals.TaxExclusiveAmount.String(); got != "6268.67" {
		t.Errorf("HT attendu 6268.67 (ligne + charge), obtenu %s", got)
	}
	if got := d.Totals.TaxAmount.String(); got != "1253.73" {
		t.Errorf("TVA attendue 1253.73 (base ligne + charge à 20%%), obtenue %s", got)
	}
	if got := d.Totals.TaxInclusiveAmount.String(); got != "7522.40" {
		t.Errorf("TTC attendu 7522.40, obtenu %s", got)
	}
	if d.Totals.ChargeTotal == nil || d.Totals.ChargeTotal.String() != "43.62" {
		t.Errorf("chargeTotal attendu 43.62, obtenu %v", d.Totals.ChargeTotal)
	}
}

func TestRepairAddsFRMandatoryNotes(t *testing.T) {
	d := model.NewDocument(model.KindInvoice)
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{Name: "SAS Martin", VATID: "FR12501234567", Address: model.Address{CountryCode: "FR"}}
	changes := repair.Repair(d, repair.Options{Fixes: []repair.Fix{repair.FixFRMandatoryNotes}})
	if len(changes) != 3 {
		t.Fatalf("attendu 3 mentions ajoutées (PMT/PMD/AAB), obtenu %d : %+v", len(changes), changes)
	}
	got := map[string]bool{}
	for _, n := range d.Notes {
		got[n.SubjectCode] = true
	}
	for _, code := range []string{"PMT", "PMD", "AAB"} {
		if !got[code] {
			t.Errorf("mention %s absente après repair", code)
		}
	}
}

func TestRepairFRMandatoryNotesOnlyAddsMissing(t *testing.T) {
	d := model.NewDocument(model.KindInvoice)
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{Name: "SAS Martin", VATID: "FR12501234567", Address: model.Address{CountryCode: "FR"}}
	d.Notes = []model.Note{{Content: "Déjà présente.", SubjectCode: "PMT"}}
	changes := repair.Repair(d, repair.Options{Fixes: []repair.Fix{repair.FixFRMandatoryNotes}})
	if len(changes) != 2 {
		t.Fatalf("attendu 2 mentions ajoutées (PMD/AAB), obtenu %d : %+v", len(changes), changes)
	}
	if d.Notes[0].Content != "Déjà présente." {
		t.Errorf("la mention PMT existante a été modifiée : %+v", d.Notes[0])
	}
}

func TestRepairFRMandatoryNotesSkipsNonFR(t *testing.T) {
	d := model.NewDocument(model.KindInvoice)
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{Name: "GmbH Berlin", Address: model.Address{CountryCode: "DE"}}
	changes := repair.Repair(d, repair.Options{Fixes: []repair.Fix{repair.FixFRMandatoryNotes}})
	if len(changes) != 0 {
		t.Errorf("un vendeur non FR ne devrait pas recevoir ces mentions : %+v", changes)
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
