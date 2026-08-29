package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/api"
	"github.com/cyprienbrisset/kanjo/pkg/convert"
	"github.com/cyprienbrisset/kanjo/pkg/pipeline"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

// stringSlice est un flag répétable (--include a --include b).
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			*s = append(*s, p)
		}
	}
	return nil
}

func runConvert(args []string) int {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	to := fs.String("to", "", "format cible : cii|ubl|facturx|xrechnung|peppol|json (requis)")
	profile := fs.String("profile", "en16931", "profil : minimum|basicwl|basic|en16931|extended")
	syntax := fs.String("syntax", "", "syntaxe pour xrechnung : ubl|cii")
	out := fs.String("out", "", "fichier ou dossier de sortie")
	allowLoss := fs.Bool("allow-loss", false, "autoriser une conversion dégradante")
	maxLoss := fs.String("max-loss", "minor", "politique de perte : none|minor|any")
	format := fs.String("format", "", "format de sortie du rapport : table|json")
	recursive := fs.Bool("recursive", false, "parcours récursif des dossiers")
	workers := fs.Int("workers", 0, "nombre de workers (0 = NumCPU)")
	resume := fs.Bool("resume", false, "reprendre un lot interrompu")
	failFast := fs.Bool("fail-fast", false, "arrêter au premier échec")
	presetName := fs.String("preset", "", "preset à appliquer")
	var include, exclude stringSlice
	fs.Var(&include, "include", "filtre glob d'inclusion (répétable)")
	fs.Var(&exclude, "exclude", "filtre glob d'exclusion (répétable)")

	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}

	// Appliquer un preset : ses valeurs servent de défaut, les flags explicites priment.
	if *presetName != "" {
		set := map[string]bool{}
		fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
		store, serr := openStore()
		if serr != nil {
			errf("convert : %v", serr)
			return ExitInternal
		}
		p, perr := store.Load(*presetName)
		if perr != nil {
			errf("convert : %v", perr)
			return ExitUnreadable
		}
		if !set["to"] && p.To != "" {
			*to = p.To
		}
		if !set["profile"] && p.Profile != "" {
			*profile = p.Profile
		}
		if !set["syntax"] && p.Syntax != "" {
			*syntax = p.Syntax
		}
		if !set["max-loss"] && p.MaxLoss != "" {
			*maxLoss = p.MaxLoss
		}
	}

	if *to == "" {
		errf("convert : --to est requis")
		return ExitUsage
	}
	if len(inputs) == 0 {
		errf("convert : aucun fichier d'entrée")
		return ExitUsage
	}

	opts := convert.Options{
		To:        *to,
		Profile:   write.Profile(*profile),
		Syntax:    *syntax,
		AllowLoss: *allowLoss,
		MaxLoss:   convert.MaxLoss(*maxLoss),
	}

	// Découverte des fichiers (fichiers, dossiers, globs, filtres).
	files, err := pipeline.Discover(inputs, *recursive, include, exclude)
	if err != nil {
		errf("convert : %v", err)
		return ExitUsage
	}
	if len(files) == 0 {
		errf("convert : aucun fichier à traiter après filtrage")
		return ExitUnreadable
	}

	outIsDir := len(files) > 1 || isDirPath(*out)

	// Reprise : filtre les fichiers déjà traités.
	var state *pipeline.State
	if *resume && *out != "" {
		statePath := filepath.Join(resumeDir(*out, outIsDir), ".kanjo-resume.log")
		if st, err := pipeline.LoadState(statePath); err == nil {
			state = st
			defer state.Close()
			files = state.Filter(files)
		}
	}

	start := time.Now()
	env := api.NewEnvelope("convert", start.UTC().Format(time.RFC3339))

	proc := func(path string) api.Result {
		res := api.Result{Input: path}
		data, err := readInput(path)
		if err != nil {
			res.Status, res.Error = api.StatusError, err.Error()
			return res
		}
		cr, err := convert.Convert(data, path, opts)
		if err != nil {
			res.Status, res.Error = api.StatusError, err.Error()
			if cr != nil {
				res.Format, res.Profile, res.Losses = string(cr.InputFormat), cr.Profile, cr.Losses
			}
			return res
		}
		res.Format, res.Profile, res.Losses = string(cr.InputFormat), cr.Profile, cr.Losses
		res.Hashes = &api.Hashes{InputSha256: sha256hex(data), OutputSha256: sha256hex(cr.Output)}

		outPath, werr := writeOutput(cr.Output, path, *out, *to, outIsDir)
		if werr != nil {
			res.Status, res.Error = api.StatusError, werr.Error()
			return res
		}
		res.Output = outPath
		if len(cr.Losses) > 0 {
			res.Status = api.StatusWarning
		} else {
			res.Status = api.StatusOK
		}
		if state != nil {
			_ = state.MarkDone(path)
		}
		return res
	}

	rep := pipeline.Run(files, proc, pipeline.Options{Workers: *workers, FailFast: *failFast})
	env.Results = rep.Results
	env.Summary = rep.Summary
	env.DurationMs = time.Since(start).Milliseconds()

	recordAudit(env, "convert")
	renderConvert(env, outputFormat(*format))
	return exitFromSummary(env)
}

// exitFromSummary déduit le code de sortie du lot : le pire cas rencontré.
func exitFromSummary(env *api.Envelope) int {
	worst := ExitOK
	for _, r := range env.Results {
		if r.Status == api.StatusError {
			// Distinguer les pertes refusées des autres erreurs pour le code de sortie.
			if strings.Contains(r.Error, convert.ErrLossExceedsPolicy.Error()) {
				worst = maxExit(worst, ExitLossRefused)
			} else if strings.Contains(r.Error, read.ErrUnsupportedFormat.Error()) {
				worst = maxExit(worst, ExitUnreadable)
			} else {
				worst = maxExit(worst, ExitUnreadable)
			}
		}
	}
	return worst
}

// resumeDir renvoie le dossier où placer le journal de reprise.
func resumeDir(out string, outIsDir bool) string {
	if outIsDir {
		return out
	}
	return filepath.Dir(out)
}

// writeOutput écrit la sortie. Renvoie le chemin écrit, ou écrit sur stdout si --out est vide
// et qu'il n'y a qu'une entrée (chemin renvoyé vide).
func writeOutput(data []byte, input, out, target string, outIsDir bool) (string, error) {
	if out == "" {
		if _, err := os.Stdout.Write(data); err != nil {
			return "", err
		}
		return "", nil
	}
	var path string
	if outIsDir {
		if err := os.MkdirAll(out, 0o755); err != nil {
			return "", fmt.Errorf("création du dossier %s: %w", out, err)
		}
		path = filepath.Join(out, deriveName(input, target))
	} else {
		if dir := filepath.Dir(out); dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
		path = out
	}
	if err := fsatomic.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// deriveName construit un nom de sortie à partir du nom d'entrée et de l'extension cible.
func deriveName(input, target string) string {
	base := filepath.Base(input)
	if input == "-" {
		base = "stdin"
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return stem + targetExt(target)
}

// targetExt renvoie l'extension de fichier associée à un format cible.
func targetExt(target string) string {
	switch target {
	case "facturx", "pdf":
		return ".pdf"
	case "json":
		return ".json"
	case "csv":
		return ".csv"
	case "xlsx":
		return ".xlsx"
	default: // cii, ubl, xrechnung, peppol
		return ".xml"
	}
}

func isDirPath(p string) bool {
	if p == "" {
		return false
	}
	if strings.HasSuffix(p, string(os.PathSeparator)) || strings.HasSuffix(p, "/") {
		return true
	}
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func maxExit(a, b int) int {
	if b > a {
		return b
	}
	return a
}

func renderConvert(env *api.Envelope, format string) {
	if format == "json" {
		printJSON(env)
		return
	}
	for _, r := range env.Results {
		switch r.Status {
		case api.StatusOK:
			fmt.Fprintf(os.Stdout, "✓ %s → %s\n", r.Input, r.Output)
		case api.StatusWarning:
			fmt.Fprintf(os.Stdout, "⚠ %s → %s (%d perte(s))\n", r.Input, r.Output, len(r.Losses))
			for _, l := range r.Losses {
				fmt.Fprintf(os.Stdout, "    %s %s\n", l.Code, l.Message)
			}
		case api.StatusError:
			fmt.Fprintf(os.Stdout, "✗ %s : %s\n", r.Input, r.Error)
		}
	}
	s := env.Summary
	fmt.Fprintf(os.Stdout, "\n%d converties · %d avec perte · %d échecs\n", s.OK, s.Warning, s.Error)
}
