package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles décimales (BR-DEC-*) : les montants monétaires ont au maximum deux décimales.

func init() {
	rules.Register(decRule("BR-DEC-12", "BT-106", func(d *model.Document) model.Amount { return d.Totals.LineExtensionAmount }))
	rules.Register(decRule("BR-DEC-13", "BT-110", func(d *model.Document) model.Amount { return d.Totals.TaxAmount }))
	rules.Register(decRule("BR-DEC-14", "BT-109", func(d *model.Document) model.Amount { return d.Totals.TaxExclusiveAmount }))
	rules.Register(decRule("BR-DEC-16", "BT-112", func(d *model.Document) model.Amount { return d.Totals.TaxInclusiveAmount }))
	rules.Register(decRule("BR-DEC-19", "BT-115", func(d *model.Document) model.Amount { return d.Totals.DuePayableAmount }))
	rules.Register(decBreakdownRule())
}

// decRule vérifie qu'un montant de totaux n'a pas plus de deux décimales.
func decRule(id, term string, get func(*model.Document) model.Amount) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{term},
		Message: map[string]string{"fr": "Un montant monétaire ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			a := get(d)
			if a.Scale > 2 {
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

// decBreakdownRule vérifie les décimales des montants de la ventilation de TVA (BT-116/117).
func decBreakdownRule() rules.Rule {
	return rules.Rule{
		ID: "BR-DEC-23", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-116", "BT-117"},
		Message: map[string]string{"fr": "Les montants de TVA (base et montant) ont au maximum deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.TaxableAmount.Scale > 2 {
					out = append(out, rules.Finding{RuleID: "BR-DEC-23", Severity: rules.SeverityError, Term: "BT-116",
						Message: "Base de TVA à plus de deux décimales.", Path: fmt.Sprintf("taxBreakdown[%d].taxableAmount", i), Fixable: true})
				}
				if ts.TaxAmount.Scale > 2 {
					out = append(out, rules.Finding{RuleID: "BR-DEC-23", Severity: rules.SeverityError, Term: "BT-117",
						Message: "Montant de TVA à plus de deux décimales.", Path: fmt.Sprintf("taxBreakdown[%d].taxAmount", i), Fixable: true})
				}
			}
			return out
		},
	}
}
