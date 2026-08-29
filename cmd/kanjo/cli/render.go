package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/render"
	"github.com/cyprienbrisset/kanjo/pkg/rules"

	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

func runRender(args []string) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	to := fs.String("to", "html", "format de rendu : html")
	lang := fs.String("lang", "fr", "langue : fr|en|de")
	out := fs.String("out", "", "dossier de sortie (défaut : à côté de l'entrée)")
	seal := fs.Bool("seal", false, "apposer un sceau reflétant la validation")
	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(inputs) == 0 {
		errf("render : aucun fichier d'entrée")
		return ExitUsage
	}
	if *to != "html" {
		errf("render : seul --to html est disponible (le PDF passe par un moteur externe, à venir)")
		return ExitCapability
	}

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
		sealGlyph := ""
		if *seal {
			sealGlyph = sealFromValidation(rd.Doc)
		}
		html, err := render.RenderInvoiceHTML(rd.Doc, model.Lang(*lang), sealGlyph)
		if err != nil {
			errf("render %s : %v", in, err)
			worst = maxExit(worst, ExitInternal)
			continue
		}
		outPath := renderOutPath(in, *out)
		if err := fsatomic.WriteFile(outPath, html, 0o644); err != nil {
			errf("render %s : %v", in, err)
			worst = maxExit(worst, ExitInternal)
			continue
		}
		fmt.Fprintf(os.Stdout, "✓ %s → %s\n", in, outPath)
	}
	return worst
}

// sealFromValidation calcule le verdict reflétant une validation réellement calculée (§17.7).
func sealFromValidation(doc *model.Document) string {
	rep := rules.Validate(doc)
	switch {
	case rep.HasErrors():
		return "error"
	case len(rep.Findings) > 0:
		return "warning"
	default:
		return "ok"
	}
}

func renderOutPath(input, outDir string) string {
	base := sanitizeName(trimExt(filepath.Base(input))) + ".html"
	if outDir != "" {
		_ = os.MkdirAll(outDir, 0o755)
		return filepath.Join(outDir, base)
	}
	return filepath.Join(filepath.Dir(input), base)
}
