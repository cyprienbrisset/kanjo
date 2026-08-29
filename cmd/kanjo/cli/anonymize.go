package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/anonymize"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func runAnonymize(args []string) int {
	fs := flag.NewFlagSet("anonymize", flag.ContinueOnError)
	seed := fs.String("seed", "", "graine pour un remplacement déterministe")
	out := fs.String("out", "", "dossier de sortie (défaut : à côté, suffixe .anon)")
	format := fs.String("format", "", "format de sortie (défaut : identique à l'entrée)")
	var keep stringSlice
	fs.Var(&keep, "keep", "BT/aspects à préserver (ex. amounts) — répétable")
	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(inputs) == 0 {
		errf("anonymize : aucun fichier d'entrée")
		return ExitUsage
	}

	opts := anonymize.Options{Seed: *seed}
	opts.SetKeep(keep)

	worst := ExitOK
	for _, in := range inputs {
		data, err := readInput(in)
		if err != nil {
			errf("%v", err)
			worst = maxExit(worst, ExitUnreadable)
			continue
		}
		rd, err := read.ReadBytes(data, in)
		if err != nil {
			errf("%v", err)
			worst = maxExit(worst, ExitUnreadable)
			continue
		}
		anonymize.Anonymize(rd.Doc, opts)

		target := *format
		if target == "" {
			target = formatToTarget(rd.Format)
		}
		outData, err := write.WriteBytes(target, rd.Doc, write.Options{Profile: write.ProfileEN16931, Indent: true})
		if err != nil {
			errf("anonymize %s : %v", in, err)
			worst = maxExit(worst, ExitInternal)
			continue
		}
		outPath := anonOutPath(in, *out, target)
		if err := fsatomic.WriteFile(outPath, outData, 0o644); err != nil {
			errf("anonymize %s : %v", in, err)
			worst = maxExit(worst, ExitInternal)
			continue
		}
		fmt.Fprintf(os.Stdout, "✓ %s → %s (anonymisé)\n", in, outPath)
	}
	return worst
}

func anonOutPath(input, outDir, target string) string {
	base := sanitizeName(trimExt(filepath.Base(input))) + ".anon" + targetExt(target)
	if outDir != "" {
		_ = os.MkdirAll(outDir, 0o755)
		return filepath.Join(outDir, base)
	}
	return filepath.Join(filepath.Dir(input), base)
}

func trimExt(name string) string {
	return name[:len(name)-len(filepath.Ext(name))]
}

// formatToTarget associe un format de lecture à une cible d'écriture équivalente.
func formatToTarget(f read.Format) string {
	switch f {
	case read.FormatUBLInvoice, read.FormatUBLCreditNote:
		return "ubl"
	case read.FormatKanjoJSON:
		return "json"
	case read.FormatFacturX, read.FormatCII:
		return "cii"
	default:
		return "json"
	}
}
