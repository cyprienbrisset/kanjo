package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestPresenceRulesFail(t *testing.T) {
	cases := []struct {
		rule   string
		mutate func(*model.Document)
	}{
		{"BR-02", func(d *model.Document) { d.ID = "" }},
		{"BR-03", func(d *model.Document) { d.IssueDate = model.Date{} }},
		{"BR-04", func(d *model.Document) { d.TypeCode = "" }},
		{"BR-05", func(d *model.Document) { d.CurrencyCode = "" }},
		{"BR-06", func(d *model.Document) { d.Seller.Name = "" }},
		{"BR-07", func(d *model.Document) { d.Buyer.Name = "" }},
		{"BR-09", func(d *model.Document) { d.Seller.Address.CountryCode = "" }},
		{"BR-11", func(d *model.Document) { d.Buyer.Address.CountryCode = "" }},
		{"BR-16", func(d *model.Document) { d.Lines = nil }},
	}
	for _, c := range cases {
		d := validDoc()
		c.mutate(d)
		rep := rules.Validate(d, "en16931")
		if !findingsByRule(rep)[c.rule] {
			t.Errorf("%s : attendait une anomalie. Findings: %+v", c.rule, rep.Findings)
		}
	}
}

func TestCodeListRulesFail(t *testing.T) {
	cases := []struct {
		rule   string
		mutate func(*model.Document)
	}{
		{"BR-CL-01", func(d *model.Document) { d.TypeCode = "999" }},
		{"BR-CL-04", func(d *model.Document) { d.CurrencyCode = "EURO" }},
		{"BR-CL-14", func(d *model.Document) { d.Seller.Address.CountryCode = "XX" }},
		{"BR-CL-17", func(d *model.Document) { d.TaxBreakdown[0].Category = "Q" }},
	}
	for _, c := range cases {
		d := validDoc()
		c.mutate(d)
		rep := rules.Validate(d, "en16931")
		if !findingsByRule(rep)[c.rule] {
			t.Errorf("%s : attendait une anomalie. Findings: %+v", c.rule, rep.Findings)
		}
	}
}

func TestVATRulesFail(t *testing.T) {
	// BR-S-05 : ligne au taux normal sans taux positif.
	d := validDoc()
	zero := model.MustParseDecimal("0")
	d.Lines[0].TaxRate = &zero
	rep := rules.Validate(d, "en16931")
	if !findingsByRule(rep)["BR-S-05"] {
		t.Errorf("BR-S-05 attendu. Findings: %+v", rep.Findings)
	}

	// BR-S-01 : ligne au taux normal mais aucune ventilation S.
	d2 := validDoc()
	d2.TaxBreakdown[0].Category = model.TaxZeroRated
	rep2 := rules.Validate(d2, "en16931")
	if !findingsByRule(rep2)["BR-S-01"] {
		t.Errorf("BR-S-01 attendu. Findings: %+v", rep2.Findings)
	}
}
