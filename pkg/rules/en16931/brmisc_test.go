package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestMiscRules(t *testing.T) {
	cases := []struct {
		rule string
		mut  func(*model.Document)
	}{
		{"BR-17", func(d *model.Document) { d.Payee = &model.Party{} }},
		{"BR-62", func(d *model.Document) { d.Seller.ElectronicAddr = &model.ElectronicAddress{Value: "x@y.z"} }},
		{"BR-63", func(d *model.Document) { d.Buyer.ElectronicAddr = &model.ElectronicAddress{Value: "a@b.c"} }},
		{"BR-64", func(d *model.Document) { d.Lines[0].StandardID = "GTIN123"; d.Lines[0].StandardScheme = "" }},
		{"BR-61", func(d *model.Document) {
			d.PaymentInstructions = &model.PaymentInstructions{MeansCode: model.PayCredit}
		}},
		{"BR-IC-12", func(d *model.Document) { d.TaxBreakdown[0].Category = model.TaxIntraCommunity; d.DeliverTo = nil }},
	}
	for _, c := range cases {
		d := validDoc()
		c.mut(d)
		if !findingsByRule(rules.Validate(d, "en16931"))[c.rule] {
			t.Errorf("%s attendu", c.rule)
		}
	}
}
