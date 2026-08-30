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
	rules.Register(brCL25EAS())
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

// brCL25EAS (BR-CL-25) : le schéma d'une adresse électronique (BT-34/49) appartient à la liste CEF EAS.
func brCL25EAS() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-25", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-34", "BT-49"},
		Message: map[string]string{"fr": "Le schéma d'une adresse électronique doit appartenir à la liste CEF EAS."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			check := func(a *model.ElectronicAddress, term, path string) {
				if a != nil && a.Scheme != "" && !model.IsKnownEAS(a.Scheme) {
					out = append(out, rules.Finding{RuleID: "BR-CL-25", Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Schéma d'adresse électronique inconnu « %s ».", a.Scheme), Path: path, Actual: a.Scheme})
				}
			}
			check(d.Seller.ElectronicAddr, "BT-34", "seller.electronicAddress.scheme")
			check(d.Buyer.ElectronicAddr, "BT-49", "buyer.electronicAddress.scheme")
			return out
		},
	}
}
