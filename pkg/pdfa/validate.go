package pdfa

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrVeraPDFAbsent indique que l'outil veraPDF n'est pas installé : la conformité PDF/A ne peut
// pas être validée et n'est donc JAMAIS déclarée (principe §17.7 « ne jamais mentir »).
var ErrVeraPDFAbsent = errors.New("pdfa: veraPDF absent, conformité PDF/A non vérifiable")

// PDFAValidation est le résultat d'une validation PDF/A effective.
type PDFAValidation struct {
	Checked   bool   `json:"checked"`           // true : veraPDF a réellement été exécuté
	Compliant bool   `json:"compliant"`         // conforme au profil demandé
	Flavour   string `json:"flavour"`           // profil visé, ex. "3b"
	Tool      string `json:"tool,omitempty"`    // outil utilisé
	Details   string `json:"details,omitempty"` // extrait du rapport
}

// VeraPDFAvailable indique si l'outil veraPDF est présent dans le PATH.
func VeraPDFAvailable() bool {
	_, err := exec.LookPath("verapdf")
	return err == nil
}

// ValidatePDFA valide la conformité PDF/A d'un PDF via veraPDF si l'outil est présent.
// Le profil par défaut est "3b" (celui des factures Factur-X). En l'absence de veraPDF, renvoie
// ErrVeraPDFAbsent avec Checked=false : Kanjō ne déclare jamais une conformité non calculée.
func ValidatePDFA(pdf []byte, flavour string) (PDFAValidation, error) {
	if flavour == "" {
		flavour = "3b"
	}
	res := PDFAValidation{Flavour: flavour}
	bin, err := exec.LookPath("verapdf")
	if err != nil {
		return res, ErrVeraPDFAbsent
	}

	f, err := os.CreateTemp("", "kanjo-pdfa-*.pdf")
	if err != nil {
		return res, fmt.Errorf("pdfa: fichier temporaire: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(pdf); err != nil {
		_ = f.Close()
		return res, fmt.Errorf("pdfa: écriture temporaire: %w", err)
	}
	if err := f.Close(); err != nil {
		return res, fmt.Errorf("pdfa: fermeture temporaire: %w", err)
	}

	// veraPDF renvoie un code de sortie non nul en cas de non-conformité ; on s'appuie sur la
	// sortie texte (« PASS »/« FAIL ») plutôt que sur le code seul.
	out, _ := exec.Command(bin, "--flavour", flavour, "--format", "text", f.Name()).CombinedOutput()
	text := string(out)
	res.Checked = true
	res.Tool = "veraPDF"
	res.Compliant = strings.Contains(text, "PASS")
	res.Details = strings.TrimSpace(firstLine(text))
	return res, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
