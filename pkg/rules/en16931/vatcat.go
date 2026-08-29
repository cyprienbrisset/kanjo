package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles par catégorie de TVA (BR-Z / BR-E / BR-AE / BR-K / BR-G / BR-O). Chaque catégorie
// « à taux nul » impose un taux de 0 ; les catégories d'exonération imposent en plus un motif.

func init() {
	// Taux nul obligatoire.
	rules.Register(zeroRateRule("BR-Z-05", model.TaxZeroRated, "à taux zéro"))
	rules.Register(zeroRateRule("BR-E-05", model.TaxExempt, "exonérée"))
	rules.Register(zeroRateRule("BR-AE-05", model.TaxReverseCharge, "en autoliquidation"))
	rules.Register(zeroRateRule("BR-K-05", model.TaxIntraCommunity, "intracommunautaire"))
	rules.Register(zeroRateRule("BR-G-05", model.TaxExport, "à l'export"))
	rules.Register(zeroRateRule("BR-O-05", model.TaxOutsideScope, "hors champ"))

	// Motif d'exonération obligatoire (sauf taux zéro Z).
	rules.Register(exemptionReasonRule("BR-E-10", model.TaxExempt, "exonérée"))
	rules.Register(exemptionReasonRule("BR-AE-10", model.TaxReverseCharge, "en autoliquidation"))
	rules.Register(exemptionReasonRule("BR-K-10", model.TaxIntraCommunity, "intracommunautaire"))
	rules.Register(exemptionReasonRule("BR-G-10", model.TaxExport, "à l'export"))
	rules.Register(exemptionReasonRule("BR-O-10", model.TaxOutsideScope, "hors champ"))
}

// zeroRateRule vérifie que toute ventilation et toute ligne de la catégorie ont un taux nul.
func zeroRateRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-119", "BT-152"},
		Message: map[string]string{"fr": fmt.Sprintf("Une TVA %s doit avoir un taux de zéro.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category == cat && !ts.Rate.IsZero() {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: "BT-119",
						Message: fmt.Sprintf("Ventilation %s avec un taux non nul (%s).", label, ts.Rate.String()),
						Path:    fmt.Sprintf("taxBreakdown[%d].rate", i), Actual: ts.Rate.String(),
					})
				}
			}
			for i, l := range d.Lines {
				if l.TaxCategory == cat && l.TaxRate != nil && !l.TaxRate.IsZero() {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: "BT-152",
						Message: fmt.Sprintf("Ligne %s %s avec un taux non nul (%s).", l.ID, label, l.TaxRate.String()),
						Path:    fmt.Sprintf("lines[%d].taxRate", i), Actual: l.TaxRate.String(),
					})
				}
			}
			return out
		},
	}
}

// exemptionReasonRule vérifie que toute ventilation de la catégorie porte un motif d'exonération.
func exemptionReasonRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-120", "BT-121"},
		Message: map[string]string{"fr": fmt.Sprintf("Une TVA %s doit indiquer un motif d'exonération.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category == cat && ts.ExemptionReason == "" && ts.ExemptionReasonCode == "" {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: "BT-120",
						Message: fmt.Sprintf("Ventilation %s sans motif d'exonération.", label),
						Path:    fmt.Sprintf("taxBreakdown[%d].exemptionReason", i),
					})
				}
			}
			return out
		},
	}
}
