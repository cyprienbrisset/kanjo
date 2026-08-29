package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de structure des remises et charges d'EN 16931.
//   - Niveau document (BG-20 remise / BG-21 charge) : BR-31/32/33 et BR-36/37/38.
//   - Niveau ligne     (BG-27 remise / BG-28 charge) : BR-41/42/43/44.
//
// Aucune donnée n'est inventée : ces règles constatent l'absence d'un terme obligatoire.

func init() {
	// --- Remises de niveau document (BG-20) ---
	rules.Register(docACRule("BR-31", "BT-92", false,
		"Une remise de niveau document doit porter un montant.", acAmountPresent))
	rules.Register(docACRule("BR-32", "BT-95", false,
		"Une remise de niveau document doit porter une catégorie de TVA.", acHasCategory))
	rules.Register(docACRule("BR-33", "BT-97", false,
		"Une remise de niveau document doit porter un motif ou un code motif.", acHasReason))

	// --- Charges de niveau document (BG-21) ---
	rules.Register(docACRule("BR-36", "BT-99", true,
		"Une charge de niveau document doit porter un montant.", acAmountPresent))
	rules.Register(docACRule("BR-37", "BT-102", true,
		"Une charge de niveau document doit porter une catégorie de TVA.", acHasCategory))
	rules.Register(docACRule("BR-38", "BT-104", true,
		"Une charge de niveau document doit porter un motif ou un code motif.", acHasReason))

	// --- Remises de ligne (BG-27) ---
	rules.Register(lineACRule("BR-41", "BT-136", false,
		"Une remise de ligne doit porter un montant.", acAmountPresent))
	rules.Register(lineACRule("BR-42", "BT-139", false,
		"Une remise de ligne doit porter un motif ou un code motif.", acHasReason))

	// --- Charges de ligne (BG-28) ---
	rules.Register(lineACRule("BR-43", "BT-141", true,
		"Une charge de ligne doit porter un montant.", acAmountPresent))
	rules.Register(lineACRule("BR-44", "BT-144", true,
		"Une charge de ligne doit porter un motif ou un code motif.", acHasReason))
}

// acAmountPresent vérifie qu'un montant est effectivement porté (devise renseignée).
func acAmountPresent(ac model.AllowanceCharge) bool { return ac.Amount.Currency != "" }

// acHasCategory vérifie la présence d'une catégorie de TVA (niveau document uniquement).
func acHasCategory(ac model.AllowanceCharge) bool { return ac.TaxCategory != "" }

// acHasReason vérifie la présence d'un motif ou d'un code motif.
func acHasReason(ac model.AllowanceCharge) bool { return ac.Reason != "" || ac.ReasonCode != "" }

// docACRule applique un prédicat à chaque remise (wantCharge=false) ou charge (true) de niveau
// document et émet une anomalie pour celles qui échouent.
func docACRule(id, term string, wantCharge bool, msgFR string, ok func(model.AllowanceCharge) bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{term},
		Message: map[string]string{"fr": msgFR},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ac := range d.AllowanceCharges {
				if ac.IsCharge != wantCharge || ok(ac) {
					continue
				}
				out = append(out, rules.Finding{
					RuleID: id, Severity: rules.SeverityError, Term: term,
					Message: msgFR,
					Path:    fmt.Sprintf("allowanceCharges[%d]", i),
				})
			}
			return out
		},
	}
}

// lineACRule applique un prédicat à chaque remise (wantCharge=false) ou charge (true) de ligne.
func lineACRule(id, term string, wantCharge bool, msgFR string, ok func(model.AllowanceCharge) bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{term},
		Message: map[string]string{"fr": msgFR},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				for j, ac := range l.AllowanceCharges {
					if ac.IsCharge != wantCharge || ok(ac) {
						continue
					}
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("%s (ligne %s)", msgFR, lineLabel(l, i)),
						Path:    fmt.Sprintf("lines[%d].allowanceCharges[%d]", i, j),
					})
				}
			}
			return out
		},
	}
}
