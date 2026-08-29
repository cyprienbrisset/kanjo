package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestRules2Fail(t *testing.T) {
	cases := []struct {
		rule   string
		mutate func(*model.Document)
	}{
		{"BR-08", func(d *model.Document) { d.Seller.Address = model.Address{} }},
		{"BR-10", func(d *model.Document) { d.Buyer.Address = model.Address{} }},
		{"BR-CO-26", func(d *model.Document) { d.Seller.VATID = ""; d.Seller.LegalID = ""; d.Seller.ID = "" }},
		{"BR-47", func(d *model.Document) { d.TaxBreakdown[0].Category = "" }},
		{"BR-DEC-24", func(d *model.Document) { d.Lines[0].NetAmount = model.NewAmount(100000, 3, "EUR") }},
	}
	for _, c := range cases {
		d := validDoc()
		c.mutate(d)
		if !findingsByRule(rules.Validate(d, "en16931"))[c.rule] {
			t.Errorf("%s : attendait une anomalie", c.rule)
		}
	}
}

func TestValidDocStillPassesWithNewRules(t *testing.T) {
	// validDoc doit rester conforme malgré l'ajout des nouvelles règles.
	if rules.Validate(validDoc(), "en16931").HasErrors() {
		t.Errorf("validDoc n'est plus conforme : %+v", rules.Validate(validDoc(), "en16931").Findings)
	}
}
