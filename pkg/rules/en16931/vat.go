package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de TVA au taux normal (BR-S-*).

func init() {
	rules.Register(brS01())
	rules.Register(brS05())
}

// brS01 : si une ligne porte la catégorie « taux normal » (S), la ventilation de TVA doit
// contenir au moins une entrée de catégorie S.
func brS01() rules.Rule {
	return rules.Rule{
		ID: "BR-S-01", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-151", "BT-118"},
		Message: map[string]string{"fr": "Une ligne au taux normal impose une ventilation de TVA de catégorie « S »."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			lineHasS := false
			for _, l := range d.Lines {
				if l.TaxCategory == model.TaxStandard {
					lineHasS = true
					break
				}
			}
			if !lineHasS {
				return nil
			}
			for _, ts := range d.TaxBreakdown {
				if ts.Category == model.TaxStandard {
					return nil
				}
			}
			return []rules.Finding{{
				RuleID: "BR-S-01", Severity: rules.SeverityError, Term: "BT-118",
				Message: "Une ligne est au taux normal mais aucune ventilation de TVA de catégorie « S » n'est présente.",
			}}
		},
	}
}

// brS05 : sur une ligne au taux normal, le taux de TVA (BT-152) doit être strictement positif.
func brS05() rules.Rule {
	return rules.Rule{
		ID: "BR-S-05", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-152"},
		Message: map[string]string{"fr": "Le taux de TVA d'une ligne au taux normal doit être supérieur à zéro."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				if l.TaxCategory != model.TaxStandard {
					continue
				}
				if l.TaxRate == nil || l.TaxRate.Unscaled <= 0 {
					out = append(out, rules.Finding{
						RuleID: "BR-S-05", Severity: rules.SeverityError, Term: "BT-152",
						Message: fmt.Sprintf("La ligne %s est au taux normal mais son taux de TVA n'est pas supérieur à zéro.", l.ID),
						Path:    fmt.Sprintf("lines[%d].taxRate", i),
					})
				}
			}
			return out
		},
	}
}
