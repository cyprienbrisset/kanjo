// Package kanjo regroupe les règles maison de Kanjō (jeu "kanjo") : contrôles de cohérence
// et de qualité au-delà de la norme (dates aberrantes, IBAN, doublons…).
package kanjo

import (
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

const setKanjo = "kanjo"

func init() {
	rules.Register(dueAfterIssue())
	rules.Register(ibanValid())
}

// dueAfterIssue : la date d'échéance (BT-9) ne doit pas précéder la date d'émission (BT-2).
func dueAfterIssue() rules.Rule {
	return rules.Rule{
		ID: "KANJO-DATE-01", Set: setKanjo, Severity: rules.SeverityWarning,
		Terms:   []string{"BT-9", "BT-2"},
		Message: map[string]string{"fr": "La date d'échéance ne doit pas précéder la date d'émission."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.DueDate == nil || d.IssueDate.IsZero() {
				return nil
			}
			if d.DueDate.Before(d.IssueDate) {
				return []rules.Finding{{
					RuleID: "KANJO-DATE-01", Severity: rules.SeverityWarning, Term: "BT-9",
					Message:  "La date d'échéance précède la date d'émission.",
					Expected: "≥ " + d.IssueDate.ISO(), Actual: d.DueDate.ISO(),
				}}
			}
			return nil
		},
	}
}

// ibanValid : tout IBAN renseigné dans les instructions de paiement doit satisfaire la
// vérification modulo 97 (ISO 13616).
func ibanValid() rules.Rule {
	return rules.Rule{
		ID: "KANJO-IBAN-01", Set: setKanjo, Severity: rules.SeverityWarning,
		Terms:   []string{"BT-84"},
		Message: map[string]string{"fr": "L'IBAN doit satisfaire la vérification modulo 97 (ISO 13616)."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.PaymentInstructions == nil {
				return nil
			}
			var out []rules.Finding
			for _, ct := range d.PaymentInstructions.CreditTransfers {
				if ct.IBAN == "" {
					continue
				}
				if !validIBAN(ct.IBAN) {
					out = append(out, rules.Finding{
						RuleID: "KANJO-IBAN-01", Severity: rules.SeverityWarning, Term: "BT-84",
						Message: "IBAN invalide (échec du contrôle modulo 97).", Actual: ct.IBAN, Fixable: false,
					})
				}
			}
			return out
		},
	}
}

// validIBAN applique l'algorithme de contrôle IBAN ISO 13616 (réarrangement + modulo 97 == 1).
func validIBAN(iban string) bool {
	s := strings.ToUpper(strings.ReplaceAll(iban, " ", ""))
	if len(s) < 15 || len(s) > 34 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	// Déplacer les 4 premiers caractères à la fin.
	rearranged := s[4:] + s[:4]
	// Convertir les lettres en nombres (A=10 … Z=35) et calculer le modulo 97 par flux.
	rem := 0
	for i := 0; i < len(rearranged); i++ {
		c := rearranged[i]
		switch {
		case c >= '0' && c <= '9':
			rem = (rem*10 + int(c-'0')) % 97
		default: // A-Z → 10..35, soit deux chiffres
			v := int(c-'A') + 10
			rem = (rem*100 + v) % 97
		}
	}
	return rem == 1
}
