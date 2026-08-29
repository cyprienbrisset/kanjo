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

	if outputFormat(*format) == "json" {
		printJSON(map[string]any{
			"schemaVersion": "github.com/cyprienbrisset/kanjo/1",
			"command":       "embed",
			"output":        outPath,
			"attachedAs":    res.AttachedAs,
			"pdfaChecked":   res.PDFAChecked, // jamais simulé (§17.7)
			"warnings":      res.Warnings,
		})
		return ExitOK
	}
	fmt.Fprintf(os.Stdout, "✓ %s ← %s embarqué (%s)\n", outPath, res.AttachedAs, filepath.Base(*xmlPath))
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stdout, "   %s\n", w)
	}
	fmt.Fprintln(os.Stdout, "   conformité PDF/A-3b : non vérifiée (veraPDF absent)")
	return ExitOK
}
