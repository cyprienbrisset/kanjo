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
	rules.Register(frMandatoryPaymentNotes())
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

// frMandatoryNoteSubjects associe chaque code sujet BT-21 obligatoire (BR-FR-05) au libellé de
// la mention correspondante, pour construire un message d'anomalie explicite par mention manquante.
var frMandatoryNoteSubjects = []struct {
	code  string
	label string
}{
	{"PMT", "mention relative aux frais de recouvrement en cas de retard de paiement"},
	{"PMD", "mention relative aux pénalités de retard de paiement"},
	{"AAB", "mention relative à l'escompte (ou à son absence)"},
}

// MandatoryPaymentNotes renvoie les 3 notes d'en-tête obligatoires (BR-FR-05/BT-22) dans leur
// libellé standard : frais de recouvrement (PMT), pénalités de retard (PMD) et absence
// d'escompte (AAB). Exporté pour être réutilisé par pkg/generate (corpus de test) et pkg/repair
// (correction d'un document existant) sans dupliquer le texte légal.
func MandatoryPaymentNotes() []model.Note {
	return []model.Note{
		{
			Content:     "En cas de retard de paiement, une indemnité forfaitaire pour frais de recouvrement de 40 € sera exigible (art. L441-10 et D441-5 du code de commerce).",
			SubjectCode: "PMT",
		},
		{
			Content:     "Pénalités de retard : taux d'intérêt légal en vigueur majoré de 10 points, exigibles à compter du jour suivant la date de règlement figurant sur la facture, sans qu'un rappel soit nécessaire.",
			SubjectCode: "PMD",
		},
		{
			Content:     "Pas d'escompte pour paiement anticipé.",
			SubjectCode: "AAB",
		},
	}
}

// frMandatoryPaymentNotes : une facture doit porter, parmi ses notes d'en-tête (BG-1/BT-22), une
// mention pour chacun des codes sujet PMT (frais de recouvrement), PMD (pénalités de retard) et
// AAB (escompte ou absence d'escompte) — sans quoi Chorus Pro / les PDP rejettent le document
// (BR-FR-05/BT-22).
func frMandatoryPaymentNotes() rules.Rule {
	return rules.Rule{
		ID: "FR-BR-05", Set: setFR, Severity: rules.SeverityError,
		Terms:   []string{"BT-22", "BT-21"},
		Message: map[string]string{"fr": "Les mentions obligatoires frais de recouvrement (PMT), pénalités de retard (PMD) et escompte (AAB) doivent figurer dans les notes (BG-1)."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			present := map[string]bool{}
			for _, n := range d.Notes {
				present[n.SubjectCode] = true
			}
			var findings []rules.Finding
			for _, m := range frMandatoryNoteSubjects {
				if present[m.code] {
					continue
				}
				findings = append(findings, rules.Finding{
					RuleID: "FR-BR-05", Severity: rules.SeverityError, Term: "BT-22",
					Message:  "Mention obligatoire absente des notes (BG-1) : " + m.label + " (code " + m.code + ").",
					Expected: m.code, Fixable: true,
				})
			}
			return findings
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
