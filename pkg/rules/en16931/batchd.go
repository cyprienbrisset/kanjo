package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Listes de codes supplémentaires : motifs de remise (UNCL 5189), motifs de charge (UNCL 7161)
// et schémas d'adresse électronique (CEF EAS).

func init() {
	rules.Register(reasonCodeRule("BR-CL-19", "BT-98", false))
	rules.Register(reasonCodeRule("BR-CL-20", "BT-105", true))
}

// reasonCodeRule vérifie que les codes motif de remise (wantCharge=false, UNCL 5189) ou de charge
// (wantCharge=true, UNCL 7161) appartiennent à leur liste, au niveau document et ligne.
func reasonCodeRule(id, term string, wantCharge bool) rules.Rule {
	known := model.IsKnownAllowanceReason
	if wantCharge {
		known = model.IsKnownChargeReason
	}
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "Le code motif de remise/charge doit appartenir à sa liste officielle."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			flag := func(code, path string) {
				if code != "" && !known(code) {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Code motif inconnu « %s ».", code), Path: path, Actual: code})
				}
			}
			for i, ac := range d.AllowanceCharges {
				if ac.IsCharge == wantCharge {
					flag(ac.ReasonCode, fmt.Sprintf("allowanceCharges[%d].reasonCode", i))
				}
			}
			for i, l := range d.Lines {
				for j, ac := range l.AllowanceCharges {
					if ac.IsCharge == wantCharge {
						flag(ac.ReasonCode, fmt.Sprintf("lines[%d].allowanceCharges[%d].reasonCode", i, j))
					}
				}
			}
			return out
		},
	}
}
