package fr_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/cius/fr"
)

func frDoc() *model.Document {
	d := model.NewDocument(model.KindInvoice)
	d.ID = "F1"
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{
		Name: "SAS Martin", VATID: "FR12501234567", LegalID: "501234567",
		Address: model.Address{CountryCode: "FR"},
	}
	d.Buyer = model.Party{Name: "Société Cliente", Address: model.Address{CountryCode: "FR"}}
	return d
}

func has(rep rules.Report, id string) bool {
	for _, f := range rep.Findings {
		if f.RuleID == id {
			return true
		}
	}
	return false
}

func TestFRSellerIdentifiedValid(t *testing.T) {
	rep := rules.Validate(frDoc(), "cius.fr")
	if rep.HasErrors() {
		t.Fatalf("document FR valide déclenche des anomalies : %+v", rep.Findings)
	}
}

func TestFRSellerNotIdentifiedFails(t *testing.T) {
	d := frDoc()
	d.Seller.VATID = ""
	d.Seller.LegalID = ""
	d.Extensions.FR = nil
	rep := rules.Validate(d, "cius.fr")
	if !has(rep, "FR-CTC-01") {
		t.Errorf("FR-CTC-01 attendu. Findings: %+v", rep.Findings)
	}
}

func TestFRSirenFormatFails(t *testing.T) {
	d := frDoc()
	d.Seller.LegalID = ""
	d.Extensions.FR = &model.FrenchCTC{SellerSIREN: "1234"} // trop court
	rep := rules.Validate(d, "cius.fr")
	if !has(rep, "FR-SIREN-01") {
		t.Errorf("FR-SIREN-01 attendu. Findings: %+v", rep.Findings)
	}
}

func TestFRSirenDerivedFromSIRET(t *testing.T) {
	d := frDoc()
	d.Seller.VATID = ""
	d.Seller.LegalID = "50123456700012" // SIRET 14 chiffres → SIREN 9 valides
	rep := rules.Validate(d, "cius.fr")
	if rep.HasErrors() {
		t.Errorf("SIRET valide ne devrait pas déclencher d'anomalie : %+v", rep.Findings)
	}
}
