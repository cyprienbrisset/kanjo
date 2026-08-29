package anonymize_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/anonymize"
	"github.com/cyprienbrisset/kanjo/pkg/generate"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

func sample() *model.Document {
	d, _ := generate.Generate(0, generate.Options{Scenario: generate.ScenarioSimple, Seed: 99})
	d.Seller.Name = "SAS Martin Réel"
	d.Seller.VATID = "FR12501234567"
	d.PaymentInstructions = &model.PaymentInstructions{
		CreditTransfers: []model.CreditTransfer{{IBAN: "FR7630006000011234567890189", AccountName: "Compte Réel"}},
	}
	return d
}

func TestAnonymizeRemovesPII(t *testing.T) {
	d := sample()
	origSeller := d.Seller.Name
	origIBAN := d.PaymentInstructions.CreditTransfers[0].IBAN

	anonymize.Anonymize(d, anonymize.Options{Seed: "abc"})

	if d.Seller.Name == origSeller {
		t.Error("le nom du vendeur n'a pas été anonymisé")
	}
	if d.Seller.VATID == "FR12501234567" {
		t.Error("le n° de TVA n'a pas été anonymisé")
	}
	iban := d.PaymentInstructions.CreditTransfers[0].IBAN
	if iban == origIBAN {
		t.Error("l'IBAN n'a pas été anonymisé")
	}
	if d.PaymentInstructions.CreditTransfers[0].AccountName != "" {
		t.Error("le nom du compte n'a pas été effacé")
	}
	if !validIBAN(iban) {
		t.Errorf("l'IBAN synthétique %q est invalide (modulo 97)", iban)
	}
}

func TestAnonymizeKeepsCoherence(t *testing.T) {
	d := sample()
	anonymize.Anonymize(d, anonymize.Options{Seed: "abc"})
	rep := rules.Validate(d, "en16931")
	if rep.HasErrors() {
		t.Errorf("le document anonymisé n'est plus conforme : %+v", rep.Findings)
	}
}

func TestAnonymizeDeterministic(t *testing.T) {
	a, b := sample(), sample()
	anonymize.Anonymize(a, anonymize.Options{Seed: "graine"})
	anonymize.Anonymize(b, anonymize.Options{Seed: "graine"})
	if a.Seller.Name != b.Seller.Name || a.Seller.VATID != b.Seller.VATID {
		t.Error("anonymisation non déterministe pour un même seed")
	}
}

// validIBAN reprend le contrôle ISO 13616 modulo 97 pour vérifier les IBAN générés.
func validIBAN(iban string) bool {
	if len(iban) < 15 {
		return false
	}
	r := iban[4:] + iban[:4]
	rem := 0
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case c >= '0' && c <= '9':
			rem = (rem*10 + int(c-'0')) % 97
		case c >= 'A' && c <= 'Z':
			rem = (rem*100 + int(c-'A') + 10) % 97
		default:
			return false
		}
	}
	return rem == 1
}
