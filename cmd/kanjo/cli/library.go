package cli

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cyprienbrisset/kanjo/pkg/library"
	"github.com/cyprienbrisset/kanjo/pkg/pipeline"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/rules"

	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

func openLibrary() (*library.Library, int) {
	path, err := library.DefaultPath()
	if err != nil {
		errf("library : %v", err)
		return nil, ExitInternal
	}
	l, err := library.Open(path)
	if err != nil {
		errf("library : %v", err)
		return nil, ExitInternal
	}
	return l, ExitOK
}

func runLibrary(args []string) int {
	if len(args) == 0 {
		errf("library : sous-commande requise (index|list|forget|purge)")
		return ExitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "index":
		return libraryIndex(rest)
	case "list", "search":
		return libraryList(rest)
	case "forget":
		return libraryForget(rest)
	case "purge":
		return libraryPurge(rest)
	default:
		errf("library : sous-commande inconnue %q", sub)
		return ExitUsage
	}
}

func libraryIndex(args []string) int {
	fs := flag.NewFlagSet("library index", flag.ContinueOnError)
	recursive := fs.Bool("recursive", false, "parcours récursif")
	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(inputs) == 0 {
		errf("library index : aucun fichier")
		return ExitUsage
	}
	files, err := pipeline.Discover(inputs, *recursive, nil, nil)
	if err != nil {
		errf("library index : %v", err)
		return ExitUsage
	}
	l, code := openLibrary()
	if l == nil {
		return code
	}
	defer l.Close()

	indexed := 0
	for _, in := range files {
		data, err := readInput(in)
		if err != nil {
			errf("%v", err)
			continue
		}
		rd, err := read.ReadBytes(data, in)
		if err != nil {
			errf("%s : %v", in, err)
			continue
		}
		doc := rd.Doc
		verdict := "ok"
		if rep := rules.Validate(doc); rep.HasErrors() {
			verdict = "error"
		} else if len(rep.Findings) > 0 {
			verdict = "warning"
		}
		rec := library.Record{
			ID: doc.ID, IssueDate: doc.IssueDate.ISO(),
			SellerName: doc.Seller.Name, BuyerName: doc.Buyer.Name,
			TotalTTC: doc.Totals.TaxInclusiveAmount.String(), Currency: doc.CurrencyCode,
			Format: string(rd.Format), Profile: rd.Profile, Verdict: verdict,
			InputSha256: sha256hex(data), InputPath: in,
		}
		if err := l.Index(rec); err != nil {
			errf("%s : %v", in, err)
			continue
		}
		indexed++
	}
	fmt.Fprintf(os.Stdout, "▸ %d documents indexés\n", indexed)
	return ExitOK
}

func libraryList(args []string) int {
	fs := flag.NewFlagSet("library list", flag.ContinueOnError)
	text := fs.String("text", "", "recherche (numéro, vendeur, acheteur)")
	verdict := fs.String("verdict", "", "filtrer par verdict")
	format := fs.String("format", "", "table|json (sortie) — ambigu avec le format de facture, voir --doc-format")
	docFormat := fs.String("doc-format", "", "filtrer par format de facture")
	from := fs.String("from", "", "date d'émission min (AAAA-MM-JJ)")
	to := fs.String("to", "", "date d'émission max")
	limit := fs.Int("limit", 100, "nombre maximum de résultats")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	l, code := openLibrary()
	if l == nil {
		return code
	}
	defer l.Close()

	recs, err := l.Search(library.Query{
		Text: *text, Verdict: *verdict, Format: *docFormat, From: *from, To: *to, Limit: *limit,
	})
	if err != nil {
		errf("library list : %v", err)
		return ExitInternal
	}
	if outputFormat(*format) == "json" {
		printJSON(recs)
		return ExitOK
	}
	if len(recs) == 0 {
		fmt.Fprintln(os.Stdout, "▸ bibliothèque vide (ou aucun résultat).")
		return ExitOK
	}
	for _, r := range recs {
		fmt.Fprintf(os.Stdout, "%s %-14s %-24s %12s %s [%s]\n",
			sealForVerdict(r.Verdict), r.ID, truncate(r.SellerName, 24), r.TotalTTC+" "+r.Currency, r.IssueDate, r.Format)
	}
	return ExitOK
}

func libraryForget(args []string) int {
	if len(args) != 1 {
		errf("library forget : un critère (texte) est requis")
		return ExitUsage
	}
	l, code := openLibrary()
	if l == nil {
		return code
	}
	defer l.Close()
	n, err := l.Forget(args[0])
	if err != nil {
		errf("library forget : %v", err)
		return ExitInternal
	}
	fmt.Fprintf(os.Stdout, "%d entrées effacées (droit à l'effacement RGPD)\n", n)
	return ExitOK
}

func libraryPurge(args []string) int {
	fs := flag.NewFlagSet("library purge", flag.ContinueOnError)
	months := fs.Int("months", 13, "purger les documents plus vieux que N mois")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	l, code := openLibrary()
	if l == nil {
		return code
	}
	defer l.Close()
	cutoff := time.Now().AddDate(0, -*months, 0)
	n, err := l.PurgeBefore(cutoff)
	if err != nil {
		errf("library purge : %v", err)
		return ExitInternal
	}
	fmt.Fprintf(os.Stdout, "%d documents purgés (antérieurs à %s)\n", n, cutoff.Format("2006-01-02"))
	return ExitOK
}

func sealForVerdict(v string) string {
	switch v {
	case "error":
		return "✗"
	case "warning":
		return "⚠"
	case "ok":
		return "✓"
	default:
		return "·"
	}
}
