package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestBRCO21ReasonOrCode(t *testing.T) {
	// Remise de niveau document sans motif ni code → BR-CO-21.
	d := docWithAllowances()
	d.AllowanceCharges[0].Reason = ""
	d.AllowanceCharges[0].ReasonCode = ""
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-CO-21"] {
		t.Error("BR-CO-21 attendu (remise document sans motif)")
	}
}

func TestDecTotalsAndBases(t *testing.T) {
	// Total des remises (BT-107) à 3 décimales → BR-DEC-10.
	d := docWithAllowances()
	a := model.NewAmount(10000, 3, "EUR")
	d.Totals.AllowanceTotal = &a
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-DEC-10"] {
		t.Error("BR-DEC-10 attendu (total remises à 3 décimales)")
	}
	// Base d'une remise document (BT-93) à 3 décimales → BR-DEC-02.
	d2 := docWithAllowances()
	b := model.NewAmount(100000, 3, "EUR")
	d2.AllowanceCharges[0].BaseAmount = &b
	if !findingsByRule(rules.Validate(d2, "en16931"))["BR-DEC-02"] {
		t.Error("BR-DEC-02 attendu (base de remise à 3 décimales)")
	}
}
