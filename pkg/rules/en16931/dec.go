package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles décimales (BR-DEC-*) : les montants monétaires ont au maximum deux décimales.

func init() {
	rules.Register(decRule("BR-DEC-09", "BT-106", func(d *model.Document) model.Amount { return d.Totals.LineExtensionAmount }))
	rules.Register(decRule("BR-DEC-13", "BT-110", func(d *model.Document) model.Amount { return d.Totals.TaxAmount }))
	rules.Register(decRule("BR-DEC-12", "BT-109", func(d *model.Document) model.Amount { return d.Totals.TaxExclusiveAmount }))
	rules.Register(decRule("BR-DEC-14", "BT-112", func(d *model.Document) model.Amount { return d.Totals.TaxInclusiveAmount }))
	rules.Register(decRule("BR-DEC-18", "BT-115", func(d *model.Document) model.Amount { return d.Totals.DuePayableAmount }))
	rules.Register(decBreakdownRule("BR-DEC-19", "BT-116", func(ts model.TaxSubtotal) model.Amount { return ts.TaxableAmount }))
	rules.Register(decBreakdownRule("BR-DEC-20", "BT-117", func(ts model.TaxSubtotal) model.Amount { return ts.TaxAmount }))
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

// decBreakdownRule vérifie les décimales d'un montant de la ventilation de TVA (BT-116 ou BT-117).
func decBreakdownRule(id, term string, get func(model.TaxSubtotal) model.Amount) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{term},
		Message: map[string]string{"fr": "Un montant de la ventilation de TVA ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if a := get(ts); a.Scale > 2 {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Montant de la ventilation #%d à %d décimales (max 2).", i+1, a.Scale),
						Path:    fmt.Sprintf("taxBreakdown[%d]", i), Actual: a.String(), Fixable: true,
					})
				}
			}
			return out
		},
	}
}
