package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/internal/version"
	"github.com/cyprienbrisset/kanjo/pkg/api"
	"github.com/cyprienbrisset/kanjo/pkg/crossvalidate"
	"github.com/cyprienbrisset/kanjo/pkg/pipeline"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/render"
	"github.com/cyprienbrisset/kanjo/pkg/rules"

	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all" // enregistrement de tous les jeux de règles (en16931, cius.fr, kanjo)
)

func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	ruleSets := fs.String("rules", "", "jeux de règles : en16931,cius.fr,kanjo (défaut : tous)")
	severity := fs.String("severity", "error", "seuil d'échec : error|warning")
	format := fs.String("format", "", "format de sortie : table|json")
	explain := fs.Bool("explain", false, "afficher le texte intégral des règles échouées")
	recursive := fs.Bool("recursive", false, "parcours récursif des dossiers")
	workers := fs.Int("workers", 0, "nombre de workers (0 = NumCPU)")
	report := fs.String("report", "", "écrire un rapport (.html|.json)")
	crossCheck := fs.Bool("cross-check", false, "confronter le verdict aux validateurs externes configurés (Mustangproject/KoSIT)")
	var include, exclude stringSlice
	fs.Var(&include, "include", "filtre glob d'inclusion (répétable)")
	fs.Var(&exclude, "exclude", "filtre glob d'exclusion (répétable)")
	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(inputs) == 0 {
		errf("validate : aucun fichier d'entrée")
		return ExitUsage
	}

	var sets []string
	if s := strings.TrimSpace(*ruleSets); s != "" {
		for _, part := range strings.Split(s, ",") {
			if p := strings.TrimSpace(part); p != "" {
				sets = append(sets, p)
			}
		}
	}
	// Fail-closed : refuser d'emblée un jeu de règles inexistant plutôt que de valider avec un jeu
	// vide (qui donnerait un faux « conforme »). Message clair avant tout traitement (§17.7).
	if unknown := rules.UnknownSets(sets...); len(unknown) > 0 {
		errf("validate : jeu(x) de règles inconnu(s) : %s (disponibles : %s)",
			strings.Join(unknown, ", "), strings.Join(rules.Sets(), ", "))
		return ExitUsage
	}
	threshold := rules.SeverityError
	if *severity == "warning" {
		threshold = rules.SeverityWarning
	}

	files, err := pipeline.Discover(inputs, *recursive, include, exclude)
	if err != nil {
		errf("validate : %v", err)
		return ExitUsage
	}
	if len(files) == 0 {
		errf("validate : aucun fichier à valider")
		return ExitUnreadable
	}

	start := time.Now()
	env := api.NewEnvelope("validate", start.UTC().Format(time.RFC3339))

	proc := func(path string) api.Result {
		res := api.Result{Input: path}
		data, err := readInput(path)
		if err != nil {
			res.Status, res.Error = api.StatusError, err.Error()
			return res
		}
		rd, err := read.ReadBytes(data, path)
		if err != nil {
			res.Status, res.Error = api.StatusError, err.Error()
			return res
		}
		res.Format, res.Profile = string(rd.Format), rd.Profile
		rep := rules.Validate(rd.Doc, sets...)
		hasBlocking := false
		for _, f := range rep.Findings {
			res.Findings = append(res.Findings, api.Finding{
				RuleID: f.RuleID, Severity: f.Severity.String(), Message: f.Message,
				Term: f.Term, Path: f.Path, Expected: f.Expected, Actual: f.Actual, Fixable: f.Fixable,
			})
			if f.Severity >= threshold {
				hasBlocking = true
			}
		}
		switch {
		case hasBlocking:
			res.Status = api.StatusError
		case len(rep.Findings) > 0:
			res.Status = api.StatusWarning
		default:
			res.Status = api.StatusOK
		}
		return res
	}

	rep := pipeline.Run(files, proc, pipeline.Options{Workers: *workers})
	env.Results = rep.Results
	env.Summary = rep.Summary
	env.DurationMs = time.Since(start).Milliseconds()

	worstExit := ExitOK
	for _, r := range env.Results {
		if r.Status == api.StatusError {
			worstExit = maxExit(worstExit, ExitValidation)
		}
	}

	recordAudit(env, "validate")

	if *report != "" {
		if err := writeReport(*report, env); err != nil {
			errf("validate : rapport : %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "rapport écrit : %s\n", *report)
		}
	}

	if outputFormat(*format) == "json" {
		printJSON(env)
		if *crossCheck {
			runCrossCheck(env)
		}
		return worstExit
	}
	renderValidate(env, *explain)
	if *crossCheck {
		runCrossCheck(env)
	}
	return worstExit
}

// runCrossCheck confronte le verdict de Kanjō aux validateurs externes configurés (§ dossier
// d'agrément). Purement informatif : n'altère pas le code de sortie. Ne simule jamais un verdict :
// si aucun outil n'est configuré, il le dit clairement (§17.7).
func runCrossCheck(env *api.Envelope) {
	avail := crossvalidate.Available()
	fmt.Fprintln(os.Stdout, "\n▸ Validation croisée")
	if len(avail) == 0 {
		fmt.Fprintln(os.Stdout, "  aucun validateur externe configuré "+
			"(définir KANJO_MUSTANG_JAR et/ou KANJO_KOSIT_JAR + KANJO_KOSIT_SCENARIOS ; java requis).")
		return
	}
	totalDisagree := 0
	for _, r := range env.Results {
		verdicts := crossvalidate.Run(r.Input)
		if len(verdicts) == 0 {
			continue
		}
		kanjoCompliant := r.Status != api.StatusError
		_, disagree, lines := crossvalidate.Compare(kanjoCompliant, verdicts)
		totalDisagree += disagree
		fmt.Fprintf(os.Stdout, "  %s (Kanjō : %s)\n", r.Input, verdictLabel(kanjoCompliant))
		for _, l := range lines {
			fmt.Fprintf(os.Stdout, "    · %s\n", l)
		}
	}
	if totalDisagree > 0 {
		fmt.Fprintf(os.Stdout, "  ⚠ %d désaccord(s) avec un validateur externe — à investiguer.\n", totalDisagree)
	}
}

func verdictLabel(compliant bool) string {
	if compliant {
		return "conforme"
	}
	return "non conforme"
}

// writeReport écrit un rapport de validation selon l'extension (.html ou .json).
func writeReport(path string, env *api.Envelope) error {
	var data []byte
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		var err error
		data, err = render.RenderValidationReportHTML(env, version.Rules)
		if err != nil {
			return err
		}
	default: // .json
		var err error
		data, err = json.MarshalIndent(env, "", "  ")
		if err != nil {
			return err
		}
	}
	return fsatomic.WriteFile(path, data, 0o644)
}

func renderValidate(env *api.Envelope, explain bool) {
	seal := map[api.Status]string{api.StatusOK: "✓", api.StatusWarning: "⚠", api.StatusError: "✗"}
	for _, r := range env.Results {
		fmt.Fprintf(os.Stdout, "%s %s", seal[r.Status], r.Input)
		if r.Format != "" {
			fmt.Fprintf(os.Stdout, "  [%s]", r.Format)
		}
		fmt.Fprintln(os.Stdout)
		if r.Error != "" {
			fmt.Fprintf(os.Stdout, "    erreur : %s\n", r.Error)
		}
		for _, f := range r.Findings {
			fmt.Fprintf(os.Stdout, "    %-9s %-10s %s\n", severityMark(f.Severity), f.RuleID, f.Message)
			if explain && (f.Expected != "" || f.Actual != "") {
				fmt.Fprintf(os.Stdout, "              attendu %s · trouvé %s\n", f.Expected, f.Actual)
			}
		}
	}
	s := env.Summary
	fmt.Fprintf(os.Stdout, "\n✓ %d   ⚠ %d   ✗ %d\n", s.OK, s.Warning, s.Error)
}

func severityMark(sev string) string {
	switch sev {
	case "error", "fatal":
		return "✗"
	case "warning":
		return "⚠"
	default:
		return "·"
	}
}
