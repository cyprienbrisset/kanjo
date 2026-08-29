package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de présence des montants d'une ventilation de TVA (BG-23).
//   - BR-45 : montant imposable de la catégorie (BT-116).
//   - BR-46 : montant de TVA de la catégorie (BT-117).
// La présence de la catégorie (BT-118) est déjà couverte par BR-47.

func init() {
	rules.Register(breakdownAmountRule("BR-45", "BT-116",
		"Chaque ventilation de TVA doit porter un montant imposable.",
		func(ts model.TaxSubtotal) bool { return ts.TaxableAmount.Currency != "" }))
	rules.Register(breakdownAmountRule("BR-46", "BT-117",
		"Chaque ventilation de TVA doit porter un montant de TVA.",
		func(ts model.TaxSubtotal) bool { return ts.TaxAmount.Currency != "" }))
}

// breakdownAmountRule émet une anomalie pour toute ventilation dont le prédicat est faux.
func breakdownAmountRule(id, term, msgFR string, ok func(model.TaxSubtotal) bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{term},
		Message: map[string]string{"fr": msgFR},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ok(ts) {
					continue
				}
				out = append(out, rules.Finding{
					RuleID: id, Severity: rules.SeverityError, Term: term,
					Message: fmt.Sprintf("%s (ventilation #%d)", msgFR, i+1),
					Path:    fmt.Sprintf("taxBreakdown[%d]", i),
				})
			}
			return out
		},
	}
}
