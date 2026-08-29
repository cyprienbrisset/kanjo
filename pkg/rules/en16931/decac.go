package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles décimales sur les montants de remises et charges (au plus deux décimales) :
//   - niveau document : BR-DEC-01 (remise BT-92), BR-DEC-05 (charge BT-99) ;
//   - niveau ligne    : BR-DEC-24 (remise BT-136), BR-DEC-27 (charge BT-141).

func init() {
	rules.Register(decDocACRule("BR-DEC-01", "BT-92", false))
	rules.Register(decDocACRule("BR-DEC-05", "BT-99", true))
	rules.Register(decLineACRule("BR-DEC-24", "BT-136", false))
	rules.Register(decLineACRule("BR-DEC-27", "BT-141", true))
}

func decDocACRule(id, term string, wantCharge bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "Un montant de remise/charge ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ac := range d.AllowanceCharges {
				if ac.IsCharge != wantCharge || ac.Amount.Scale <= 2 {
					continue
				}
				out = append(out, rules.Finding{
					RuleID: id, Severity: rules.SeverityError, Term: term,
					Message: fmt.Sprintf("Montant à %d décimales (max 2).", ac.Amount.Scale),
					Path:    fmt.Sprintf("allowanceCharges[%d].amount", i), Actual: ac.Amount.String(), Fixable: true,
				})
			}
			return out
		},
	}
}

func decLineACRule(id, term string, wantCharge bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "Un montant de remise/charge de ligne ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				for j, ac := range l.AllowanceCharges {
					if ac.IsCharge != wantCharge || ac.Amount.Scale <= 2 {
						continue
					}
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Montant à %d décimales (max 2) sur la ligne %s.", ac.Amount.Scale, lineLabel(l, i)),
						Path:    fmt.Sprintf("lines[%d].allowanceCharges[%d].amount", i, j), Actual: ac.Amount.String(), Fixable: true,
					})
				}
			}
			return out
		},
	}
}
