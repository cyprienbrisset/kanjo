package en16931

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles sur les instructions de paiement (BG-16), les pièces jointes (BG-24) et la devise de
// comptabilisation de la TVA (BT-6/111). Sémantique alignée sur le Schematron officiel.

func init() {
	rules.Register(br49PaymentMeansCode())
	rules.Register(br50CreditTransferAccount())
	rules.Register(br51CardPAN())
	rules.Register(br52Attachments())
	rules.Register(br53TaxAccountingCurrency())
	rules.Register(brCL05TaxCurrency())
	rules.Register(brDEC15())
}

// br49PaymentMeansCode (BR-49) : une instruction de paiement doit préciser un moyen de paiement (BT-81).
func br49PaymentMeansCode() rules.Rule {
	return rules.Rule{
		ID: "BR-49", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-81"},
		Message: map[string]string{"fr": "Une instruction de paiement doit préciser un moyen de paiement."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			pi := d.PaymentInstructions
			if pi != nil && pi.MeansCode == "" {
				return []rules.Finding{{RuleID: "BR-49", Severity: rules.SeverityError, Term: "BT-81",
					Message: "Instruction de paiement sans moyen de paiement.", Path: "paymentInstructions.meansCode"}}
			}
			return nil
		},
	}
}

// br50CreditTransferAccount (BR-50) : un virement (BG-17) impose un identifiant de compte (BT-84).
func br50CreditTransferAccount() rules.Rule {
	return rules.Rule{
		ID: "BR-50", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-84"},
		Message: map[string]string{"fr": "Un virement doit indiquer un identifiant de compte."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			pi := d.PaymentInstructions
			if pi == nil || len(pi.CreditTransfers) == 0 {
				return nil
			}
			for _, ct := range pi.CreditTransfers {
				if ct.IBAN != "" {
					return nil
				}
			}
			return []rules.Finding{{RuleID: "BR-50", Severity: rules.SeverityError, Term: "BT-84",
				Message: "Virement sans identifiant de compte.", Path: "paymentInstructions.creditTransfers"}}
		},
	}
}

// br51CardPAN (BR-51) : sécurité carte — le numéro de carte (BT-87) ne doit pas dépasser 10 caractères
// (numéro masqué). Avertissement, non bloquant.
func br51CardPAN() rules.Rule {
	return rules.Rule{
		ID: "BR-51", Set: setEN, Severity: rules.SeverityWarning, Terms: []string{"BT-87"},
		Message: map[string]string{"fr": "Le numéro de carte doit être masqué (au plus 10 caractères)."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			pi := d.PaymentInstructions
			if pi == nil || pi.Card == nil {
				return nil
			}
			if len(strings.TrimSpace(pi.Card.PAN)) > 10 {
				return []rules.Finding{{RuleID: "BR-51", Severity: rules.SeverityWarning, Term: "BT-87",
					Message: "Numéro de carte possiblement complet (doit être masqué).", Path: "paymentInstructions.card.pan"}}
			}
			return nil
		},
	}
}

// br52Attachments (BR-52) : chaque document additionnel (BG-24) doit porter une référence (BT-122).
func br52Attachments() rules.Rule {
	return rules.Rule{
		ID: "BR-52", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-122"},
		Message: map[string]string{"fr": "Chaque document additionnel doit porter une référence."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, a := range d.Attachments {
				if a.ID == "" {
					out = append(out, rules.Finding{RuleID: "BR-52", Severity: rules.SeverityError, Term: "BT-122",
						Message: "Document additionnel sans référence.", Path: fmt.Sprintf("attachments[%d].id", i)})
				}
			}
			return out
		},
	}
}

// br53TaxAccountingCurrency (BR-53) : si la devise de comptabilisation de la TVA (BT-6) est présente,
// le total de TVA dans cette devise (BT-111) doit l'être aussi.
func br53TaxAccountingCurrency() rules.Rule {
	return rules.Rule{
		ID: "BR-53", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-111"},
		Message: map[string]string{"fr": "Une devise de comptabilisation de la TVA impose le total de TVA correspondant."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.TaxCurrencyCode != "" && d.Totals.TaxAmountInAccountingCurrency == nil {
				return []rules.Finding{{RuleID: "BR-53", Severity: rules.SeverityError, Term: "BT-111",
					Message: "Total de TVA en devise de comptabilisation manquant.", Path: "totals.taxAmountAccounting"}}
			}
			return nil
		},
	}
}

// brCL05TaxCurrency (BR-CL-05) : la devise de comptabilisation de la TVA (BT-6) est un code ISO 4217.
func brCL05TaxCurrency() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-05", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-6"},
		Message: map[string]string{"fr": "La devise de comptabilisation de la TVA doit être un code ISO 4217."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.TaxCurrencyCode != "" && !model.IsKnownCurrency(d.TaxCurrencyCode) {
				return []rules.Finding{{RuleID: "BR-CL-05", Severity: rules.SeverityError, Term: "BT-6",
					Message: fmt.Sprintf("Devise de comptabilisation inconnue « %s ».", d.TaxCurrencyCode), Actual: d.TaxCurrencyCode}}
			}
			return nil
		},
	}
}

// brDEC15 (BR-DEC-15) : le total de TVA en devise de comptabilisation (BT-111) a au plus 2 décimales.
func brDEC15() rules.Rule {
	return rules.Rule{
		ID: "BR-DEC-15", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-111"},
		Message: map[string]string{"fr": "Le total de TVA en devise de comptabilisation ne doit pas avoir plus de deux décimales."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if a := d.Totals.TaxAmountInAccountingCurrency; a != nil && a.Scale > 2 {
				return []rules.Finding{{RuleID: "BR-DEC-15", Severity: rules.SeverityError, Term: "BT-111",
					Message: fmt.Sprintf("Montant à %d décimales (max 2).", a.Scale), Actual: a.String(), Fixable: true}}
			}
			return nil
		},
	}
}
