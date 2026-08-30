package en16931

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de présence/structure BR-22, BR-48, BR-65. Elles s'appuient sur la distinction
// « absent » vs « zéro » tracée à la lecture (règle 5 du CDC) : une quantité ou un taux réellement
// absents de la source ne sont JAMAIS confondus avec une valeur nulle légitime (§17.7).
func init() {
	// BR-22 : chaque ligne doit porter une quantité (BT-129). On distingue l'absence (source qui
	// n'émet pas l'élément) d'une valeur nulle. Pour les documents construits en mémoire — jamais
	// issus d'une source — une quantité non nulle vaut présence.
	rules.Register(rules.Rule{
		ID: "BR-22", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-129"},
		Message: map[string]string{"fr": "Chaque ligne doit porter une quantité (BT-129)."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for _, l := range d.Lines {
				if l.QuantityPresent || !l.Quantity.IsZero() {
					continue
				}
				out = append(out, rules.Finding{RuleID: "BR-22", Severity: rules.SeverityError, Term: "BT-129",
					Message: "Quantité (BT-129) absente pour la ligne " + l.ID + "."})
			}
			return out
		},
	})

	// BR-48 : chaque ventilation de TVA doit porter un taux (BT-119), sauf catégorie « O » (hors
	// champ). Le taux réellement absent de la source est distingué d'un taux nul (Z/E/AE…).
	rules.Register(rules.Rule{
		ID: "BR-48", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-119"},
		Message: map[string]string{"fr": "Chaque ventilation de TVA doit porter un taux (BT-119), sauf hors champ (O)."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for _, ts := range d.TaxBreakdown {
				if ts.Category == model.TaxOutsideScope || ts.RatePresent {
					continue
				}
				// Repli pour les documents en mémoire (non issus d'une source) : une catégorie à
				// taux positif porte de fait un taux. Les catégories à taux nul, elles, exigent la
				// présence explicite tracée à la lecture.
				if !ts.Rate.IsZero() {
					continue
				}
				out = append(out, rules.Finding{RuleID: "BR-48", Severity: rules.SeverityError, Term: "BT-119",
					Message: "Taux de TVA (BT-119) absent pour la ventilation de catégorie " + string(ts.Category) + "."})
			}
			return out
		},
	})

	// BR-65 : si une classification d'article (BT-158) est présente, elle doit porter un identifiant
	// de schéma (listID). Ne se déclenche que lorsque la classification est effectivement portée.
	rules.Register(rules.Rule{
		ID: "BR-65", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-158"},
		Message: map[string]string{"fr": "La classification d'article (BT-158) doit porter un identifiant de schéma."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for _, l := range d.Lines {
				if l.ClassificationID != "" && l.ClassificationScheme == "" {
					out = append(out, rules.Finding{RuleID: "BR-65", Severity: rules.SeverityError, Term: "BT-158",
						Message: "Classification d'article (BT-158) sans identifiant de schéma, ligne " + l.ID + "."})
				}
			}
			return out
		},
	})
}
