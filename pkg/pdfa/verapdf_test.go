package pdfa

import (
	"os"
	"os/exec"
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
	// L'embed procède désormais par mise à jour INCRÉMENTALE : les octets du PDF de base sont
	// préservés (préfixe exact), donc sa conformité PDF/A-3b doit être conservée. veraPDF en est
	// l'arbitre : la sortie doit rester conforme.
	t.Logf("conformité PDF/A-3b après embed incrémental : compliant=%v — %s", val.Compliant, val.Details)
	if !val.Compliant {
		t.Logf("RAPPORT veraPDF détaillé :\n%s", veraPDFVerbose(t, res.PDF))
		t.Errorf("l'embed incrémental doit PRÉSERVER la conformité PDF/A-3b du PDF de base — veraPDF : %s", val.Details)
	}
	// Le PDF d'origine doit être le préfixe exact de la sortie (preuve de non-réécriture).
	if len(res.PDF) < len(base) || string(res.PDF[:len(base)]) != string(base) {
		t.Error("les octets du PDF de base ne sont pas préservés à l'identique (préfixe)")
	}
}

// veraPDFVerbose exécute veraPDF avec un rapport détaillé (règles échouées) pour diagnostic.
func veraPDFVerbose(t *testing.T, pdf []byte) string {
	t.Helper()
	bin, err := exec.LookPath("verapdf")
	if err != nil {
		return "(veraPDF absent)"
	}
	f, err := os.CreateTemp("", "kanjo-diag-*.pdf")
	if err != nil {
		return err.Error()
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.Write(pdf)
	_ = f.Close()
	// --format xml : rapport détaillé (description de la règle + contexte de l'objet fautif).
	out, _ := exec.Command(bin, "--flavour", "3b", "--format", "xml", f.Name()).CombinedOutput()
	return string(out)
}
