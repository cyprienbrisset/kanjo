package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestRegionalFamilies(t *testing.T) {
	// Ligne IGIC sans ventilation correspondante → BR-AF-01.
	d := validDoc()
	rate := model.MustParseDecimal("7")
	d.Lines[0].TaxCategory = model.TaxCanaryIGIC
	d.Lines[0].TaxRate = &rate
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-AF-01"] {
		t.Error("BR-AF-01 attendu (IGIC sans ventilation)")
	}
	// Ligne IPSI à taux zéro → BR-AG-05 (le taux doit être positif).
	d2 := validDoc()
	zero := model.MustParseDecimal("0")
	d2.Lines[0].TaxCategory = model.TaxCeutaMelillaIPSI
	d2.Lines[0].TaxRate = &zero
	if !findingsByRule(rules.Validate(d2, "en16931"))["BR-AG-05"] {
		t.Error("BR-AG-05 attendu (IPSI à taux zéro)")
	}
}
