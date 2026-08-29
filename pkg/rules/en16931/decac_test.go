package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestAllowanceChargeDecimalRules(t *testing.T) {
	// Niveau document : montant de remise à 3 décimales → BR-DEC-01.
	d := docWithAllowances()
	d.AllowanceCharges[0].Amount = model.MustParseAmount("10.000", "EUR")
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-DEC-01"] {
		t.Error("BR-DEC-01 attendu (remise document à 3 décimales)")
	}
	// Niveau document : montant de charge à 3 décimales → BR-DEC-05.
	d2 := docWithAllowances()
	d2.AllowanceCharges[1].Amount = model.MustParseAmount("4.000", "EUR")
	if !findingsByRule(rules.Validate(d2, "en16931"))["BR-DEC-05"] {
		t.Error("BR-DEC-05 attendu (charge document à 3 décimales)")
	}

	// Niveau ligne : remise et charge à 3 décimales → BR-DEC-24 / BR-DEC-27.
	d3 := validDoc()
	d3.Lines[0].AllowanceCharges = []model.AllowanceCharge{
		{IsCharge: false, Amount: model.MustParseAmount("5.000", "EUR"), ReasonCode: "95", Reason: "Remise"},
		{IsCharge: true, Amount: model.MustParseAmount("2.000", "EUR"), ReasonCode: "FC", Reason: "Frais"},
	}
	byRule := findingsByRule(rules.Validate(d3, "en16931"))
	if !byRule["BR-DEC-24"] {
		t.Error("BR-DEC-24 attendu (remise de ligne à 3 décimales)")
	}
	if !byRule["BR-DEC-27"] {
		t.Error("BR-DEC-27 attendu (charge de ligne à 3 décimales)")
	}

	// Montants à 2 décimales → aucune de ces règles.
	clean := findingsByRule(rules.Validate(docWithAllowances(), "en16931"))
	for _, id := range []string{"BR-DEC-01", "BR-DEC-05"} {
		if clean[id] {
			t.Errorf("%s ne devrait pas se déclencher sur des montants à 2 décimales", id)
		}
	}
}
