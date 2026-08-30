package en16931

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// catSplitPayment est la catégorie de TVA « B » (split payment, régime italien).
const catSplitPayment = model.TaxCategoryCode("B")

// usesCategory indique si une catégorie de TVA donnée apparaît dans le document (lignes,
// ventilation, remises/charges de niveau document).
func usesCategory(d *model.Document, cat model.TaxCategoryCode) bool {
	for _, l := range d.Lines {
		if l.TaxCategory == cat {
			return true
		}
	}
	for _, ts := range d.TaxBreakdown {
		if ts.Category == cat {
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

func init() {
	// BR-B-01 : si la catégorie « B » (split payment) est employée, la facture doit être une
	// facture nationale italienne — tous les codes pays présents valent « IT ».
	rules.Register(rules.Rule{
		ID: "BR-B-01", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-151", "BT-95", "BT-102"},
		Message: map[string]string{"fr": "Le split payment (catégorie B) est réservé aux factures nationales italiennes."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !usesCategory(d, catSplitPayment) {
				return nil
			}
			for _, cc := range countryCodes(d) {
				if cc != "" && cc != "IT" {
					return []rules.Finding{{RuleID: "BR-B-01", Severity: rules.SeverityError, Term: "BT-151",
						Message: "Catégorie B (split payment) employée hors facture nationale italienne (pays " + cc + ").",
						Actual:  cc}}
				}
			}
			return nil
		},
	})

	// BR-B-02 : une facture qui emploie la catégorie « B » (split payment) ne doit pas aussi
	// employer la catégorie « S » (taux normal).
	rules.Register(rules.Rule{
		ID: "BR-B-02", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-151", "BT-95", "BT-102"},
		Message: map[string]string{"fr": "Une facture en split payment (B) ne peut pas mélanger la catégorie taux normal (S)."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if usesCategory(d, catSplitPayment) && usesCategory(d, model.TaxStandard) {
				return []rules.Finding{{RuleID: "BR-B-02", Severity: rules.SeverityError, Term: "BT-151",
					Message: "Catégories B (split payment) et S (taux normal) présentes simultanément."}}
			}
			return nil
		},
	})
}

// countryCodes rassemble les codes pays présents dans le document (vendeur, acheteur, livraison).
func countryCodes(d *model.Document) []string {
	out := []string{d.Seller.Address.CountryCode, d.Buyer.Address.CountryCode}
	if d.DeliverTo != nil {
		out = append(out, d.DeliverTo.Address.CountryCode)
	}
	return out
}
