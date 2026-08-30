package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func oDoc() *model.Document {
	d := validDoc()
	zero := model.MustParseDecimal("0")
	for i := range d.Lines {
		d.Lines[i].TaxCategory = model.TaxOutsideScope
		d.Lines[i].TaxRate = &zero
	}
	d.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxOutsideScope, Rate: zero,
		TaxableAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:     model.MustParseAmount("0.00", "EUR"),
	}}
	return d
}

func TestBROExclusivity(t *testing.T) {
	// Facture hors champ mais avec identifiant TVA vendeur → BR-O-02.
	if !findingsByRule(rules.Validate(oDoc(), "en16931"))["BR-O-02"] {
		t.Error("BR-O-02 attendu (TVA vendeur sur facture hors champ)")
	}
	// Ventilation hors champ mélangée à une autre catégorie → BR-O-11.
	d := oDoc()
	rate := model.MustParseDecimal("20")
	d.TaxBreakdown = append(d.TaxBreakdown, model.TaxSubtotal{Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("10.00", "EUR"), TaxAmount: model.MustParseAmount("2.00", "EUR")})
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-O-11"] {
		t.Error("BR-O-11 attendu (mélange de catégories)")
	}
	// Ligne d'une autre catégorie dans une facture hors champ → BR-O-12.
	d2 := oDoc()
	d2.Lines[0].TaxCategory = model.TaxStandard
	if !findingsByRule(rules.Validate(d2, "en16931"))["BR-O-12"] {
		t.Error("BR-O-12 attendu (ligne non hors champ)")
	}
}
