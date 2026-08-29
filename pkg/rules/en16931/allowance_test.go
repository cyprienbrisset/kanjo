package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func ptrDec(s string) *model.Decimal { d := model.MustParseDecimal(s); return &d }

// docWithAllowances renvoie un document valide muni d'une remise et d'une charge de niveau
// document complètes (montant, catégorie, motif), donc conforme aux BR-31..38.
func docWithAllowances() *model.Document {
	d := validDoc()
	d.AllowanceCharges = []model.AllowanceCharge{
		{IsCharge: false, Amount: model.MustParseAmount("10.00", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: ptrDec("20"), ReasonCode: "95", Reason: "Remise"},
		{IsCharge: true, Amount: model.MustParseAmount("4.00", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: ptrDec("20"), ReasonCode: "FC", Reason: "Frais"},
	}
	al := model.MustParseAmount("10.00", "EUR")
	ch := model.MustParseAmount("4.00", "EUR")
	d.Totals.AllowanceTotal = &al
	d.Totals.ChargeTotal = &ch
	d.Totals.TaxExclusiveAmount = model.MustParseAmount("1243.90", "EUR")
	d.TaxBreakdown[0].TaxableAmount = model.MustParseAmount("1243.90", "EUR")
	d.TaxBreakdown[0].TaxAmount = model.MustParseAmount("248.78", "EUR")
	d.Totals.TaxAmount = model.MustParseAmount("248.78", "EUR")
	d.Totals.TaxInclusiveAmount = model.MustParseAmount("1492.68", "EUR")
	d.Totals.DuePayableAmount = model.MustParseAmount("1492.68", "EUR")
	return d
}

func TestDocumentAllowanceChargeRules(t *testing.T) {
	// Cas conforme : aucune anomalie sur les BR de remise/charge.
	if rep := rules.Validate(docWithAllowances(), "en16931"); rep.HasErrors() {
		t.Fatalf("document conforme rejeté : %+v", rep.Findings)
	}

	cases := []struct {
		name string
		want string
		mut  func(*model.Document)
	}{
		{"remise sans catégorie (BR-32)", "BR-32", func(d *model.Document) { d.AllowanceCharges[0].TaxCategory = "" }},
		{"remise sans motif (BR-33)", "BR-33", func(d *model.Document) {
			d.AllowanceCharges[0].Reason = ""
			d.AllowanceCharges[0].ReasonCode = ""
		}},
		{"charge sans catégorie (BR-37)", "BR-37", func(d *model.Document) { d.AllowanceCharges[1].TaxCategory = "" }},
		{"charge sans motif (BR-38)", "BR-38", func(d *model.Document) {
			d.AllowanceCharges[1].Reason = ""
			d.AllowanceCharges[1].ReasonCode = ""
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := docWithAllowances()
			c.mut(d)
			if !findingsByRule(rules.Validate(d, "en16931"))[c.want] {
				t.Errorf("%s attendu", c.want)
			}
		})
	}
}

func TestLineAllowanceChargeRules(t *testing.T) {
	// Ligne conforme : remise et charge complètes.
	mk := func() *model.Document {
		d := validDoc()
		d.Lines[0].AllowanceCharges = []model.AllowanceCharge{
			{IsCharge: false, Amount: model.MustParseAmount("5.00", "EUR"), ReasonCode: "95", Reason: "Remise"},
			{IsCharge: true, Amount: model.MustParseAmount("2.00", "EUR"), ReasonCode: "FC", Reason: "Frais"},
		}
		return d
	}
	if rep := rules.Validate(mk(), "en16931"); rep.HasErrors() {
		t.Fatalf("ligne conforme rejetée : %+v", rep.Findings)
	}

	// BR-42 : remise de ligne sans motif.
	d := mk()
	d.Lines[0].AllowanceCharges[0].Reason = ""
	d.Lines[0].AllowanceCharges[0].ReasonCode = ""
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-42"] {
		t.Error("BR-42 attendu")
	}
	// BR-44 : charge de ligne sans motif.
	d2 := mk()
	d2.Lines[0].AllowanceCharges[1].Reason = ""
	d2.Lines[0].AllowanceCharges[1].ReasonCode = ""
	if !findingsByRule(rules.Validate(d2, "en16931"))["BR-44"] {
		t.Error("BR-44 attendu")
	}
}
