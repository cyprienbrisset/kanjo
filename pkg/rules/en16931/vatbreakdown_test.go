package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestVATBreakdownAmountRules(t *testing.T) {
	// Cas conforme : la ventilation du document valide porte base et montant.
	if rep := rules.Validate(validDoc(), "en16931"); rep.HasErrors() {
		t.Fatalf("document conforme rejeté : %+v", rep.Findings)
	}

	// BR-45 : ventilation sans montant imposable.
	d := validDoc()
	d.TaxBreakdown[0].TaxableAmount = model.Amount{}
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-45"] {
		t.Error("BR-45 attendu (montant imposable absent)")
	}

	// BR-46 : ventilation sans montant de TVA.
	d2 := validDoc()
	d2.TaxBreakdown[0].TaxAmount = model.Amount{}
	if !findingsByRule(rules.Validate(d2, "en16931"))["BR-46"] {
		t.Error("BR-46 attendu (montant de TVA absent)")
	}
}
