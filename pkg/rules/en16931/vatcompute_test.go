package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

// zeroRatedDoc renvoie un document valide entièrement à taux zéro (Z), cohérent.
func zeroRatedDoc() *model.Document {
	d := validDoc()
	zero := model.MustParseDecimal("0")
	for i := range d.Lines {
		d.Lines[i].TaxCategory = model.TaxZeroRated
		d.Lines[i].TaxRate = &zero
	}
	d.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxZeroRated, Rate: zero,
		TaxableAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:     model.MustParseAmount("0.00", "EUR"),
	}}
	d.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:           model.MustParseAmount("0.00", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("1249.90", "EUR"),
		DuePayableAmount:    model.MustParseAmount("1249.90", "EUR"),
	}
	return d
}

func TestBreakdownZeroTax(t *testing.T) {
	// Ventilation à taux zéro avec un montant de TVA non nul → BR-Z-09.
	d := zeroRatedDoc()
	d.TaxBreakdown[0].TaxAmount = model.MustParseAmount("5.00", "EUR")
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-Z-09"] {
		t.Error("BR-Z-09 attendu (TVA non nulle sur ventilation Z)")
	}
}

func TestBreakdownTaxableSum(t *testing.T) {
	// Base d'imposition incohérente avec la somme des lignes → BR-Z-08.
	d := zeroRatedDoc()
	d.TaxBreakdown[0].TaxableAmount = model.MustParseAmount("999.99", "EUR")
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-Z-08"] {
		t.Error("BR-Z-08 attendu (base ≠ somme des lignes)")
	}
}

func TestBreakdownZeroNoExemptionReason(t *testing.T) {
	// Ventilation à taux zéro avec motif d'exonération indu → BR-Z-10.
	d := zeroRatedDoc()
	d.TaxBreakdown[0].ExemptionReason = "motif indu"
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-Z-10"] {
		t.Error("BR-Z-10 attendu (motif d'exonération sur ventilation Z)")
	}
}

func TestBreakdownStandardTax(t *testing.T) {
	// Montant de TVA au taux normal incohérent avec base × taux → BR-S-09.
	d := validDoc()
	d.TaxBreakdown[0].TaxAmount = model.MustParseAmount("1.00", "EUR")
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-S-09"] {
		t.Error("BR-S-09 attendu (TVA ≠ base × taux)")
	}
}
