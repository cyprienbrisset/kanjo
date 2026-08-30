package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestFinalBatch(t *testing.T) {
	cases := []struct {
		rule string
		mut  func(*model.Document)
	}{
		{"BR-52", func(d *model.Document) { d.Attachments = []model.Attachment{{Description: "x"}} }},
		{"BR-53", func(d *model.Document) { d.TaxCurrencyCode = "USD" }},
		{"BR-CL-05", func(d *model.Document) {
			d.TaxCurrencyCode = "US"
			a := model.MustParseAmount("1.00", "USD")
			d.Totals.TaxAmountInAccountingCurrency = &a
		}},
		{"BR-18", func(d *model.Document) {
			d.TaxRep = &model.Party{Address: model.Address{CountryCode: "FR"}, VATID: "FR1"}
		}},
		{"BR-56", func(d *model.Document) {
			d.TaxRep = &model.Party{Name: "Rep", Address: model.Address{CountryCode: "FR"}}
		}},
		{"BR-49", func(d *model.Document) { d.PaymentInstructions = &model.PaymentInstructions{RemittanceInfo: "x"} }},
		{"BR-CL-16", func(d *model.Document) {
			d.PaymentInstructions = &model.PaymentInstructions{MeansCode: model.PaymentMeansCode("999")}
		}},
		{"BR-CL-19", func(d *model.Document) {
			r := model.MustParseDecimal("20")
			d.AllowanceCharges = []model.AllowanceCharge{{IsCharge: false, Amount: model.MustParseAmount("1.00", "EUR"),
				TaxCategory: model.TaxStandard, TaxRate: &r, ReasonCode: "ZZ", Reason: "x"}}
		}},
		{"BR-O-06", func(d *model.Document) {
			r := model.MustParseDecimal("20")
			d.AllowanceCharges = []model.AllowanceCharge{{IsCharge: false, Amount: model.MustParseAmount("1.00", "EUR"),
				TaxCategory: model.TaxOutsideScope, TaxRate: &r, Reason: "x"}}
		}},
	}
	for _, c := range cases {
		d := validDoc()
		c.mut(d)
		if !findingsByRule(rules.Validate(d, "en16931"))[c.rule] {
			t.Errorf("%s attendu", c.rule)
		}
	}
}
