package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de calcul et de cohérence de la ventilation de TVA par catégorie (EN 16931) :
//   - « -08 » : base d'imposition de la catégorie = Σ nets de ligne − Σ remises + Σ charges de la
//     même catégorie ;
//   - « -09 » : montant de TVA = 0 pour les catégories à taux nul ; = base × taux pour le taux normal ;
//   - BR-Z-10 : une ventilation à taux zéro ne porte pas de motif d'exonération.

var zeroTaxCats = []struct {
	cat        model.TaxCategoryCode
	label      string
	id08, id09 string
}{
	{model.TaxZeroRated, "à taux zéro", "BR-Z-08", "BR-Z-09"},
	{model.TaxExempt, "exonérée", "BR-E-08", "BR-E-09"},
	{model.TaxReverseCharge, "en autoliquidation", "BR-AE-08", "BR-AE-09"},
	{model.TaxExport, "à l'export", "BR-G-08", "BR-G-09"},
	{model.TaxIntraCommunity, "intracommunautaire", "BR-IC-08", "BR-IC-09"},
	{model.TaxOutsideScope, "hors champ", "BR-O-08", "BR-O-09"},
}

func init() {
	for _, c := range zeroTaxCats {
		rules.Register(breakdownTaxableSumRule(c.id08, c.cat, c.label))
		rules.Register(breakdownZeroTaxRule(c.id09, c.cat, c.label))
	}
	// BR-S-08 (base par taux) est couvert au niveau totaux par BR-CO-10/13 ; BR-S-09 vérifie le
	// calcul du montant de TVA au taux normal (base × taux).
	rules.Register(breakdownStandardTaxRule())
	// BR-S-08 : base par taux (une ventilation « taux normal » par valeur de taux).
	rules.Register(breakdownStandardTaxableSumRule())
	// BR-Z-10 : pas de motif d'exonération sur une ventilation à taux zéro.
	rules.Register(noExemptionReasonRule("BR-Z-10", model.TaxZeroRated, "à taux zéro"))
}

// categoryNetSum calcule Σ nets de ligne − Σ remises document + Σ charges document pour une
// catégorie donnée (base d'imposition attendue, arrondie à 2 décimales).
func categoryNetSum(d *model.Document, cat model.TaxCategoryCode) model.Amount {
	sum := model.ZeroAmount(d.CurrencyCode)
	for _, l := range d.Lines {
		if l.TaxCategory == cat {
			sum, _ = sum.Add(l.NetAmount)
		}
	}
	for _, ac := range d.AllowanceCharges {
		if ac.TaxCategory != cat {
			continue
		}
		if ac.IsCharge {
			sum, _ = sum.Add(ac.Amount)
		} else {
			sum, _ = sum.Sub(ac.Amount)
		}
	}
	return sum.Rescale(2)
}

func amountValueEqual(a, b model.Amount) bool {
	return a.Rescale(2).Value == b.Rescale(2).Value
}

// breakdownTaxableSumRule (-08) : la base de la catégorie égale la somme calculée.
func breakdownTaxableSumRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-116"},
		Message: map[string]string{"fr": fmt.Sprintf("La base d'imposition d'une ventilation %s doit égaler la somme des montants de cette catégorie.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			expected := categoryNetSum(d, cat)
			for i, ts := range d.TaxBreakdown {
				if ts.Category != cat {
					continue
				}
				if !amountValueEqual(ts.TaxableAmount, expected) {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: "BT-116",
						Message:  fmt.Sprintf("Base %s = %s, attendu %s.", label, ts.TaxableAmount.String(), expected.String()),
						Path:     fmt.Sprintf("taxBreakdown[%d].taxableAmount", i),
						Expected: expected.String(), Actual: ts.TaxableAmount.String(),
					})
				}
			}
			return out
		},
	}
}

// breakdownZeroTaxRule (-09) : le montant de TVA d'une catégorie à taux nul est 0.
func breakdownZeroTaxRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-117"},
		Message: map[string]string{"fr": fmt.Sprintf("Le montant de TVA d'une ventilation %s doit être nul.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category == cat && ts.TaxAmount.Value != 0 {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: "BT-117",
						Message: fmt.Sprintf("Montant de TVA non nul (%s) sur une ventilation %s.", ts.TaxAmount.String(), label),
						Path:    fmt.Sprintf("taxBreakdown[%d].taxAmount", i),
					})
				}
			}
			return out
		},
	}
}

func rateEqual(a *model.Decimal, b model.Decimal) bool {
	if a == nil {
		return b.IsZero()
	}
	return a.Rescale(4).Unscaled == b.Rescale(4).Unscaled
}

// breakdownStandardTaxableSumRule (BR-S-08) : pour chaque taux, la base de la ventilation « taux
// normal » égale Σ nets de ligne + Σ charges − Σ remises de même catégorie ET même taux.
func breakdownStandardTaxableSumRule() rules.Rule {
	return rules.Rule{
		ID: "BR-S-08", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-116"},
		Message: map[string]string{"fr": "La base d'une ventilation au taux normal doit égaler la somme des montants de même catégorie et même taux."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category != model.TaxStandard {
					continue
				}
				expected := model.ZeroAmount(d.CurrencyCode)
				for _, l := range d.Lines {
					if l.TaxCategory == model.TaxStandard && rateEqual(l.TaxRate, ts.Rate) {
						expected, _ = expected.Add(l.NetAmount)
					}
				}
				for _, ac := range d.AllowanceCharges {
					if ac.TaxCategory != model.TaxStandard || !rateEqual(ac.TaxRate, ts.Rate) {
						continue
					}
					if ac.IsCharge {
						expected, _ = expected.Add(ac.Amount)
					} else {
						expected, _ = expected.Sub(ac.Amount)
					}
				}
				expected = expected.Rescale(2)
				if !amountValueEqual(ts.TaxableAmount, expected) {
					out = append(out, rules.Finding{
						RuleID: "BR-S-08", Severity: rules.SeverityError, Term: "BT-116",
						Message:  fmt.Sprintf("Base au taux %s = %s, attendu %s.", ts.Rate.String(), ts.TaxableAmount.String(), expected.String()),
						Path:     fmt.Sprintf("taxBreakdown[%d].taxableAmount", i),
						Expected: expected.String(), Actual: ts.TaxableAmount.String(),
					})
				}
			}
			return out
		},
	}
}

// breakdownStandardTaxRule (BR-S-09) : montant de TVA = base × taux au taux normal.
func breakdownStandardTaxRule() rules.Rule {
	return rules.Rule{
		ID: "BR-S-09", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-117"},
		Message: map[string]string{"fr": "Le montant de TVA au taux normal doit égaler la base multipliée par le taux."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category != model.TaxStandard {
					continue
				}
				if !amountValueEqual(ts.TaxAmount, ts.ComputeTaxAmount()) {
					out = append(out, rules.Finding{
						RuleID: "BR-S-09", Severity: rules.SeverityError, Term: "BT-117",
						Message:  fmt.Sprintf("Montant de TVA = %s, attendu %s (base × taux).", ts.TaxAmount.String(), ts.ComputeTaxAmount().String()),
						Path:     fmt.Sprintf("taxBreakdown[%d].taxAmount", i),
						Expected: ts.ComputeTaxAmount().String(), Actual: ts.TaxAmount.String(),
					})
				}
			}
			return out
		},
	}
}
