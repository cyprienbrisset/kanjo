// Package en16931 implémente les règles de la norme EN 16931. Ce fichier couvre les règles
// de cohérence des totaux (BR-CO-*), réellement calculées à partir du pivot (§17.7).
//
// Chaque règle est enregistrée dans le registre du moteur (pkg/rules) via init().
package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

const setEN = "en16931"

func init() {
	rules.Register(brCO10())
	rules.Register(brCO13())
	rules.Register(brCO14())
	rules.Register(brCO15())
	rules.Register(brCO16())
	rules.Register(brCO17())
}

// mismatch construit une anomalie d'écart de montant si expected != actual.
func mismatch(ruleID, term, msgFR string, expected, actual model.Amount) []rules.Finding {
	if expected.Equal(actual) {
		return nil
	}
	return []rules.Finding{{
		RuleID:   ruleID,
		Severity: rules.SeverityError,
		Message:  msgFR,
		Term:     term,
		Expected: expected.String(),
		Actual:   actual.String(),
		Fixable:  true,
	}}
}

func brCO10() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-10", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-106", "BT-131"},
		Message: map[string]string{"fr": "La somme des montants nets de ligne doit être égale au total des lignes (BT-106).", "en": "Sum of line net amounts must equal BT-106."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			sum, err := d.SumLineNetAmounts()
			if err != nil {
				return nil
			}
			return mismatch("BR-CO-10", "BT-106",
				"La somme des montants nets de ligne ne correspond pas au total des lignes.",
				sum, d.Totals.LineExtensionAmount)
		},
	}
}

func brCO13() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-13", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-106", "BT-107", "BT-108", "BT-109"},
		Message: map[string]string{"fr": "Le total HT (BT-109) doit être égal au total des lignes moins les remises plus les charges.", "en": "BT-109 must equal BT-106 - BT-107 + BT-108."},
		Check: func(d *model.Document, ctx *rules.Context) []rules.Finding {
			exp, err := d.Totals.ComputeTaxExclusive(ctx.Currency)
			if err != nil {
				return nil
			}
			return mismatch("BR-CO-13", "BT-109",
				"Le total HT ne correspond pas à la somme des lignes moins les remises plus les charges.",
				exp, d.Totals.TaxExclusiveAmount)
		},
	}
}

func brCO14() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-14", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-110", "BT-117"},
		Message: map[string]string{"fr": "Le total de TVA (BT-110) doit être égal à la somme des montants de TVA par catégorie.", "en": "BT-110 must equal the sum of BT-117."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			sum, err := d.SumTaxAmounts()
			if err != nil {
				return nil
			}
			return mismatch("BR-CO-14", "BT-110",
				"Le total de TVA ne correspond pas à la somme des montants de TVA par catégorie.",
				sum, d.Totals.TaxAmount)
		},
	}
}

func brCO15() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-15", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-109", "BT-110", "BT-112"},
		Message: map[string]string{"fr": "Le total TTC (BT-112) doit être égal au total HT plus le total de TVA.", "en": "BT-112 must equal BT-109 + BT-110."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			exp, err := d.Totals.ComputeTaxInclusive()
			if err != nil {
				return nil
			}
			return mismatch("BR-CO-15", "BT-112",
				"Le total TTC ne correspond pas au total HT plus le total de TVA.",
				exp, d.Totals.TaxInclusiveAmount)
		},
	}
}

func brCO16() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-16", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-112", "BT-113", "BT-114", "BT-115"},
		Message: map[string]string{"fr": "Le net à payer (BT-115) doit être égal au total TTC moins l'acompte plus l'arrondi.", "en": "BT-115 must equal BT-112 - BT-113 + BT-114."},
		Check: func(d *model.Document, ctx *rules.Context) []rules.Finding {
			exp, err := d.Totals.ComputeDuePayable(ctx.Currency)
			if err != nil {
				return nil
			}
			return mismatch("BR-CO-16", "BT-115",
				"Le net à payer ne correspond pas au total TTC moins l'acompte plus l'arrondi.",
				exp, d.Totals.DuePayableAmount)
		},
	}
}

func brCO17() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-17", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-116", "BT-117", "BT-119"},
		Message: map[string]string{"fr": "Le montant de TVA par catégorie (BT-117) doit être égal à la base multipliée par le taux.", "en": "BT-117 must equal BT-116 × (BT-119 / 100)."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var findings []rules.Finding
			for i, ts := range d.TaxBreakdown {
				exp := ts.ComputeTaxAmount()
				if exp.Equal(ts.TaxAmount) {
					continue
				}
				findings = append(findings, rules.Finding{
					RuleID:   "BR-CO-17",
					Severity: rules.SeverityError,
					Message:  fmt.Sprintf("Le montant de TVA de la ventilation #%d ne correspond pas à la base × taux.", i+1),
					Term:     "BT-117",
					Path:     fmt.Sprintf("taxBreakdown[%d].taxAmount", i),
					Expected: exp.String(),
					Actual:   ts.TaxAmount.String(),
					Fixable:  true,
				})
			}
			return findings
		},
	}
}
