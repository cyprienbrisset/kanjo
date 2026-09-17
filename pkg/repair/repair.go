// Package repair corrige les anomalies sûres d'une facture (§8.5). Contrainte MUST : repair
// n'invente JAMAIS de donnée métier (ni SIREN, ni taux de TVA, ni échéance). Il se limite aux
// corrections sûres et journalise chaque changement (avant/après).
package repair

import (
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	fr "github.com/cyprienbrisset/kanjo/pkg/rules/cius/fr"
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
	FixTrimIdentifiers  Fix = "trim-identifiers"   // espaces parasites dans SIREN/TVA/IBAN
	FixRecomputeTotals  Fix = "recompute-totals"   // recalcul des totaux si les lignes sont cohérentes
	FixFRMandatoryNotes Fix = "fr-mandatory-notes" // mentions BR-FR-05 manquantes (PMT/PMD/AAB)
)

// AllFixes est l'ensemble des corrections sûres appliquées par défaut.
var AllFixes = []Fix{FixTrimIdentifiers, FixRecomputeTotals, FixFRMandatoryNotes}

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
	if opts.enabled(FixFRMandatoryNotes) {
		changes = append(changes, addFRMandatoryNotes(doc)...)
	}
	return changes
}

// addFRMandatoryNotes ajoute, pour un vendeur français, les mentions obligatoires (BR-FR-05/BT-22)
// absentes des notes d'en-tête (BG-1) : frais de recouvrement (PMT), pénalités de retard (PMD) et
// escompte ou son absence (AAB). N'ajoute que les mentions manquantes, dans leur libellé légal
// standard ; ne touche jamais aux notes déjà présentes.
func addFRMandatoryNotes(doc *model.Document) []Change {
	if !strings.EqualFold(doc.Seller.Address.CountryCode, "FR") {
		return nil
	}
	present := map[string]bool{}
	for _, n := range doc.Notes {
		present[n.SubjectCode] = true
	}
	var ch []Change
	for _, n := range fr.MandatoryPaymentNotes() {
		if present[n.SubjectCode] {
			continue
		}
		doc.Notes = append(doc.Notes, n)
		ch = append(ch, Change{
			Path: "notes[]", Before: "", After: n.Content, Fix: string(FixFRMandatoryNotes),
		})
	}
	return ch
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
// et des remises/charges de niveau document (BG-20/21 — source de vérité au même titre que les
// lignes, §8.5). Ne recalcule pas les montants de ligne eux-mêmes, ni le détail d'une remise/
// charge : repair ne fabrique pas de donnée métier.
//
// BR-CO-13 : HT (BT-109) = somme des lignes (BT-106) − remises document (BT-107) + charges
// document (BT-108). Une charge/remise document taxée (BG-20/21 avec catégorie/taux TVA) doit
// aussi entrer dans la base taxable de son taux, sans quoi le total TVA (BT-110) et le HT
// divergent du TTC déclaré dès qu'une charge document existe (ex. éco-contribution REP) — c'est
// ce qui faisait *perdre* le montant de l'éco-contribution lors d'un repair (facture 834514).
func zeroOr(a *model.Amount) string {
	if a == nil {
		return "0"
	}
	return a.String()
}

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
	ensure := func(cat model.TaxCategoryCode, r model.Decimal) key {
		k := key{cat, r.String()}
		if _, ok := bases[k]; !ok {
			order = append(order, k)
			bases[k] = model.ZeroAmount(doc.CurrencyCode)
			rates[k] = r
		}
		return k
	}

	lineTotal := model.ZeroAmount(doc.CurrencyCode)
	for _, l := range doc.Lines {
		r := model.MustParseDecimal("0")
		if l.TaxRate != nil {
			r = *l.TaxRate
		}
		k := ensure(l.TaxCategory, r)
		bases[k], _ = bases[k].Add(l.NetAmount)
		lineTotal, _ = lineTotal.Add(l.NetAmount)
	}
	lineTotal = lineTotal.Rescale(2)

	allowanceTotal := model.ZeroAmount(doc.CurrencyCode)
	chargeTotal := model.ZeroAmount(doc.CurrencyCode)
	for i := range doc.AllowanceCharges {
		ac := &doc.AllowanceCharges[i]
		if ac.IsCharge {
			chargeTotal, _ = chargeTotal.Add(ac.Amount)
		} else {
			allowanceTotal, _ = allowanceTotal.Add(ac.Amount)
		}
		if ac.TaxCategory != "" {
			r := model.MustParseDecimal("0")
			if ac.TaxRate != nil {
				r = *ac.TaxRate
			}
			k := ensure(ac.TaxCategory, r)
			bases[k], _ = bases[k].Add(ac.Signed())
		}
	}
	allowanceTotal = allowanceTotal.Rescale(2)
	chargeTotal = chargeTotal.Rescale(2)

	taxTotal := model.ZeroAmount(doc.CurrencyCode)
	var breakdown []model.TaxSubtotal
	for _, k := range order {
		ts := model.TaxSubtotal{Category: k.cat, Rate: rates[k], TaxableAmount: bases[k].Rescale(2)}
		ts.TaxAmount = ts.ComputeTaxAmount()
		breakdown = append(breakdown, ts)
		taxTotal, _ = taxTotal.Add(ts.TaxAmount)
	}
	taxTotal = taxTotal.Rescale(2)

	recordPtr := func(path string, before *model.Amount, after model.Amount) {
		if before == nil || !before.Equal(after) {
			ch = append(ch, Change{Path: path, Before: zeroOr(before), After: after.String(), Fix: string(FixRecomputeTotals)})
		}
	}

	newTotals := doc.Totals
	newTotals.LineExtensionAmount = lineTotal
	if !allowanceTotal.IsZero() {
		recordPtr("totals.allowanceTotal", doc.Totals.AllowanceTotal, allowanceTotal)
		newTotals.AllowanceTotal = &allowanceTotal
	} else {
		newTotals.AllowanceTotal = nil
	}
	if !chargeTotal.IsZero() {
		recordPtr("totals.chargeTotal", doc.Totals.ChargeTotal, chargeTotal)
		newTotals.ChargeTotal = &chargeTotal
	} else {
		newTotals.ChargeTotal = nil
	}
	taxExclusive, _ := newTotals.ComputeTaxExclusive(doc.CurrencyCode)
	newTotals.TaxExclusiveAmount = taxExclusive
	newTotals.TaxAmount = taxTotal
	ttc, _ := newTotals.ComputeTaxInclusive()
	newTotals.TaxInclusiveAmount = ttc

	record("totals.lineExtensionAmount", doc.Totals.LineExtensionAmount, lineTotal)
	record("totals.taxExclusiveAmount", doc.Totals.TaxExclusiveAmount, taxExclusive)
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
		newTotals.DuePayableAmount = due
		doc.Totals = newTotals
	}
	return ch
}
