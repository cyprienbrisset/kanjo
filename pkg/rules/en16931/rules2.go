package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Lot complémentaire de règles EN 16931 (présence, identification, décimales de ligne),
// choisies pour être calculables sans ambiguïté à partir du pivot actuel.

func init() {
	rules.Register(presence("BR-08", "BG-5", "L'adresse postale du vendeur est obligatoire.",
		func(d *model.Document) bool { return !d.Seller.Address.Empty() }))
	rules.Register(presence("BR-10", "BG-8", "L'adresse postale de l'acheteur est obligatoire.",
		func(d *model.Document) bool { return !d.Buyer.Address.Empty() }))
	rules.Register(presence("BR-CO-26", "BT-29", "Le vendeur doit être identifiable (identifiant, SIREN/SIRET ou n° de TVA).",
		func(d *model.Document) bool {
			return d.Seller.ID != "" || d.Seller.LegalID != "" || d.Seller.VATID != ""
		}))
	rules.Register(brBreakdownCategory())
	rules.Register(brLineNetDecimals())
}

// brBreakdownCategory (BR-47) : chaque ventilation de TVA doit porter un code catégorie.
func brBreakdownCategory() rules.Rule {
	return rules.Rule{
		ID: "BR-47", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-118"},
		Message: map[string]string{"fr": "Chaque ventilation de TVA doit indiquer une catégorie."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category == "" {
					out = append(out, rules.Finding{
						RuleID: "BR-47", Severity: rules.SeverityError, Term: "BT-118",
						Message: fmt.Sprintf("Ventilation de TVA #%d sans catégorie.", i+1),
						Path:    fmt.Sprintf("taxBreakdown[%d].category", i),
					})
				}
			}
			return out
		},
	}
}

// brLineNetDecimals (BR-DEC-23) : le montant net de chaque ligne (BT-131) a au plus 2 décimales.
func brLineNetDecimals() rules.Rule {
	return rules.Rule{
		ID: "BR-DEC-23", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-131"},
		Message: map[string]string{"fr": "Le montant net d'une ligne ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				if l.NetAmount.Scale > 2 {
					out = append(out, rules.Finding{
						RuleID: "BR-DEC-23", Severity: rules.SeverityError, Term: "BT-131",
						Message: fmt.Sprintf("Montant net de la ligne %s à plus de deux décimales.", l.ID),
						Path:    fmt.Sprintf("lines[%d].netAmount", i), Fixable: true,
					})
				}
			}
			return out
		},
	}
}
