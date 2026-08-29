package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/pdfa"
)

func runEmbed(args []string) int {
	fs := flag.NewFlagSet("embed", flag.ContinueOnError)
	xmlPath := fs.String("xml", "", "XML de facture à embarquer (requis)")
	out := fs.String("out", "", "PDF de sortie")
	profile := fs.String("profile", "en16931", "profil déclaré (informatif)")
	name := fs.String("name", "factur-x.xml", "nom de la pièce jointe embarquée")
	format := fs.String("format", "", "format du rapport : table|json")
	verifyPDFA := fs.Bool("verify-pdfa", false, "valider la conformité PDF/A-3b avec veraPDF (si installé)")
	_ = profile
	positionals, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if *xmlPath == "" {
		errf("embed : --xml est requis")
		return ExitUsage
	}
	if len(positionals) != 1 {
		errf("embed : exactement un PDF d'entrée est requis")
		return ExitUsage
	}
	pdfIn := positionals[0]

	pdfData, err := readInput(pdfIn)
	if err != nil {
		errf("%v", err)
		return ExitUnreadable
	}
	xmlData, err := readInput(*xmlPath)
	if err != nil {
		errf("%v", err)
		return ExitUnreadable
	}

	res, err := pdfa.EmbedXML(pdfData, xmlData, *name)
	if err != nil {
		errf("embed : %v", err)
		return ExitUnreadable
	}

	outPath := *out
	if outPath == "" {
		base := strings.TrimSuffix(filepath.Base(pdfIn), filepath.Ext(pdfIn))
		outPath = base + ".facturx.pdf"
	}
	if err := fsatomic.WriteFile(outPath, res.PDF, 0o644); err != nil {
		errf("embed : écriture de %s : %v", outPath, err)
		return ExitInternal
	}

	// Validation PDF/A optionnelle et effective (jamais simulée, §17.7).
	var pdfaVal *pdfa.PDFAValidation
	if *verifyPDFA {
		if v, err := pdfa.ValidatePDFA(res.PDF, "3b"); err == nil {
			pdfaVal = &v
		} else {
			v.Details = "veraPDF absent"
			pdfaVal = &v
		}
	}

	if outputFormat(*format) == "json" {
		payload := map[string]any{
			"schemaVersion": "github.com/cyprienbrisset/kanjo/1",
			"command":       "embed",
			"output":        outPath,
			"attachedAs":    res.AttachedAs,
			"pdfaChecked":   res.PDFAChecked, // jamais simulé (§17.7)
			"warnings":      res.Warnings,
		}
		if pdfaVal != nil {
			payload["pdfaValidation"] = pdfaVal
		}
		printJSON(payload)
		return ExitOK
	}
	fmt.Fprintf(os.Stdout, "✓ %s ← %s embarqué (%s)\n", outPath, res.AttachedAs, filepath.Base(*xmlPath))
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stdout, "   %s\n", w)
	}
	switch {
	case pdfaVal == nil:
		fmt.Fprintln(os.Stdout, "   conformité PDF/A-3b : non vérifiée (utilisez --verify-pdfa)")
	case !pdfaVal.Checked:
		fmt.Fprintln(os.Stdout, "   conformité PDF/A-3b : non vérifiable (veraPDF absent)")
	case pdfaVal.Compliant:
		fmt.Fprintln(os.Stdout, "   conformité PDF/A-3b : ✓ conforme (veraPDF)")
	default:
		fmt.Fprintln(os.Stdout, "   conformité PDF/A-3b : ✗ non conforme (veraPDF)")
	}
	return ExitOK
}
