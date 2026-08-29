package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Complément des règles par catégorie de TVA (EN 16931) :
//   - « -01 » : toute catégorie employée sur une ligne/remise/charge impose une ventilation de
//     cette catégorie (généralisation de BR-S-01) ;
//   - « -02 » : les catégories imposables exigent un identifiant TVA (ou fiscal) du vendeur ;
//     l'autoliquidation et l'intracommunautaire exigent en plus l'identifiant TVA de l'acheteur ;
//   - BR-S-10 : une ventilation au taux normal ne porte pas de motif d'exonération.

func init() {
	// -01 : existence d'une ventilation pour chaque catégorie utilisée (S déjà couvert par BR-S-01).
	rules.Register(breakdownExistsRule("BR-Z-01", model.TaxZeroRated, "à taux zéro"))
	rules.Register(breakdownExistsRule("BR-E-01", model.TaxExempt, "exonérée"))
	rules.Register(breakdownExistsRule("BR-AE-01", model.TaxReverseCharge, "en autoliquidation"))
	rules.Register(breakdownExistsRule("BR-K-01", model.TaxIntraCommunity, "intracommunautaire"))
	rules.Register(breakdownExistsRule("BR-G-01", model.TaxExport, "à l'export"))
	rules.Register(breakdownExistsRule("BR-O-01", model.TaxOutsideScope, "hors champ"))

	// -02 : identifiant TVA/fiscal du vendeur pour les catégories imposables usuelles.
	rules.Register(sellerVATRule("BR-S-02", model.TaxStandard, "au taux normal"))
	rules.Register(sellerVATRule("BR-Z-02", model.TaxZeroRated, "à taux zéro"))
	rules.Register(sellerVATRule("BR-E-02", model.TaxExempt, "exonérée"))
	rules.Register(sellerVATRule("BR-G-02", model.TaxExport, "à l'export"))

	// -02 : autoliquidation et intracommunautaire exigent aussi l'identifiant TVA de l'acheteur.
	rules.Register(sellerBuyerVATRule("BR-AE-02", model.TaxReverseCharge, "en autoliquidation"))
	rules.Register(sellerBuyerVATRule("BR-K-02", model.TaxIntraCommunity, "intracommunautaire"))

	// BR-S-10 : pas de motif d'exonération sur une ventilation au taux normal.
	rules.Register(noExemptionReasonRule("BR-S-10", model.TaxStandard, "au taux normal"))
}

// categoryOnLineOrAC indique si une catégorie apparaît sur une ligne, une remise ou une charge.
func categoryOnLineOrAC(d *model.Document, cat model.TaxCategoryCode) bool {
	for _, l := range d.Lines {
		if l.TaxCategory == cat {
			return true
		}
	}
	for _, ac := range d.AllowanceCharges {
		if ac.TaxCategory == cat {
			return true
		}
	}
	return false
}

// categoryUsed inclut en plus la ventilation de TVA.
func categoryUsed(d *model.Document, cat model.TaxCategoryCode) bool {
	if categoryOnLineOrAC(d, cat) {
		return true
	}
	for _, ts := range d.TaxBreakdown {
		if ts.Category == cat {
			return true
		}
	}
	return false
}

func breakdownExistsRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-118"},
		Message: map[string]string{"fr": fmt.Sprintf("Une catégorie %s employée impose une ventilation de TVA correspondante.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !categoryOnLineOrAC(d, cat) {
				return nil
			}
			for _, ts := range d.TaxBreakdown {
				if ts.Category == cat {
					return nil
				}
			}
			return []rules.Finding{{
				RuleID: id, Severity: rules.SeverityError, Term: "BT-118",
				Message: fmt.Sprintf("Catégorie %s utilisée sans ventilation de TVA correspondante.", label),
			}}
		},
	}
}

func sellerVATRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-31", "BT-32"},
		Message: map[string]string{"fr": fmt.Sprintf("Une catégorie %s exige un identifiant TVA ou fiscal du vendeur.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !categoryUsed(d, cat) {
				return nil
			}
			if d.Seller.VATID != "" || d.Seller.TaxID != "" {
				return nil
			}
			return []rules.Finding{{
				RuleID: id, Severity: rules.SeverityError, Term: "BT-31",
				Message: fmt.Sprintf("Catégorie %s présente sans identifiant TVA/fiscal du vendeur.", label),
				Path:    "seller.vatId",
			}}
		},
	}
}

func sellerBuyerVATRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-31", "BT-48"},
		Message: map[string]string{"fr": fmt.Sprintf("Une catégorie %s exige les identifiants TVA du vendeur et de l'acheteur.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !categoryUsed(d, cat) {
				return nil
			}
			var out []rules.Finding
			if d.Seller.VATID == "" && d.Seller.TaxID == "" {
				out = append(out, rules.Finding{
					RuleID: id, Severity: rules.SeverityError, Term: "BT-31",
					Message: fmt.Sprintf("Catégorie %s sans identifiant TVA/fiscal du vendeur.", label),
					Path:    "seller.vatId",
				})
			}
			if d.Buyer.VATID == "" {
				out = append(out, rules.Finding{
					RuleID: id, Severity: rules.SeverityError, Term: "BT-48",
					Message: fmt.Sprintf("Catégorie %s sans identifiant TVA de l'acheteur.", label),
					Path:    "buyer.vatId",
				})
			}
			return out
		},
	}
}

func noExemptionReasonRule(id string, cat model.TaxCategoryCode, label string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-120", "BT-121"},
		Message: map[string]string{"fr": fmt.Sprintf("Une ventilation %s ne doit pas porter de motif d'exonération.", label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category == cat && (ts.ExemptionReason != "" || ts.ExemptionReasonCode != "") {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: "BT-120",
						Message: fmt.Sprintf("Ventilation %s avec un motif d'exonération indu.", label),
						Path:    fmt.Sprintf("taxBreakdown[%d].exemptionReason", i),
					})
				}
			}
			return out
		},
	}
}
