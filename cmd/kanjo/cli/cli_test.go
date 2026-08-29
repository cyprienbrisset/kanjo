package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	wcii "github.com/cyprienbrisset/kanjo/pkg/write/cii"
)

// capture exécute fn en redirigeant les sorties standard et renvoie (stdout, stderr, code).
func capture(t *testing.T, fn func() int) (string, string, int) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	code := fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = origOut, origErr
	var bo, be bytes.Buffer
	_, _ = io.Copy(&bo, rOut)
	_, _ = io.Copy(&be, rErr)
	return bo.String(), be.String(), code
}

// writeSampleCII écrit une facture CII cohérente dans un fichier temporaire et renvoie son chemin.
func writeSampleCII(t *testing.T) string {
	t.Helper()
	rate := model.MustParseDecimal("20")
	d := model.NewDocument(model.KindInvoice)
	d.ID = "F2026-0100"
	d.TypeCode = model.TypeCommercialInvoice
	d.IssueDate, _ = model.ParseISO("2026-08-12")
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{Name: "SAS Martin", VATID: "FR12501234567", Address: model.Address{CountryCode: "FR"}}
	d.Buyer = model.Party{Name: "Société Cliente", Address: model.Address{CountryCode: "FR"}}
	due, _ := model.ParseISO("2026-09-11")
	d.DueDate = &due
	d.Lines = []model.Line{{
		ID: "1", Name: "Conseil", Quantity: model.DecimalFromInt(1), UnitCode: model.UnitPiece,
		NetPrice: model.MustParseAmount("100.00", "EUR"), TaxCategory: model.TaxStandard,
		TaxRate: &rate, NetAmount: model.MustParseAmount("100.00", "EUR"),
	}}
	d.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("100.00", "EUR"),
		TaxAmount:     model.MustParseAmount("20.00", "EUR"),
	}}
	d.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("100.00", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("100.00", "EUR"),
		TaxAmount:           model.MustParseAmount("20.00", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("120.00", "EUR"),
		DuePayableAmount:    model.MustParseAmount("120.00", "EUR"),
	}
	b, err := wcii.Write(d, write.Options{Profile: write.ProfileEN16931})
	if err != nil {
		t.Fatalf("préparation CII: %v", err)
	}
	p := filepath.Join(t.TempDir(), "facture.xml")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatalf("écriture fichier: %v", err)
	}
	return p
}

func TestExecuteVersionJSON(t *testing.T) {
	out, _, code := capture(t, func() int { return Execute([]string{"version", "--format", "json"}) })
	if code != ExitOK {
		t.Fatalf("code = %d, veut %d", code, ExitOK)
	}
	if !strings.Contains(out, "\"tool\"") && !strings.Contains(out, "kanjo") {
		t.Errorf("sortie version inattendue : %s", out)
	}
}

func TestExecuteHelpAndNoArgs(t *testing.T) {
	// help explicite.
	out, _, code := capture(t, func() int { return Execute([]string{"help"}) })
	if code != ExitOK || !strings.Contains(out, "kanjo") {
		t.Errorf("help: code=%d out=%q", code, out)
	}
	// Sans argument, sortie non-TTY (pipe) → aide.
	out2, _, code2 := capture(t, func() int { return Execute(nil) })
	if code2 != ExitOK || !strings.Contains(out2, "Commandes") {
		t.Errorf("no-args: code=%d out=%q", code2, out2)
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	_, errOut, code := capture(t, func() int { return Execute([]string{"n-existe-pas"}) })
	if code != ExitUsage {
		t.Errorf("code = %d, veut %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, "commande inconnue") {
		t.Errorf("stderr inattendu : %q", errOut)
	}
}

func TestExecuteConvertToUBL(t *testing.T) {
	in := writeSampleCII(t)
	outFile := filepath.Join(t.TempDir(), "sortie.xml")
	_, errOut, code := capture(t, func() int {
		return Execute([]string{"convert", in, "--to", "ubl", "--out", outFile})
	})
	if code != ExitOK {
		t.Fatalf("code = %d (stderr: %s)", code, errOut)
	}
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("sortie absente: %v", err)
	}
	if f := read.Detect(b); f != read.FormatUBLInvoice {
		t.Errorf("sortie détectée = %s, veut ubl", f)
	}
}

func TestExecuteValidateAndInspect(t *testing.T) {
	in := writeSampleCII(t)
	// validate en JSON : document cohérent → succès.
	_, errOut, code := capture(t, func() int {
		return Execute([]string{"validate", in, "--format", "json"})
	})
	if code != ExitOK {
		t.Fatalf("validate code = %d (stderr: %s)", code, errOut)
	}
	// inspect en JSON.
	out, errOut2, code2 := capture(t, func() int {
		return Execute([]string{"inspect", in, "--format", "json"})
	})
	if code2 != ExitOK {
		t.Fatalf("inspect code = %d (stderr: %s)", code2, errOut2)
	}
	if !strings.Contains(out, "F2026-0100") {
		t.Errorf("inspect ne mentionne pas l'identifiant : %s", out)
	}
}

func TestExecuteUnreadableInput(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "pasxml.xml")
	_ = os.WriteFile(bad, []byte("ceci n'est pas du xml"), 0o644)
	_, _, code := capture(t, func() int {
		return Execute([]string{"validate", bad, "--format", "json"})
	})
	if code == ExitOK {
		t.Error("un fichier illisible ne devrait pas réussir")
	}
}
