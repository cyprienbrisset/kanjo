package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestLineRulesFail(t *testing.T) {
	neg := model.NewAmount(-100, 2, "EUR")
	cases := []struct {
		rule   string
		mutate func(*model.Document)
	}{
		{"BR-21", func(d *model.Document) { d.Lines[0].ID = "" }},
		{"BR-23", func(d *model.Document) { d.Lines[0].UnitCode = "" }},
		{"BR-25", func(d *model.Document) { d.Lines[0].Name = "" }},
		{"BR-26", func(d *model.Document) { d.Lines[0].TaxCategory = "" }},
		{"BR-27", func(d *model.Document) { d.Lines[0].NetPrice = neg }},
	}
	for _, c := range cases {
		d := validDoc()
		c.mutate(d)
		if !findingsByRule(rules.Validate(d, "en16931"))[c.rule] {
			t.Errorf("%s : attendait une anomalie", c.rule)
		}
	}
}

func TestVATCategoryRulesFail(t *testing.T) {
	twenty := model.MustParseDecimal("20")
	cases := []struct {
		rule   string
		mutate func(*model.Document)
	}{
		{"BR-Z-05", func(d *model.Document) {
			d.TaxBreakdown[0].Category = model.TaxZeroRated
			d.TaxBreakdown[0].Rate = twenty
		}},
		{"BR-E-10", func(d *model.Document) {
			d.TaxBreakdown[0].Category = model.TaxExempt
			d.TaxBreakdown[0].Rate = model.MustParseDecimal("0")
			d.TaxBreakdown[0].ExemptionReason = ""
		}},
		{"BR-AE-10", func(d *model.Document) {
			d.TaxBreakdown[0].Category = model.TaxReverseCharge
			d.TaxBreakdown[0].Rate = model.MustParseDecimal("0")
		}},
		{"BR-IC-10", func(d *model.Document) {
			d.TaxBreakdown[0].Category = model.TaxIntraCommunity
			d.TaxBreakdown[0].Rate = model.MustParseDecimal("0")
		}},
		{"BR-AE-05", func(d *model.Document) {
			d.TaxBreakdown[0].Category = model.TaxReverseCharge // taux 20 conservé → non nul
		}},
	}
	for _, c := range cases {
		d := validDoc()
		c.mutate(d)
		if !findingsByRule(rules.Validate(d, "en16931"))[c.rule] {
			t.Errorf("%s : attendait une anomalie. Findings: %+v", c.rule, rules.Validate(d, "en16931").Findings)
		}
	}
}

func TestDecimalRulesFail(t *testing.T) {
	d := validDoc()
	d.Totals.TaxInclusiveAmount = model.NewAmount(149988, 3, "EUR") // 3 décimales
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-DEC-14"] {
		t.Error("BR-DEC-14 attendu pour un montant à 3 décimales")
	}
}

func TestCO2RulesFail(t *testing.T) {
	cases := []struct {
		rule   string
		mutate func(*model.Document)
	}{
		{"BR-CO-09", func(d *model.Document) { d.Seller.VATID = "12501234567" }}, // pas de préfixe pays
		{"BR-CO-18", func(d *model.Document) { d.TaxBreakdown = nil }},
		{"BR-CO-25", func(d *model.Document) { d.DueDate = nil; d.PaymentTerms = "" }},
	}
	for _, c := range cases {
		d := validDoc()
		c.mutate(d)
		if !findingsByRule(rules.Validate(d, "en16931"))[c.rule] {
			t.Errorf("%s : attendait une anomalie", c.rule)
		}
	}
}
