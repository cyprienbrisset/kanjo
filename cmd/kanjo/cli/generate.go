package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/generate"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func runGenerate(args []string) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	count := fs.Int("count", 10, "nombre de factures à générer")
	format := fs.String("format", "cii", "format : cii|ubl|xrechnung|peppol|json|csv")
	profile := fs.String("profile", "en16931", "profil")
	out := fs.String("out", "", "dossier de sortie (requis)")
	scenario := fs.String("scenario", "simple", "simple|multi-tva|avoir|autoliquidation|intracommunautaire|acompte")
	invalid := fs.Bool("invalid", false, "générer des cas volontairement non conformes")
	seed := fs.Int64("seed", 1, "graine de génération (reproductibilité)")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	if *out == "" {
		errf("generate : --out est requis")
		return ExitUsage
	}
	if write.Get(*format) == nil {
		errf("generate : format %q non pris en charge (cibles : cii, ubl, xrechnung, peppol, json, csv)", *format)
		return ExitUsage
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		errf("generate : %v", err)
		return ExitInternal
	}

	opts := generate.Options{Scenario: generate.Scenario(*scenario), Seed: *seed, Invalid: *invalid}
	wopts := write.Options{Profile: write.Profile(*profile), Indent: true}
	produced := 0
	for i := 0; i < *count; i++ {
		doc, err := generate.Generate(i, opts)
		if err != nil {
			errf("generate #%d : %v", i, err)
			return ExitInternal
		}
		data, err := write.WriteBytes(*format, doc, wopts)
		if err != nil {
			errf("generate #%d : %v", i, err)
			return ExitInternal
		}
		name := sanitizeName(doc.ID) + targetExt(*format)
		if err := fsatomic.WriteFile(filepath.Join(*out, name), data, 0o644); err != nil {
			errf("generate #%d : %v", i, err)
			return ExitInternal
		}
		produced++
	}
	fmt.Fprintf(os.Stdout, "✓ %d factures générées (%s, scénario %s) dans %s\n", produced, *format, *scenario, *out)
	return ExitOK
}

// sanitizeName retire les caractères problématiques d'un nom de fichier.
func sanitizeName(s string) string {
	repl := func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		out = append(out, repl(r))
	}
	if len(out) == 0 {
		return "facture"
	}
	return string(out)
}
