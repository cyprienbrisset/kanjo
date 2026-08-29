// Package repair corrige les anomalies sûres d'une facture (§8.5). Contrainte MUST : repair
// n'invente JAMAIS de donnée métier (ni SIREN, ni taux de TVA, ni échéance). Il se limite aux
// corrections sûres et journalise chaque changement (avant/après).
package repair

import (
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// Change décrit une correction appliquée (journalisée pour l'audit, §8.5).
type Change struct {
	Path   string `json:"path"`
	Before string `json:"before"`
	After  string `json:"after"`
	Fix    string `json:"fix"`
}

// Fix identifie un type de correction.
type Fix string

const (
	FixTrimIdentifiers Fix = "trim-identifiers" // espaces parasites dans SIREN/TVA/IBAN
	FixRecomputeTotals Fix = "recompute-totals" // recalcul des totaux si les lignes sont cohérentes
)

// AllFixes est l'ensemble des corrections sûres appliquées par défaut.
var AllFixes = []Fix{FixTrimIdentifiers, FixRecomputeTotals}

// Options paramètre la réparation.
type Options struct {
	Fixes []Fix // corrections à appliquer (vide = toutes les sûres)
}

func (o Options) enabled(f Fix) bool {
	if len(o.Fixes) == 0 {
		return true
	}
	for _, x := range o.Fixes {
		if x == f {
			return true
		}
	}
	return false
}

// Repair applique les corrections sûres au document et renvoie la liste des changements.
func Repair(doc *model.Document, opts Options) []Change {
	var changes []Change
	if opts.enabled(FixTrimIdentifiers) {
		changes = append(changes, trimIdentifiers(doc)...)
	}
	if opts.enabled(FixRecomputeTotals) {
		changes = append(changes, recomputeTotals(doc)...)
	}
	return changes
}

func trimIdentifiers(doc *model.Document) []Change {
	var ch []Change
	trim := func(path string, p *string) {
		clean := strings.ReplaceAll(strings.TrimSpace(*p), " ", "")
		if clean != *p {
			ch = append(ch, Change{Path: path, Before: *p, After: clean, Fix: string(FixTrimIdentifiers)})
			*p = clean
		}
	}
	trim("seller.vatId", &doc.Seller.VATID)
	trim("seller.legalId", &doc.Seller.LegalID)
	trim("buyer.vatId", &doc.Buyer.VATID)
	trim("buyer.legalId", &doc.Buyer.LegalID)
	if doc.PaymentInstructions != nil {
		for i := range doc.PaymentInstructions.CreditTransfers {
			trim("paymentInstructions.creditTransfers[].iban", &doc.PaymentInstructions.CreditTransfers[i].IBAN)
		}
	}
	return ch
}

// recomputeTotals recalcule la ventilation de TVA et les totaux d'en-tête à partir des lignes
// (source de vérité, §8.5). Ne recalcule pas les montants de ligne eux-mêmes : repair ne
// fabrique pas de donnée métier de ligne.
func recomputeTotals(doc *model.Document) []Change {
	if len(doc.Lines) == 0 {
		return nil
	}
	var ch []Change
	record := func(path string, before, after model.Amount) {
		if !before.Equal(after) {
			ch = append(ch, Change{Path: path, Before: before.String(), After: after.String(), Fix: string(FixRecomputeTotals)})
		}
	}

	// Ventilation groupée par (catégorie, taux).
	type key struct {
		cat  model.TaxCategoryCode
		rate string
	}
	var order []key
	bases := map[key]model.Amount{}
	rates := map[key]model.Decimal{}
	for _, l := range doc.Lines {
		r := model.MustParseDecimal("0")
		if l.TaxRate != nil {
			r = *l.TaxRate
		}
		k := key{l.TaxCategory, r.String()}
		if _, ok := bases[k]; !ok {
			order = append(order, k)
			bases[k] = model.ZeroAmount(doc.CurrencyCode)
			rates[k] = r
		}
		bases[k], _ = bases[k].Add(l.NetAmount)
	}

	lineTotal := model.ZeroAmount(doc.CurrencyCode)
	taxTotal := model.ZeroAmount(doc.CurrencyCode)
	var breakdown []model.TaxSubtotal
	for _, k := range order {
		ts := model.TaxSubtotal{Category: k.cat, Rate: rates[k], TaxableAmount: bases[k].Rescale(2)}
		ts.TaxAmount = ts.ComputeTaxAmount()
		breakdown = append(breakdown, ts)
		lineTotal, _ = lineTotal.Add(ts.TaxableAmount)
		taxTotal, _ = taxTotal.Add(ts.TaxAmount)
	}
	lineTotal = lineTotal.Rescale(2)
	taxTotal = taxTotal.Rescale(2)
	ttc, _ := lineTotal.Add(taxTotal)
	ttc = ttc.Rescale(2)

	record("totals.lineExtensionAmount", doc.Totals.LineExtensionAmount, lineTotal)
	record("totals.taxExclusiveAmount", doc.Totals.TaxExclusiveAmount, lineTotal)
	record("totals.taxAmount", doc.Totals.TaxAmount, taxTotal)
	record("totals.taxInclusiveAmount", doc.Totals.TaxInclusiveAmount, ttc)

	// Net à payer = TTC − acompte + arrondi (conserve un éventuel acompte existant).
	due := ttc
	if doc.Totals.PrepaidAmount != nil {
		due, _ = ttc.Sub(*doc.Totals.PrepaidAmount)
	}
	if doc.Totals.RoundingAmount != nil {
		due, _ = due.Add(*doc.Totals.RoundingAmount)
	}
	due = due.Rescale(2)
	record("totals.duePayableAmount", doc.Totals.DuePayableAmount, due)

	if len(ch) > 0 {
		doc.TaxBreakdown = breakdown
		doc.Totals.LineExtensionAmount = lineTotal
		doc.Totals.TaxExclusiveAmount = lineTotal
		doc.Totals.TaxAmount = taxTotal
		doc.Totals.TaxInclusiveAmount = ttc
		doc.Totals.DuePayableAmount = due
	}
	return ch
}
