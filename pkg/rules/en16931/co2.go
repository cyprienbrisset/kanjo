package en16931

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de cohérence complémentaires (BR-CO-09, BR-CO-18, BR-CO-25).

func init() {
	rules.Register(brCO09())
	rules.Register(brCO18())
	rules.Register(brCO25())
}

// brCO09 : le n° de TVA du vendeur (BT-31), s'il est présent, commence par un préfixe pays
// de deux lettres.
func brCO09() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-09", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-31"},
		Message: map[string]string{"fr": "Le n° de TVA du vendeur doit commencer par un préfixe pays à deux lettres."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			v := d.Seller.VATID
			if v == "" {
				return nil
			}
			if len(v) < 3 || !isAlpha(v[0]) || !isAlpha(v[1]) {
				return []rules.Finding{{
					RuleID: "BR-CO-09", Severity: rules.SeverityError, Term: "BT-31",
					Message: "Le n° de TVA du vendeur ne commence pas par un préfixe pays valide.",
					Actual:  v, Fixable: false,
				}}
			}
			return nil
		},
	}
}

// brCO18 : la facture doit comporter au moins une ventilation de TVA.
func brCO18() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-18", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BG-23"},
		Message: map[string]string{"fr": "Une facture doit comporter au moins une ventilation de TVA."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if len(d.TaxBreakdown) > 0 {
				return nil
			}
			return []rules.Finding{{RuleID: "BR-CO-18", Severity: rules.SeverityError, Term: "BG-23",
				Message: "Aucune ventilation de TVA."}}
		},
	}
}

// brCO25 : si le net à payer (BT-115) est positif, une date d'échéance (BT-9) ou des
// conditions de paiement (BT-20) doivent être présentes.
func brCO25() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-25", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-9", "BT-20", "BT-115"},
		Message: map[string]string{"fr": "Si un montant reste dû, une date d'échéance ou des conditions de paiement sont requises."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			// La notion de « montant dû » vise la facture ; un avoir n'a pas d'échéance de paiement.
			if d.IsCreditNote() {
				return nil
			}
			if d.Totals.DuePayableAmount.Value <= 0 {
				return nil
			}
			if d.DueDate != nil || d.PaymentTerms != "" {
				return nil
			}
			return []rules.Finding{{RuleID: "BR-CO-25", Severity: rules.SeverityError, Term: "BT-9",
				Message: "Montant dû positif sans date d'échéance ni conditions de paiement."}}
		},
	}
}

func isAlpha(c byte) bool { return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') }
