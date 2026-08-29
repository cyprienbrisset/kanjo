package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/pdfa"
)

func runExtract(args []string) int {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	xmlOnly := fs.Bool("xml-only", true, "n'extraire que le XML principal de facture")
	attachments := fs.Bool("attachments", false, "extraire aussi les pièces jointes additionnelles")
	stdout := fs.Bool("stdout", false, "écrire le XML sur la sortie standard")
	out := fs.String("out", "", "dossier de sortie")
	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(inputs) == 0 {
		errf("extract : aucun fichier d'entrée")
		return ExitUsage
	}

	worst := ExitOK
	for _, in := range inputs {
		data, err := readInput(in)
		if err != nil {
			errf("%v", err)
			worst = maxExit(worst, ExitUnreadable)
			continue
		}

		if *attachments && !*xmlOnly {
			files, err := pdfa.ExtractAttachments(data)
			if err != nil {
				worst = maxExit(worst, exitForPDFError(err))
				errf("%s : %v", in, err)
				continue
			}
			for _, f := range files {
				if e := writeExtracted(f.Name, f.Data, in, *out); e != nil {
					errf("%s : %v", in, e)
					worst = maxExit(worst, ExitInternal)
				} else {
					fmt.Fprintf(os.Stdout, "✓ %s → %s\n", in, f.Name)
				}
			}
			continue
		}

		xml, filename, warnCode, err := pdfa.ExtractInvoiceXML(data)
		if err != nil {
			worst = maxExit(worst, exitForPDFError(err))
			errf("%s : %v", in, err)
			continue
		}
		if warnCode != "" {
			errf("%s : %s (nom non conforme, contenu extrait)", in, warnCode)
		}
		if *stdout {
			os.Stdout.Write(xml)
			continue
		}
		if err := writeExtracted(filename, xml, in, *out); err != nil {
			errf("%s : %v", in, err)
			worst = maxExit(worst, ExitInternal)
			continue
		}
		fmt.Fprintf(os.Stdout, "✓ %s → %s\n", in, filename)
	}
	return worst
}

func writeExtracted(name string, data []byte, input, outDir string) error {
	if outDir == "" {
		// À défaut de dossier, écrire à côté du nom de base d'entrée (jamais dans un dossier source
		// implicite : on écrit dans le dossier courant).
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(outDir, filepath.Base(name))
	return fsatomic.WriteFile(path, data, 0o644)
}

func exitForPDFError(err error) int {
	switch {
	case errors.Is(err, pdfa.ErrEncrypted):
		return ExitUnreadable
	case errors.Is(err, pdfa.ErrNoInvoiceXML):
		return ExitUnreadable
	default:
		return ExitUnreadable
	}
}
