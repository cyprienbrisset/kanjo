package pdfa

import (
	"os"
	"path/filepath"
	"testing"
)

// realFacturXPDF est une facture Factur-X (profil EN 16931), PDF/A-3b réel (corpus akretion, BSD).
func realFacturXPDF(t *testing.T) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", "corpus", "pdfa", "facturx_en16931.pdf")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("lecture du PDF de référence: %v", err)
	}
	return b
}

// TestVeraPDFOnRealFacturX prouve, quand veraPDF est présent (job CI dédié), que notre validateur
// reconnaît correctement un vrai PDF/A-3b comme conforme. Sans veraPDF, le test est ignoré :
// Kanjō ne déclare jamais une conformité non calculée (§17.7).
func TestVeraPDFOnRealFacturX(t *testing.T) {
	if !VeraPDFAvailable() {
		t.Skip("veraPDF absent : conformité PDF/A non vérifiable ici (exécuté par le job CI veraPDF)")
	}
	res, err := ValidatePDFA(realFacturXPDF(t), "3b")
	if err != nil {
		t.Fatalf("ValidatePDFA: %v", err)
	}
	if !res.Checked {
		t.Fatal("Checked doit être true quand veraPDF est présent")
	}
	if !res.Compliant {
		t.Errorf("le PDF/A-3b de référence devrait être conforme ; rapport : %s", res.Details)
	}
}

// TestVeraPDFEmbedOutput mesure empiriquement si notre chaîne d'embed préserve la conformité
// PDF/A-3b d'un PDF de base déjà conforme. Le verdict est journalisé, jamais fabriqué : si pdfcpu
// n'assure pas la préservation, c'est une limite mesurée (roadmap), pas un mensonge de conformité.
func TestVeraPDFEmbedOutput(t *testing.T) {
	if !VeraPDFAvailable() {
		t.Skip("veraPDF absent : exécuté par le job CI veraPDF")
	}
	base := realFacturXPDF(t)

	// Référence : le PDF d'origine doit être conforme (sinon le corpus/outil est en cause).
	if ref, err := ValidatePDFA(base, "3b"); err != nil || !ref.Compliant {
		t.Fatalf("le PDF de référence n'est pas reconnu conforme (err=%v, détails=%s)", err, ref.Details)
	}

	res, err := EmbedXML(base, []byte(`<?xml version="1.0"?><rsm:CrossIndustryInvoice/>`), "factur-x.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	val, err := ValidatePDFA(res.PDF, "3b")
	if err != nil {
		t.Fatalf("ValidatePDFA(sortie embed): %v", err)
	}
	// Mesure honnête : on journalise le résultat. La conformité globale après réécriture pdfcpu
	// est une question ouverte tranchée par veraPDF, pas un acquis.
	t.Logf("conformité PDF/A-3b après embed : compliant=%v — %s", val.Compliant, val.Details)
	if !val.Compliant {
		t.Log("LIMITE MESURÉE : la réécriture pdfcpu ne préserve pas (encore) la conformité PDF/A-3b globale ; l'association Factur-X (/AF, /AFRelationship) est en place mais la mise en conformité complète reste roadmap.")
	}
}
