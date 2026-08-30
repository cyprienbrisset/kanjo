package en16931

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// BR-01 : une facture doit porter un identifiant de spécification (BT-24, CustomizationID),
// tracé dans la provenance à la lecture. Son absence signale un document non identifié.
func init() {
	rules.Register(rules.Rule{
		ID: "BR-01", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-24"},
		Message: map[string]string{"fr": "La facture doit porter un identifiant de spécification (BT-24)."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.Provenance != nil && d.Provenance.SpecIdentifier != "" {
				return nil
			}
			return []rules.Finding{{RuleID: "BR-01", Severity: rules.SeverityError, Term: "BT-24",
				Message: "Identifiant de spécification (BT-24) absent."}}
		},
	})
}
