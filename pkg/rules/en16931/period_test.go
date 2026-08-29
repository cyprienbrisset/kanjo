package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func dt(s string) *model.Date { d, _ := model.ParseISO(s); return &d }

func TestPeriodRules(t *testing.T) {
	// BR-CO-19 : période document vide.
	d := validDoc()
	d.Period = &model.Period{}
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-CO-19"] {
		t.Error("BR-CO-19 attendu (période document vide)")
	}
	// Période document valide (une seule date) → pas de BR-CO-19.
	d2 := validDoc()
	d2.Period = &model.Period{Start: dt("2026-01-01")}
	if findingsByRule(rules.Validate(d2, "en16931"))["BR-CO-19"] {
		t.Error("BR-CO-19 ne devrait pas se déclencher avec une date de début")
	}
	// BR-29 : fin avant début.
	d3 := validDoc()
	d3.Period = &model.Period{Start: dt("2026-02-01"), End: dt("2026-01-01")}
	if !findingsByRule(rules.Validate(d3, "en16931"))["BR-29"] {
		t.Error("BR-29 attendu (fin avant début)")
	}
	// Période cohérente → pas de BR-29.
	d4 := validDoc()
	d4.Period = &model.Period{Start: dt("2026-01-01"), End: dt("2026-01-31")}
	if findingsByRule(rules.Validate(d4, "en16931"))["BR-29"] {
		t.Error("BR-29 ne devrait pas se déclencher sur une période cohérente")
	}
}

func TestLinePeriodRules(t *testing.T) {
	// BR-CO-20 : période de ligne vide.
	d := validDoc()
	d.Lines[0].Period = &model.Period{}
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-CO-20"] {
		t.Error("BR-CO-20 attendu (période de ligne vide)")
	}
	// BR-30 : fin avant début sur une ligne.
	d2 := validDoc()
	d2.Lines[0].Period = &model.Period{Start: dt("2026-03-01"), End: dt("2026-02-01")}
	if !findingsByRule(rules.Validate(d2, "en16931"))["BR-30"] {
		t.Error("BR-30 attendu (fin avant début sur ligne)")
	}
}

func TestPrecedingInvoiceRule(t *testing.T) {
	// BR-55 : référence de facture antérieure sans identifiant.
	d := validDoc()
	d.Precedings = []model.Preceding{{IssueDate: dt("2026-01-15")}}
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-55"] {
		t.Error("BR-55 attendu (préc. sans identifiant)")
	}
	// Avec identifiant → conforme.
	d2 := validDoc()
	d2.Precedings = []model.Preceding{{ID: "F2025-0999"}}
	if findingsByRule(rules.Validate(d2, "en16931"))["BR-55"] {
		t.Error("BR-55 ne devrait pas se déclencher avec un identifiant")
	}
}
