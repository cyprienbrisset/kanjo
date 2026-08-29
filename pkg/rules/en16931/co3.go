package en16931

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Cohérence des totaux de remises et charges de niveau document (BR-CO-11, BR-CO-12).
// Désormais calculables car les remises/charges (BG-20/21) sont lues par les lecteurs.

func init() {
	rules.Register(brCO11())
	rules.Register(brCO12())
}

func sumAllowanceCharges(d *model.Document, charge bool) model.Amount {
	acc := model.ZeroAmount(d.CurrencyCode)
	for _, ac := range d.AllowanceCharges {
		if ac.IsCharge == charge {
			acc, _ = acc.Add(ac.Amount)
		}
	}
	return acc.Rescale(2)
}

func brCO11() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-11", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-107", "BT-92"},
		Message: map[string]string{"fr": "Le total des remises (BT-107) doit être égal à la somme des remises de niveau document."},
		Check: func(d *model.Document, ctx *rules.Context) []rules.Finding {
			sum := sumAllowanceCharges(d, false)
			declared := amountOrZero2(d.Totals.AllowanceTotal, ctx.Currency)
			// Rien à vérifier si aucune remise et aucun total déclaré.
			if sum.IsZero() && d.Totals.AllowanceTotal == nil {
				return nil
			}
			return diffAmount("BR-CO-11", "BT-107",
				"Le total des remises ne correspond pas à la somme des remises de niveau document.",
				sum, declared)
		},
	}
}

func brCO12() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-12", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-108", "BT-99"},
		Message: map[string]string{"fr": "Le total des charges (BT-108) doit être égal à la somme des charges de niveau document."},
		Check: func(d *model.Document, ctx *rules.Context) []rules.Finding {
			sum := sumAllowanceCharges(d, true)
			declared := amountOrZero2(d.Totals.ChargeTotal, ctx.Currency)
			if sum.IsZero() && d.Totals.ChargeTotal == nil {
				return nil
			}
			return diffAmount("BR-CO-12", "BT-108",
				"Le total des charges ne correspond pas à la somme des charges de niveau document.",
				sum, declared)
		},
	}
}

func amountOrZero2(a *model.Amount, currency string) model.Amount {
	if a == nil {
		return model.ZeroAmount(currency)
	}
	return *a
}

func diffAmount(id, term, msg string, expected, actual model.Amount) []rules.Finding {
	if expected.Equal(actual) {
		return nil
	}
	return []rules.Finding{{
		RuleID: id, Severity: rules.SeverityError, Message: msg, Term: term,
		Expected: expected.String(), Actual: actual.String(), Fixable: true,
	}}
}
