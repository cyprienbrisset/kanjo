package pdfa

import (
	"errors"
	"testing"
)

// TestValidatePDFAAbsent vérifie qu'en l'absence de veraPDF, aucune conformité n'est déclarée
// (Checked=false) et que l'erreur sentinelle est renvoyée — jamais de verdict inventé.
func TestValidatePDFAAbsent(t *testing.T) {
	if VeraPDFAvailable() {
		t.Skip("veraPDF est installé : ce test cible le comportement en son absence")
	}
	res, err := ValidatePDFA([]byte("%PDF-1.7"), "3b")
	if !errors.Is(err, ErrVeraPDFAbsent) {
		t.Fatalf("erreur = %v, veut ErrVeraPDFAbsent", err)
	}
	if res.Checked {
		t.Error("Checked doit rester false quand veraPDF est absent")
	}
	if res.Compliant {
		t.Error("Compliant ne doit jamais être vrai sans validation effective")
	}
	if res.Flavour != "3b" {
		t.Errorf("Flavour = %q, veut 3b", res.Flavour)
	}
}

// TestValidatePDFAWhenPresent n'exécute la validation réelle que si veraPDF est disponible.
func TestValidatePDFAWhenPresent(t *testing.T) {
	if !VeraPDFAvailable() {
		t.Skip("veraPDF absent : validation réelle non testée ici")
	}
	base := createBasePDF(t)
	res, err := EmbedXML(base, []byte(`<rsm:CrossIndustryInvoice/>`), "factur-x.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	v, err := ValidatePDFA(res.PDF, "3b")
	if err != nil {
		t.Fatalf("ValidatePDFA: %v", err)
	}
	if !v.Checked {
		t.Error("Checked doit être true quand veraPDF est présent")
	}
	// On ne présume pas du verdict (le PDF de test n'est pas garanti PDF/A-3b) ;
	// on vérifie seulement que la validation a bien été menée.
}
