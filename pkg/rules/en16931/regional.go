package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Familles régionales espagnoles à taux positif, qui se comportent comme le taux normal :
//   - BR-AF : IGIC (Canaries, catégorie « L ») ;
//   - BR-AG : IPSI (Ceuta/Melilla, catégorie « M »).
// Chaque famille reprend les 10 règles du taux normal (-01 à -10), via des constructeurs génériques.

func init() {
	registerRatedFamily("BR-AF", model.TaxCanaryIGIC, "IGIC (Canaries)")
	registerRatedFamily("BR-AG", model.TaxCeutaMelillaIPSI, "IPSI (Ceuta/Melilla)")
}

func registerRatedFamily(p string, cat model.TaxCategoryCode, label string) {
	rules.Register(breakdownExistsRule(p+"-01", cat, label))
	rules.Register(lineSellerVATRule(p+"-02", cat, label))
	rules.Register(acSellerVATRule(p+"-03", cat, label, false))
	rules.Register(acSellerVATRule(p+"-04", cat, label, true))
	rules.Register(lineRatePositiveRule(p+"-05", cat, label))
	rules.Register(acRateRule(p+"-06", cat, label, false, ratePositive))
	rules.Register(acRateRule(p+"-07", cat, label, true, ratePositive))
	rules.Register(ratedTaxableSumRule(p+"-08", cat, label))
	rules.Register(ratedTaxRule(p+"-09", cat, label))
	rules.Register(noExemptionReasonRule(p+"-10", cat, label))
}

// lineSellerVATRule (-02) : une ligne de la catégorie impose un identifiant TVA/fiscal du vendeur.
func lineSellerVATRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-31", "BT-32"},
		Message: map[string]string{"fr": fmt.Sprintf("Une ligne %s exige un identifiant TVA/fiscal du vendeur.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !lineHasCategory(d, cat) || d.Seller.VATID != "" || d.Seller.TaxID != "" {
				return nil
			}
			return []rules.Finding{{RuleID: id, Severity: rules.SeverityError, Term: "BT-31",
				Message: fmt.Sprintf("Catégorie %s sans identifiant TVA/fiscal du vendeur.", label), Path: "seller.vatId"}}
		},
	}
}

// lineRatePositiveRule (-05) : sur une ligne de la catégorie, le taux de TVA doit être positif.
func lineRatePositiveRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-152"},
		Message: map[string]string{"fr": fmt.Sprintf("Le taux de TVA d'une ligne %s doit être supérieur à zéro.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				if l.TaxCategory == cat && !ratePositive(l.TaxRate) {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: "BT-152",
						Message: fmt.Sprintf("Ligne %s %s sans taux positif.", l.ID, label),
						Path:    fmt.Sprintf("lines[%d].taxRate", i)})
				}
			}
			return out
		},
	}
}

// ratedTaxableSumRule (-08) : par taux, la base de la ventilation égale Σ nets + Σ charges − Σ remises
// de même catégorie et même taux.
func ratedTaxableSumRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-116"},
		Message: map[string]string{"fr": fmt.Sprintf("La base d'une ventilation %s doit égaler la somme des montants de même catégorie et même taux.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category != cat {
					continue
				}
				expected := model.ZeroAmount(d.CurrencyCode)
				for _, l := range d.Lines {
					if l.TaxCategory == cat && rateEqual(l.TaxRate, ts.Rate) {
						expected, _ = expected.Add(l.NetAmount)
					}
				}
				for _, ac := range d.AllowanceCharges {
					if ac.TaxCategory != cat || !rateEqual(ac.TaxRate, ts.Rate) {
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
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: "BT-116",
						Message:  fmt.Sprintf("Base %s au taux %s = %s, attendu %s.", label, ts.Rate.String(), ts.TaxableAmount.String(), expected.String()),
						Path:     fmt.Sprintf("taxBreakdown[%d].taxableAmount", i),
						Expected: expected.String(), Actual: ts.TaxableAmount.String()})
				}
			}
			return out
		},
	}
}

// ratedTaxRule (-09) : montant de TVA = base × taux.
func ratedTaxRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-117"},
		Message: map[string]string{"fr": fmt.Sprintf("Le montant de TVA d'une ventilation %s doit égaler la base multipliée par le taux.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category != cat {
					continue
				}
				if !amountValueEqual(ts.TaxAmount, ts.ComputeTaxAmount()) {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: "BT-117",
						Message:  fmt.Sprintf("Montant de TVA %s = %s, attendu %s.", label, ts.TaxAmount.String(), ts.ComputeTaxAmount().String()),
						Path:     fmt.Sprintf("taxBreakdown[%d].taxAmount", i),
						Expected: ts.ComputeTaxAmount().String(), Actual: ts.TaxAmount.String()})
				}
			}
			return out
		},
	}
}
