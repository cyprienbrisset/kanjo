package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestTotalsPresenceRules(t *testing.T) {
	if rep := rules.Validate(validDoc(), "en16931"); rep.HasErrors() {
		t.Fatalf("document conforme rejeté : %+v", rep.Findings)
	}

	cases := []struct {
		want string
		mut  func(*model.Document)
	}{
		{"BR-12", func(d *model.Document) { d.Totals.LineExtensionAmount = model.Amount{} }},
		{"BR-13", func(d *model.Document) { d.Totals.TaxExclusiveAmount = model.Amount{} }},
		{"BR-14", func(d *model.Document) { d.Totals.TaxInclusiveAmount = model.Amount{} }},
		{"BR-15", func(d *model.Document) { d.Totals.DuePayableAmount = model.Amount{} }},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			d := validDoc()
			c.mut(d)
			if !findingsByRule(rules.Validate(d, "en16931"))[c.want] {
				t.Errorf("%s attendu", c.want)
			}
		})
	}
}
