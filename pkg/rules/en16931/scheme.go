package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de schéma d'identifiant : les identifiants de parties, d'enregistrement, d'article
// normalisé et de lieu de livraison doivent porter un schéma de la liste ISO 6523 ICD.
// (Le registre CEF EAS partage ce socle ISO 6523 ; il sert de référence commune.)

func init() {
	rules.Register(schemeRule("BR-CL-10", "BT-29", func(d *model.Document) []string {
		return []string{d.Seller.IDScheme, d.Buyer.IDScheme}
	}))
	rules.Register(schemeRule("BR-CL-11", "BT-30", func(d *model.Document) []string {
		return []string{d.Seller.LegalIDScheme, d.Buyer.LegalIDScheme}
	}))
	rules.Register(schemeRule("BR-CL-21", "BT-157-1", func(d *model.Document) []string {
		var s []string
		for _, l := range d.Lines {
			s = append(s, l.StandardScheme)
		}
		return s
	}))
	rules.Register(schemeRule("BR-CL-26", "BT-71-1", func(d *model.Document) []string {
		if d.DeliverTo != nil {
			return []string{d.DeliverTo.LocationScheme}
		}
		return nil
	}))
}

// schemeRule vérifie que chaque schéma non vide renvoyé par get appartient au registre ISO 6523 ICD.
func schemeRule(id, term string, get func(*model.Document) []string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "Un identifiant de schéma doit appartenir à la liste ISO 6523 ICD."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for _, s := range get(d) {
				if s != "" && !model.IsKnownICD(s) {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Schéma d'identifiant inconnu « %s ».", s), Actual: s})
				}
			}
			return out
		},
	}
}
