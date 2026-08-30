package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Complément de règles de cohérence (BR-CO-21..24) et de décimales (BR-DEC) sur les bases de
// remise/charge, les totaux et l'arrondi. Sémantique alignée sur le Schematron officiel.

func init() {
	// BR-CO-21..24 : chaque remise/charge (document ou ligne) doit porter un motif ou un code motif.
	rules.Register(docACRule("BR-CO-21", "BT-97", false,
		"Une remise de niveau document doit porter un motif ou un code motif.", acHasReason))
	rules.Register(docACRule("BR-CO-22", "BT-104", true,
		"Une charge de niveau document doit porter un motif ou un code motif.", acHasReason))
	rules.Register(lineACRule("BR-CO-23", "BT-139", false,
		"Une remise de ligne doit porter un motif ou un code motif.", acHasReason))
	rules.Register(lineACRule("BR-CO-24", "BT-144", true,
		"Une charge de ligne doit porter un motif ou un code motif.", acHasReason))

	// BR-DEC sur les totaux de niveau document (au plus 2 décimales) :
	rules.Register(decTotalPtrRule("BR-DEC-10", "BT-107", func(d *model.Document) *model.Amount { return d.Totals.AllowanceTotal }))
	rules.Register(decTotalPtrRule("BR-DEC-11", "BT-108", func(d *model.Document) *model.Amount { return d.Totals.ChargeTotal }))
	rules.Register(decTotalPtrRule("BR-DEC-16", "BT-113", func(d *model.Document) *model.Amount { return d.Totals.PrepaidAmount }))
	rules.Register(decTotalPtrRule("BR-DEC-17", "BT-114", func(d *model.Document) *model.Amount { return d.Totals.RoundingAmount }))

	// BR-DEC sur les bases de remise/charge (BT-93/100 document, BT-137/142 ligne) :
	rules.Register(decDocACBaseRule("BR-DEC-02", "BT-93", false))
	rules.Register(decDocACBaseRule("BR-DEC-06", "BT-100", true))
	rules.Register(decLineACBaseRule("BR-DEC-25", "BT-137", false))
	rules.Register(decLineACBaseRule("BR-DEC-28", "BT-142", true))
}

func decTotalPtrRule(id, term string, get func(*model.Document) *model.Amount) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "Un montant total ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if a := get(d); a != nil && a.Scale > 2 {
				return []rules.Finding{{
					RuleID: id, Severity: rules.SeverityError, Term: term,
					Message: fmt.Sprintf("Montant à %d décimales (max 2).", a.Scale),
					Actual:  a.String(), Fixable: true,
				}}
			}
			return nil
		},
	}
}

func decDocACBaseRule(id, term string, wantCharge bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "La base d'une remise/charge ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ac := range d.AllowanceCharges {
				if ac.IsCharge != wantCharge || ac.BaseAmount == nil || ac.BaseAmount.Scale <= 2 {
					continue
				}
				out = append(out, rules.Finding{
					RuleID: id, Severity: rules.SeverityError, Term: term,
					Message: fmt.Sprintf("Base à %d décimales (max 2).", ac.BaseAmount.Scale),
					Path:    fmt.Sprintf("allowanceCharges[%d].baseAmount", i), Actual: ac.BaseAmount.String(), Fixable: true,
				})
			}
			return out
		},
	}
}

func decLineACBaseRule(id, term string, wantCharge bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "La base d'une remise/charge de ligne ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				for j, ac := range l.AllowanceCharges {
					if ac.IsCharge != wantCharge || ac.BaseAmount == nil || ac.BaseAmount.Scale <= 2 {
						continue
					}
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Base à %d décimales (max 2) sur la ligne %s.", ac.BaseAmount.Scale, lineLabel(l, i)),
						Path:    fmt.Sprintf("lines[%d].allowanceCharges[%d].baseAmount", i, j), Actual: ac.BaseAmount.String(), Fixable: true,
					})
				}
			}
			return out
		},
	}
}
