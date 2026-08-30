package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// hasFinding indique si le rapport contient une anomalie pour la règle donnée.
func hasFinding(rep rules.Report, id string) bool {
	for _, f := range rep.Findings {
		if f.RuleID == id {
			return true
		}
	}
	return false
}

// TestBR22 : une ligne sans quantité (absente ET nulle) échoue ; validDoc passe.
func TestBR22(t *testing.T) {
	if rep := rules.Validate(validDoc(), "en16931"); hasFinding(rep, "BR-22") {
		t.Error("validDoc ne devrait pas déclencher BR-22")
	}
	d := validDoc()
	d.Lines[0].Quantity = model.MustParseDecimal("0")
	d.Lines[0].QuantityPresent = false
	if rep := rules.Validate(d, "en16931"); !hasFinding(rep, "BR-22") {
		t.Error("une ligne sans quantité doit déclencher BR-22")
	}
	// Quantité réellement portée à 0 (cas source explicite) : conforme.
	d2 := validDoc()
	d2.Lines[0].Quantity = model.MustParseDecimal("0")
	d2.Lines[0].QuantityPresent = true
	if rep := rules.Validate(d2, "en16931"); hasFinding(rep, "BR-22") {
		t.Error("une quantité 0 explicitement portée ne doit pas déclencher BR-22")
	}
}

// TestBR48 : une ventilation sans taux (hors O) échoue ; catégorie O exemptée.
func TestBR48(t *testing.T) {
	if rep := rules.Validate(validDoc(), "en16931"); hasFinding(rep, "BR-48") {
		t.Error("validDoc ne devrait pas déclencher BR-48")
	}
	d := validDoc()
	d.TaxBreakdown[0].Rate = model.MustParseDecimal("0")
	d.TaxBreakdown[0].RatePresent = false
	d.TaxBreakdown[0].Category = model.TaxZeroRated
	if rep := rules.Validate(d, "en16931"); !hasFinding(rep, "BR-48") {
		t.Error("une ventilation à taux nul non porté doit déclencher BR-48")
	}
	// Catégorie hors champ (O) : exemptée de BR-48.
	d2 := validDoc()
	d2.TaxBreakdown[0].Rate = model.MustParseDecimal("0")
	d2.TaxBreakdown[0].RatePresent = false
	d2.TaxBreakdown[0].Category = model.TaxOutsideScope
	if rep := rules.Validate(d2, "en16931"); hasFinding(rep, "BR-48") {
		t.Error("la catégorie O ne doit pas déclencher BR-48")
	}
}

// TestBR65 : une classification d'article sans schéma échoue.
func TestBR65(t *testing.T) {
	if rep := rules.Validate(validDoc(), "en16931"); hasFinding(rep, "BR-65") {
		t.Error("validDoc (sans classification) ne devrait pas déclencher BR-65")
	}
	d := validDoc()
	d.Lines[0].ClassificationID = "65141904"
	d.Lines[0].ClassificationScheme = ""
	if rep := rules.Validate(d, "en16931"); !hasFinding(rep, "BR-65") {
		t.Error("une classification sans schéma doit déclencher BR-65")
	}
	// Avec schéma : conforme.
	d2 := validDoc()
	d2.Lines[0].ClassificationID = "65141904"
	d2.Lines[0].ClassificationScheme = "STI"
	if rep := rules.Validate(d2, "en16931"); hasFinding(rep, "BR-65") {
		t.Error("une classification avec schéma ne doit pas déclencher BR-65")
	}
}
