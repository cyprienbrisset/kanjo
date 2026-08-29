package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de présence et de validité au niveau ligne (BR-21 à BR-28, BR-CO-04).
// Les identifiants suivent la sémantique EXACTE du Schematron officiel EN 16931.

func init() {
	rules.Register(lineRule("BR-21", "BT-126", "Chaque ligne doit avoir un identifiant.",
		func(l model.Line) bool { return l.ID != "" }))
	rules.Register(lineRule("BR-23", "BT-130", "Chaque ligne doit indiquer une unité de mesure.",
		func(l model.Line) bool { return l.UnitCode != "" }))
	rules.Register(lineRule("BR-24", "BT-131", "Chaque ligne doit porter un montant net.",
		func(l model.Line) bool { return l.NetAmount.Currency != "" }))
	rules.Register(lineRule("BR-25", "BT-153", "Chaque ligne doit désigner un article (nom).",
		func(l model.Line) bool { return l.Name != "" }))
	rules.Register(lineRule("BR-26", "BT-146", "Chaque ligne doit porter un prix net d'article.",
		func(l model.Line) bool { return l.NetPrice.Currency != "" }))
	rules.Register(lineRule("BR-27", "BT-146", "Le prix net d'une ligne ne doit pas être négatif.",
		func(l model.Line) bool { return l.NetPrice.Value >= 0 }))
	rules.Register(lineRule("BR-28", "BT-148", "Le prix brut d'une ligne ne doit pas être négatif.",
		func(l model.Line) bool { return l.GrossPrice == nil || l.GrossPrice.Value >= 0 }))
	// BR-CO-04 : chaque ligne doit être catégorisée par un code de catégorie de TVA (BT-151).
	rules.Register(lineRule("BR-CO-04", "BT-151", "Chaque ligne doit porter une catégorie de TVA.",
		func(l model.Line) bool { return l.TaxCategory != "" }))
}

// lineRule construit une règle appliquée à chaque ligne : elle émet une anomalie pour toute
// ligne dont le prédicat est faux.
func lineRule(id, term, msgFR string, ok func(model.Line) bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{term},
		Message: map[string]string{"fr": msgFR},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				if !ok(l) {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("%s (ligne %s)", msgFR, lineLabel(l, i)),
						Path:    fmt.Sprintf("lines[%d]", i),
					})
				}
			}
			return out
		},
	}
}

func lineLabel(l model.Line, i int) string {
	if l.ID != "" {
		return l.ID
	}
	return fmt.Sprintf("#%d", i+1)
}
