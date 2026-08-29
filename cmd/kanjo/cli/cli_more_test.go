package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteGenerate(t *testing.T) {
	dir := t.TempDir()
	_, errOut, code := capture(t, func() int {
		return Execute([]string{"generate", "--count", "2", "--format", "cii", "--seed", "42", "--out", dir})
	})
	if code != ExitOK {
		t.Fatalf("generate code = %d (stderr: %s)", code, errOut)
	}
	n, _ := filepath.Glob(filepath.Join(dir, "*.xml"))
	if len(n) != 2 {
		t.Errorf("attendu 2 fichiers générés, obtenu %d", len(n))
	}
}

func TestExecuteDoctor(t *testing.T) {
	out, _, code := capture(t, func() int { return Execute([]string{"doctor", "--format", "json"}) })
	if code != ExitOK {
		t.Fatalf("doctor code = %d", code)
	}
	if !strings.Contains(out, "verapdf") && !strings.Contains(out, "capabilities") {
		t.Errorf("doctor JSON inattendu : %s", out)
	}
}

func TestExecuteDiffIdentical(t *testing.T) {
	f := writeSampleCII(t)
	_, errOut, code := capture(t, func() int { return Execute([]string{"diff", f, f, "--format", "json"}) })
	if code != ExitOK {
		t.Fatalf("diff (identique) code = %d (stderr: %s)", code, errOut)
	}
}

func TestExecuteRender(t *testing.T) {
	f := writeSampleCII(t)
	dir := t.TempDir()
	_, errOut, code := capture(t, func() int { return Execute([]string{"render", f, "--out", dir}) })
	if code != ExitOK {
		t.Fatalf("render code = %d (stderr: %s)", code, errOut)
	}
	html, _ := filepath.Glob(filepath.Join(dir, "*.html"))
	if len(html) == 0 {
		t.Error("aucun HTML produit")
	}
}

func TestExecuteAnonymize(t *testing.T) {
	f := writeSampleCII(t)
	dir := t.TempDir()
	_, errOut, code := capture(t, func() int { return Execute([]string{"anonymize", f, "--out", dir}) })
	if code != ExitOK {
		t.Fatalf("anonymize code = %d (stderr: %s)", code, errOut)
	}
	xml, _ := filepath.Glob(filepath.Join(dir, "*.xml"))
	if len(xml) == 0 {
		t.Error("aucun fichier anonymisé produit")
	}
}

func TestExecuteRepairDryRun(t *testing.T) {
	f := writeSampleCII(t)
	_, errOut, code := capture(t, func() int { return Execute([]string{"repair", f, "--dry-run"}) })
	if code != ExitOK {
		t.Fatalf("repair --dry-run code = %d (stderr: %s)", code, errOut)
	}
	// En dry-run, l'entrée ne doit pas être modifiée ni sauvegardée (.bak).
	if _, err := os.Stat(f + ".bak"); err == nil {
		t.Error("repair --dry-run ne doit pas créer de .bak")
	}
}
