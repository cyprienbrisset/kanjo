package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/api"
	"github.com/cyprienbrisset/kanjo/pkg/audit"
)

func runAudit(args []string) int {
	if len(args) == 0 {
		errf("audit : sous-commande requise (list|export|verify|path)")
		return ExitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "list":
		return auditList(rest)
	case "export":
		return auditExport(rest)
	case "verify":
		return auditVerify(rest)
	case "path":
		p, _ := audit.DefaultPath()
		fmt.Fprintln(os.Stdout, p)
		return ExitOK
	default:
		errf("audit : sous-commande inconnue %q", sub)
		return ExitUsage
	}
}

// auditVerify vérifie la chaîne d'intégrité du journal (tamper-evident).
func auditVerify(args []string) int {
	fs := flag.NewFlagSet("audit verify", flag.ContinueOnError)
	format := fs.String("format", "", "table|json")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	path, _ := audit.DefaultPath()
	entries, err := audit.Read(path)
	if err != nil {
		errf("audit verify : %v", err)
		return ExitInternal
	}
	rep := audit.VerifyChain(entries)
	if outputFormat(*format) == "json" {
		printJSON(rep)
	} else {
		fmt.Fprintf(os.Stdout, "▸ %d entrées · %d chaînées · %d héritées\n", rep.Total, rep.Chained, rep.Unchained)
		if rep.OK {
			fmt.Fprintln(os.Stdout, "✓ intégrité vérifiée : chaîne intacte.")
		} else {
			fmt.Fprintf(os.Stdout, "✗ RUPTURE : %d anomalie(s) détectée(s).\n", len(rep.Issues))
			for _, is := range rep.Issues {
				fmt.Fprintf(os.Stdout, "   · entrée #%d (seq %d) : %s\n", is.Index, is.Seq, is.Problem)
			}
		}
	}
	if !rep.OK {
		return ExitValidation
	}
	return ExitOK
}

func auditList(args []string) int {
	fs := flag.NewFlagSet("audit list", flag.ContinueOnError)
	format := fs.String("format", "", "table|json")
	action := fs.String("action", "", "filtrer par action")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	path, _ := audit.DefaultPath()
	entries, err := audit.Read(path)
	if err != nil {
		errf("audit list : %v", err)
		return ExitInternal
	}
	if *action != "" {
		entries = filterAction(entries, *action)
	}
	if outputFormat(*format) == "json" {
		printJSON(entries)
		return ExitOK
	}
	if len(entries) == 0 {
		fmt.Fprintln(os.Stdout, "▸ aucune entrée d'audit.")
		return ExitOK
	}
	for _, e := range entries {
		fmt.Fprintf(os.Stdout, "%s  %-9s  %-8s → %-8s  %s\n", e.Ts, e.Action, e.InputFormat, e.OutputFormat, e.Verdict)
	}
	return ExitOK
}

func auditExport(args []string) int {
	fs := flag.NewFlagSet("audit export", flag.ContinueOnError)
	out := fs.String("out", "", "fichier de sortie (.csv|.jsonl|.html) (requis)")
	format := fs.String("format", "", "csv|jsonl|html (déduit de l'extension par défaut)")
	from := fs.String("from", "", "borne de début (YYYY-MM-DD ou RFC3339)")
	to := fs.String("to", "", "borne de fin (YYYY-MM-DD ou RFC3339)")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	if *out == "" {
		errf("audit export : --out est requis")
		return ExitUsage
	}
	fromT, err := parseAuditDate(*from, false)
	if err != nil {
		errf("audit export : --from %v", err)
		return ExitUsage
	}
	toT, err := parseAuditDate(*to, true)
	if err != nil {
		errf("audit export : --to %v", err)
		return ExitUsage
	}
	path, _ := audit.DefaultPath()
	entries, err := audit.Read(path)
	if err != nil {
		errf("audit export : %v", err)
		return ExitInternal
	}
	entries = audit.FilterByPeriod(entries, fromT, toT)

	kind := *format
	if kind == "" {
		switch {
		case hasSuffixFold(*out, ".csv"):
			kind = "csv"
		case hasSuffixFold(*out, ".html"), hasSuffixFold(*out, ".htm"):
			kind = "html"
		default:
			kind = "jsonl"
		}
	}
	switch kind {
	case "csv":
		err = fsatomic.WriteFile(*out, audit.ExportCSV(entries), 0o600)
	case "html":
		err = fsatomic.WriteFile(*out, audit.ExportHTML(entries, "Journal d'audit Kanjō"), 0o600)
	default:
		err = audit.WriteJSONL(*out, entries)
	}
	if err != nil {
		errf("audit export : %v", err)
		return ExitInternal
	}
	rep := audit.VerifyChain(entries)
	integ := "intègre"
	if !rep.OK {
		integ = fmt.Sprintf("RUPTURE (%d anomalie(s))", len(rep.Issues))
	}
	fmt.Fprintf(os.Stdout, "%d entrées exportées vers %s (%s) — intégrité : %s\n", len(entries), *out, kind, integ)
	return ExitOK
}

// parseAuditDate accepte une date « YYYY-MM-DD » (bornée à minuit, ou fin de journée si endOfDay)
// ou un horodatage RFC3339 complet. Une chaîne vide renvoie le temps zéro (borne ignorée).
func parseAuditDate(s string, endOfDay bool) (time.Time, error) {
	s = trimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("date invalide %q (attendu YYYY-MM-DD ou RFC3339)", s)
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Nanosecond)
	}
	return t, nil
}

func trimSpace(s string) string { return strings.TrimSpace(s) }

func filterAction(entries []audit.Entry, action string) []audit.Entry {
	var out []audit.Entry
	for _, e := range entries {
		if e.Action == action {
			out = append(out, e)
		}
	}
	return out
}

// recordAudit journalise chaque résultat d'un lot (sans PII, §17.5). Les erreurs d'audit ne
// doivent jamais interrompre un traitement.
func recordAudit(env *api.Envelope, action string) {
	for _, r := range env.Results {
		e := audit.Entry{
			Action: action, Verdict: string(r.Status), Profile: r.Profile,
			InputFormat: r.Format, LossCount: len(r.Losses),
		}
		if r.Hashes != nil {
			e.InputSha256 = r.Hashes.InputSha256
			e.OutputSha256 = r.Hashes.OutputSha256
		}
		if action == "convert" && r.Output != "" {
			e.OutputFormat = detectTargetFromResult(r)
		}
		_ = audit.Record(e)
	}
}

// detectTargetFromResult déduit le format de sortie d'un résultat de conversion (best-effort).
func detectTargetFromResult(r api.Result) string {
	ext := extOf(r.Output)
	switch ext {
	case ".pdf":
		return "facturx"
	case ".json":
		return "json"
	case ".csv":
		return "csv"
	default:
		return "xml"
	}
}

func extOf(p string) string {
	for i := len(p) - 1; i >= 0 && p[i] != '/' && p[i] != '\\'; i-- {
		if p[i] == '.' {
			return p[i:]
		}
	}
	return ""
}

func hasSuffixFold(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return equalFold(s[len(s)-len(suffix):], suffix)
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
