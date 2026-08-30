package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931" // déclenche l'enregistrement des règles
)

// validDoc construit une facture arithmétiquement cohérente (toutes les BR-CO passent).
func validDoc() *model.Document {
	rate := model.MustParseDecimal("20")
	d := model.NewDocument(model.KindInvoice)
	d.ID = "F1"
	d.TypeCode = model.TypeCommercialInvoice
	d.IssueDate, _ = model.ParseISO("2026-08-12")
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{Name: "SAS Martin", VATID: "FR12501234567", Address: model.Address{CountryCode: "FR"}}
	d.Buyer = model.Party{Name: "Société Cliente", Address: model.Address{CountryCode: "FR"}}
	due, _ := model.ParseISO("2026-09-11")
	d.DueDate = &due
	d.Lines = []model.Line{
		{ID: "1", Name: "Conseil", Quantity: model.DecimalFromInt(2), UnitCode: model.UnitPiece,
			NetPrice:    model.MustParseAmount("500.00", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: &rate, NetAmount: model.MustParseAmount("1000.00", "EUR")},
		{ID: "2", Name: "Licence", Quantity: model.DecimalFromInt(1), UnitCode: model.UnitPiece,
			NetPrice:    model.MustParseAmount("249.90", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: &rate, NetAmount: model.MustParseAmount("249.90", "EUR")},
	}
	d.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:     model.MustParseAmount("249.98", "EUR"),
	}}
	d.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:           model.MustParseAmount("249.98", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("1499.88", "EUR"),
		DuePayableAmount:    model.MustParseAmount("1499.88", "EUR"),
	}
	d.Provenance = model.NewProvenance("", "test", "en16931")
	d.Provenance.SpecIdentifier = "urn:cen.eu:en16931:2017"
	return d
}

func findingsByRule(rep rules.Report) map[string]bool {
	m := map[string]bool{}
	for _, f := range rep.Findings {
		m[f.RuleID] = true
	}
	return m
}

func TestValidDocPassesAllBRCO(t *testing.T) {
	rep := rules.Validate(validDoc(), "en16931")
	if rep.HasErrors() {
		t.Fatalf("le document valide déclenche des anomalies : %+v", rep.Findings)
	}
	if rep.RulesRun < 6 {
		t.Errorf("attendait ≥ 6 règles exécutées, obtenu %d", rep.RulesRun)
	}
}

func TestEachBRCOFails(t *testing.T) {
	cases := []struct {
		rule   string
		mutate func(*model.Document)
	}{
		{"BR-CO-10", func(d *model.Document) { d.Lines[0].NetAmount = model.MustParseAmount("999.00", "EUR") }},
		{"BR-CO-13", func(d *model.Document) { d.Totals.TaxExclusiveAmount = model.MustParseAmount("1250.00", "EUR") }},
		{"BR-CO-14", func(d *model.Document) { d.Totals.TaxAmount = model.MustParseAmount("250.00", "EUR") }},
		{"BR-CO-15", func(d *model.Document) { d.Totals.TaxInclusiveAmount = model.MustParseAmount("1500.00", "EUR") }},
		{"BR-CO-16", func(d *model.Document) { d.Totals.DuePayableAmount = model.MustParseAmount("1400.00", "EUR") }},
		{"BR-CO-17", func(d *model.Document) { d.TaxBreakdown[0].TaxAmount = model.MustParseAmount("240.00", "EUR") }},
	}
	for _, c := range cases {
		d := validDoc()
		c.mutate(d)
		rep := rules.Validate(d, "en16931")
		if !findingsByRule(rep)[c.rule] {
			t.Errorf("%s : attendait une anomalie, aucune détectée. Findings: %+v", c.rule, rep.Findings)
		}
	}
}
