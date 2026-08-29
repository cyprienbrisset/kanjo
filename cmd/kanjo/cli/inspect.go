package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

func runInspect(args []string) int {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	show := fs.String("show", "summary", "summary|terms|json")
	term := fs.String("term", "", "afficher un BT précis (ex : BT-112)")
	format := fs.String("format", "", "format de sortie : table|json")
	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(inputs) != 1 {
		errf("inspect : exactement un fichier d'entrée est requis")
		return ExitUsage
	}
	data, err := readInput(inputs[0])
	if err != nil {
		errf("%v", err)
		return ExitUnreadable
	}
	rd, err := read.ReadBytes(data, inputs[0])
	if err != nil {
		errf("%v", err)
		return ExitUnreadable
	}
	doc := rd.Doc

	if outputFormat(*format) == "json" || *show == "json" {
		printJSON(doc)
		return ExitOK
	}
	if *term != "" {
		printTerm(doc, *term)
		return ExitOK
	}
	switch *show {
	case "terms":
		printTerms(doc)
	default:
		printSummary(doc, rd.Format, rd.Profile)
	}
	return ExitOK
}

func printSummary(doc *model.Document, format read.Format, profile string) {
	w := os.Stdout
	fmt.Fprintf(w, "▸ %s  (%s / %s)\n", doc.ID, format, profile)
	fmt.Fprintf(w, "  Type          %s (%s)\n", doc.TypeCode, doc.TypeCode.Label(model.LangFR))
	fmt.Fprintf(w, "  Émission      %s\n", doc.IssueDate.ISO())
	if doc.DueDate != nil {
		fmt.Fprintf(w, "  Échéance      %s\n", doc.DueDate.ISO())
	}
	fmt.Fprintf(w, "  Devise        %s\n", doc.CurrencyCode)
	fmt.Fprintf(w, "  Émetteur      %s", doc.Seller.Name)
	if doc.Seller.VATID != "" {
		fmt.Fprintf(w, "  (TVA %s)", doc.Seller.VATID)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Destinataire  %s\n", doc.Buyer.Name)
	fmt.Fprintf(w, "  Lignes        %d\n", len(doc.Lines))
	fmt.Fprintf(w, "  Total HT      %s %s\n", doc.Totals.TaxExclusiveAmount.String(), doc.CurrencyCode)
	fmt.Fprintf(w, "  Total TVA     %s %s\n", doc.Totals.TaxAmount.String(), doc.CurrencyCode)
	fmt.Fprintf(w, "  Total TTC     %s %s\n", doc.Totals.TaxInclusiveAmount.String(), doc.CurrencyCode)
	fmt.Fprintf(w, "  Net à payer   %s %s\n", doc.Totals.DuePayableAmount.String(), doc.CurrencyCode)
}

func printTerms(doc *model.Document) {
	for _, t := range collectTerms(doc) {
		fmt.Fprintf(os.Stdout, "  %-8s %-28s %s\n", t.id, t.label, t.value)
	}
}

func printTerm(doc *model.Document, term string) {
	term = strings.ToUpper(strings.TrimSpace(term))
	for _, t := range collectTerms(doc) {
		if strings.EqualFold(t.id, term) {
			fmt.Fprintln(os.Stdout, t.value)
			return
		}
	}
	errf("terme %s absent du document", term)
}

type termRow struct{ id, label, value string }

func collectTerms(doc *model.Document) []termRow {
	rows := []termRow{
		{"BT-1", "Numéro de facture", doc.ID},
		{"BT-2", "Date d'émission", doc.IssueDate.ISO()},
		{"BT-3", "Code type", string(doc.TypeCode)},
		{"BT-5", "Devise", doc.CurrencyCode},
		{"BT-10", "Référence acheteur", doc.BuyerReference},
		{"BT-13", "Réf. bon de commande", doc.PurchaseOrderRef},
		{"BT-27", "Nom du vendeur", doc.Seller.Name},
		{"BT-31", "N° TVA vendeur", doc.Seller.VATID},
		{"BT-44", "Nom de l'acheteur", doc.Buyer.Name},
		{"BT-106", "Total des lignes", doc.Totals.LineExtensionAmount.String()},
		{"BT-109", "Total HT", doc.Totals.TaxExclusiveAmount.String()},
		{"BT-110", "Total TVA", doc.Totals.TaxAmount.String()},
		{"BT-112", "Total TTC", doc.Totals.TaxInclusiveAmount.String()},
		{"BT-115", "Net à payer", doc.Totals.DuePayableAmount.String()},
	}
	if doc.DueDate != nil {
		rows = append(rows, termRow{"BT-9", "Date d'échéance", doc.DueDate.ISO()})
	}
	out := rows[:0]
	for _, r := range rows {
		if r.value != "" {
			out = append(out, r)
		}
	}
	return out
}
