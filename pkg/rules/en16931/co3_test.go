package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestBRCO11And12(t *testing.T) {
	// Document avec une remise de 10 et une charge de 4, totaux cohérents.
	mk := func() *model.Document {
		d := validDoc()
		d.AllowanceCharges = []model.AllowanceCharge{
			{IsCharge: false, Amount: model.MustParseAmount("10.00", "EUR")},
			{IsCharge: true, Amount: model.MustParseAmount("4.00", "EUR")},
		}
		al := model.MustParseAmount("10.00", "EUR")
		ch := model.MustParseAmount("4.00", "EUR")
		d.Totals.AllowanceTotal = &al
		d.Totals.ChargeTotal = &ch
		// HT = lignes(1249.90) - 10 + 4 = 1243.90 ; ajuster pour rester cohérent BR-CO-13/15.
		d.Totals.TaxExclusiveAmount = model.MustParseAmount("1243.90", "EUR")
		d.TaxBreakdown[0].TaxableAmount = model.MustParseAmount("1243.90", "EUR")
		d.TaxBreakdown[0].TaxAmount = model.MustParseAmount("248.78", "EUR")
		d.Totals.TaxAmount = model.MustParseAmount("248.78", "EUR")
		d.Totals.TaxInclusiveAmount = model.MustParseAmount("1492.68", "EUR")
		d.Totals.DuePayableAmount = model.MustParseAmount("1492.68", "EUR")
		return d
	}
	// Cas conforme.
	if rules.Validate(mk(), "en16931").HasErrors() {
		t.Fatalf("document avec remises/charges cohérentes ne devrait pas échouer : %+v", rules.Validate(mk(), "en16931").Findings)
	}
	// BR-CO-11 : total remises faux.
	d := mk()
	wrong := model.MustParseAmount("99.00", "EUR")
	d.Totals.AllowanceTotal = &wrong
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-CO-11"] {
		t.Error("BR-CO-11 attendu")
	}
	// BR-CO-12 : total charges faux.
	d2 := mk()
	d2.Totals.ChargeTotal = &wrong
	if !findingsByRule(rules.Validate(d2, "en16931"))["BR-CO-12"] {
		t.Error("BR-CO-12 attendu")
	}
}
