// Package fr implémente la CIUS française : mentions obligatoires de la facturation
// électronique (SIREN, identification des parties). Jeu de règles "cius.fr".
package fr

import (
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

const setFR = "cius.fr"

func init() {
	rules.Register(frSellerIdentified())
	rules.Register(frSirenFormat())
}

// frSellerIdentified : un vendeur établi en France doit être identifié par un n° de TVA
// intracommunautaire ou un SIREN/SIRET (mention obligatoire CTC).
func frSellerIdentified() rules.Rule {
	return rules.Rule{
		ID: "FR-CTC-01", Set: setFR, Severity: rules.SeverityError,
		Terms:   []string{"BT-31", "BT-30"},
		Message: map[string]string{"fr": "Un vendeur français doit être identifié par un n° de TVA ou un SIREN/SIRET."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !strings.EqualFold(d.Seller.Address.CountryCode, "FR") {
				return nil
			}
			if d.Seller.VATID != "" || sellerSIREN(d) != "" {
				return nil
			}
			return []rules.Finding{{
				RuleID: "FR-CTC-01", Severity: rules.SeverityError, Term: "BT-30",
				Message: "Vendeur français sans identification (ni n° de TVA, ni SIREN/SIRET).",
			}}
		},
	}
}

// frSirenFormat : un SIREN renseigné doit comporter exactement 9 chiffres.
func frSirenFormat() rules.Rule {
	return rules.Rule{
		ID: "FR-SIREN-01", Set: setFR, Severity: rules.SeverityError,
		Terms:   []string{"BT-30"},
		Message: map[string]string{"fr": "Un SIREN doit comporter exactement 9 chiffres."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			siren := sellerSIREN(d)
			if siren == "" || isNDigits(siren, 9) {
				return nil
			}
			return []rules.Finding{{
				RuleID: "FR-SIREN-01", Severity: rules.SeverityError, Term: "BT-30",
				Message: "Le SIREN du vendeur n'a pas 9 chiffres.", Actual: siren, Fixable: true,
			}}
		},
	}
}

// sellerSIREN extrait le SIREN du vendeur : d'abord des extensions FR, sinon dérivé de
// l'identifiant légal (SIRET à 14 chiffres → 9 premiers, ou SIREN à 9 chiffres).
func sellerSIREN(d *model.Document) string {
	if d.Extensions.FR != nil && d.Extensions.FR.SellerSIREN != "" {
		return digitsOnly(d.Extensions.FR.SellerSIREN)
	}
	legal := digitsOnly(d.Seller.LegalID)
	switch len(legal) {
	case 14:
		return legal[:9]
	case 9:
		return legal
	default:
		return ""
	}
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isNDigits(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for i := 0; i < n; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
