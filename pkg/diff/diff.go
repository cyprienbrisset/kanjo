// Package diff fournit le comparateur sémantique de Kanjō (§8, §11.2, G5).
//
// Le principe (§G5) est de comparer deux documents pivot terme à terme (Business Terms
// EN 16931) plutôt qu'octet à octet. Puisque toute entrée passe d'abord par le pivot,
// comparer une Factur-X à une UBL revient simplement à comparer leurs pivots respectifs.
//
// Deux catégories de différences sont distinguées et comptées séparément :
//   - une PERTE (KindLoss) : un terme présent d'un seul côté ;
//   - une DIVERGENCE (KindDivergence) : un terme présent des deux côtés mais de valeurs
//     différentes.
//
// Les termes identiques (KindEqual) et les termes absents des deux côtés ne comptent pas
// comme des différences.
package diff

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// ChangeKind qualifie la nature d'une différence sur un terme donné.
type ChangeKind string

const (
	KindEqual      ChangeKind = "equal"      // présent des deux côtés, valeurs identiques
	KindLoss       ChangeKind = "loss"       // présent d'un seul côté (perte)
	KindDivergence ChangeKind = "divergence" // présent des deux côtés, valeurs différentes
	KindAdded      ChangeKind = "added"      // présent à droite seulement (ajout)
)

// TermDiff décrit la comparaison d'un Business Term entre les deux documents.
type TermDiff struct {
	Term  string     `json:"term"`            // identifiant EN 16931 (ex. "BT-112")
	Label string     `json:"label,omitempty"` // libellé lisible (ex. "Total TTC")
	Left  string     `json:"left"`            // valeur textuelle à gauche ("" = absent)
	Right string     `json:"right"`           // valeur textuelle à droite ("" = absent)
	Kind  ChangeKind `json:"kind"`            // nature de la différence
}

// Report agrège l'ensemble des comparaisons et les compteurs par catégorie.
type Report struct {
	Terms       []TermDiff `json:"terms"`
	Losses      int        `json:"losses"`
	Divergences int        `json:"divergences"`
	Equal       int        `json:"equal"`
}

// Options paramètre la comparaison.
type Options struct {
	HideEqual        bool            // n'inclut pas les termes identiques dans le rapport
	IgnoreFormatting bool            // normalise nombres/dates avant comparaison
	Ignore           map[string]bool // termes exclus du comptage et de l'affichage
}

// term est une valeur préparée pour comparaison : identifiant, libellé et les deux
// représentations textuelles. Pour les montants, une comparaison sémantique via
// Amount.Equal est disponible.
type term struct {
	id    string
	label string
	left  string
	right string
	// leftAmt/rightAmt portent la comparaison sémantique des montants (IgnoreFormatting).
	amounts  bool
	leftAmt  model.Amount
	rightAmt model.Amount
	leftHas  bool
	rightHas bool
}

// Compare compare deux documents pivot terme à terme et renvoie un rapport.
// left ou right peuvent être nil (traité comme un document vide).
func Compare(left, right *model.Document, opts Options) Report {
	if left == nil {
		left = &model.Document{}
	}
	if right == nil {
		right = &model.Document{}
	}

	var terms []term
	terms = append(terms, headerTerms(left, right)...)
	terms = append(terms, totalTerms(left, right)...)
	terms = append(terms, lineCountTerm(left, right))
	terms = append(terms, lineTerms(left, right)...)
	terms = append(terms, taxTerms(left, right)...)

	var rep Report
	for _, t := range terms {
		if opts.Ignore != nil && opts.Ignore[t.id] {
			continue
		}
		kind, include := classify(t, opts)
		if kind == "" { // deux côtés vides : ignoré
			continue
		}
		switch kind {
		case KindEqual:
			rep.Equal++
		case KindLoss, KindAdded:
			rep.Losses++
		case KindDivergence:
			rep.Divergences++
		}
		if include {
			rep.Terms = append(rep.Terms, TermDiff{
				Term:  t.id,
				Label: t.label,
				Left:  t.left,
				Right: t.right,
				Kind:  kind,
			})
		}
	}
	return rep
}

// classify détermine la nature de la différence d'un terme et si elle doit figurer
// dans le rapport. Un kind vide signifie « les deux côtés absents » (à ignorer).
func classify(t term, opts Options) (ChangeKind, bool) {
	leftEmpty := t.left == ""
	rightEmpty := t.right == ""
	switch {
	case leftEmpty && rightEmpty:
		return "", false
	case leftEmpty && !rightEmpty:
		return KindAdded, true
	case !leftEmpty && rightEmpty:
		return KindLoss, true
	}
	// Les deux présents : égaux ou divergents.
	if t.equalValues(opts) {
		return KindEqual, !opts.HideEqual
	}
	return KindDivergence, true
}

// equalValues compare les deux valeurs présentes, en tenant compte d'IgnoreFormatting.
func (t term) equalValues(opts Options) bool {
	if opts.IgnoreFormatting && t.amounts && t.leftHas && t.rightHas {
		return t.leftAmt.Equal(t.rightAmt)
	}
	return t.left == t.right
}

// --- Constructeurs de termes ---------------------------------------------------------

// txt construit un terme purement textuel.
func txt(id, label, left, right string) term {
	return term{id: id, label: label, left: left, right: right}
}

// amt construit un terme monétaire : la représentation textuelle sert à l'affichage,
// les Amount servent à la comparaison sémantique quand IgnoreFormatting est actif.
func amt(id, label string, left, right model.Amount, leftHas, rightHas bool) term {
	t := term{id: id, label: label, amounts: true, leftHas: leftHas, rightHas: rightHas}
	if leftHas {
		t.left = left.String()
		t.leftAmt = left
	}
	if rightHas {
		t.right = right.String()
		t.rightAmt = right
	}
	return t
}

func dateStr(d model.Date) string {
	if d.IsZero() {
		return ""
	}
	return d.ISO()
}

func dueStr(d *model.Date) string {
	if d == nil || d.IsZero() {
		return ""
	}
	return d.ISO()
}

func headerTerms(l, r *model.Document) []term {
	return []term{
		txt("BT-1", "Numéro de facture", l.ID, r.ID),
		txt("BT-2", "Date d'émission", dateStr(l.IssueDate), dateStr(r.IssueDate)),
		txt("BT-3", "Code type", string(l.TypeCode), string(r.TypeCode)),
		txt("BT-5", "Devise", l.CurrencyCode, r.CurrencyCode),
		txt("BT-9", "Date d'échéance", dueStr(l.DueDate), dueStr(r.DueDate)),
		txt("BT-10", "Référence acheteur", l.BuyerReference, r.BuyerReference),
		txt("BT-13", "Réf. bon de commande", l.PurchaseOrderRef, r.PurchaseOrderRef),
		txt("BT-27", "Nom du vendeur", l.Seller.Name, r.Seller.Name),
		txt("BT-31", "N° TVA vendeur", l.Seller.VATID, r.Seller.VATID),
		txt("BT-44", "Nom de l'acheteur", l.Buyer.Name, r.Buyer.Name),
	}
}

func totalTerms(l, r *model.Document) []term {
	return []term{
		amt("BT-106", "Total des lignes", l.Totals.LineExtensionAmount, r.Totals.LineExtensionAmount, true, true),
		amt("BT-109", "Total HT", l.Totals.TaxExclusiveAmount, r.Totals.TaxExclusiveAmount, true, true),
		amt("BT-110", "Total TVA", l.Totals.TaxAmount, r.Totals.TaxAmount, true, true),
		amt("BT-112", "Total TTC", l.Totals.TaxInclusiveAmount, r.Totals.TaxInclusiveAmount, true, true),
		amt("BT-115", "Net à payer", l.Totals.DuePayableAmount, r.Totals.DuePayableAmount, true, true),
	}
}

func lineCountTerm(l, r *model.Document) term {
	return txt("BG-25#", "Nombre de lignes", fmt.Sprintf("%d", len(l.Lines)), fmt.Sprintf("%d", len(r.Lines)))
}

// lineTerms compare les lignes appariées par identifiant (BT-126). Chaque ligne présente
// d'un seul côté produit des pertes/ajouts ; chaque ligne présente des deux côtés produit
// des comparaisons champ à champ (désignation, quantité, prix net, montant net).
func lineTerms(l, r *model.Document) []term {
	var out []term
	ids := orderedLineIDs(l, r)
	for _, id := range ids {
		ll := l.LineByID(id)
		rr := r.LineByID(id)
		suffix := "[" + id + "]"
		out = append(out, term{
			id:    "BT-126" + suffix,
			label: "Ligne " + id + " · identifiant",
			left:  lineID(ll),
			right: lineID(rr),
		})
		out = append(out, txt("BT-153"+suffix, "Ligne "+id+" · désignation", lineName(ll), lineName(rr)))
		out = append(out, txt("BT-129"+suffix, "Ligne "+id+" · quantité", lineQty(ll), lineQty(rr)))
		out = append(out, lineAmt("BT-146"+suffix, "Ligne "+id+" · prix net", ll, rr, netPriceOf))
		out = append(out, lineAmt("BT-131"+suffix, "Ligne "+id+" · montant net", ll, rr, netAmountOf))
	}
	return out
}

// orderedLineIDs renvoie l'union ordonnée des identifiants de ligne (gauche d'abord,
// puis les identifiants présents uniquement à droite), pour un rapport déterministe.
func orderedLineIDs(l, r *model.Document) []string {
	seen := map[string]bool{}
	var ids []string
	for i := range l.Lines {
		id := l.Lines[i].ID
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for i := range r.Lines {
		id := r.Lines[i].ID
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

func lineID(l *model.Line) string {
	if l == nil {
		return ""
	}
	return l.ID
}

func lineName(l *model.Line) string {
	if l == nil {
		return ""
	}
	return l.Name
}

func lineQty(l *model.Line) string {
	if l == nil {
		return ""
	}
	return l.Quantity.String()
}

func netPriceOf(l *model.Line) (model.Amount, bool) {
	if l == nil {
		return model.Amount{}, false
	}
	return l.NetPrice, true
}

func netAmountOf(l *model.Line) (model.Amount, bool) {
	if l == nil {
		return model.Amount{}, false
	}
	return l.NetAmount, true
}

// lineAmt construit un terme monétaire de ligne à partir d'un accès optionnel.
func lineAmt(id, label string, l, r *model.Line, get func(*model.Line) (model.Amount, bool)) term {
	la, lh := get(l)
	ra, rh := get(r)
	return amt(id, label, la, ra, lh, rh)
}

// taxTerms compare chaque ventilation de TVA (BG-23), appariée par couple catégorie/taux.
// Pour chaque ventilation : base (BT-116), montant de TVA (BT-117), taux (BT-119).
func taxTerms(l, r *model.Document) []term {
	var out []term
	keys := orderedTaxKeys(l, r)
	for _, k := range keys {
		ls := taxByKey(l, k)
		rs := taxByKey(r, k)
		suffix := "[" + k + "]"
		out = append(out, term{
			id:    "BT-119" + suffix,
			label: "TVA " + k + " · taux",
			left:  taxRate(ls),
			right: taxRate(rs),
		})
		out = append(out, taxAmt("BT-116"+suffix, "TVA "+k+" · base", ls, rs, func(t *model.TaxSubtotal) model.Amount { return t.TaxableAmount }))
		out = append(out, taxAmt("BT-117"+suffix, "TVA "+k+" · montant", ls, rs, func(t *model.TaxSubtotal) model.Amount { return t.TaxAmount }))
	}
	return out
}

// taxKey identifie une ventilation par catégorie et taux.
func taxKey(t model.TaxSubtotal) string {
	return string(t.Category) + "@" + t.Rate.String()
}

func orderedTaxKeys(l, r *model.Document) []string {
	seen := map[string]bool{}
	var keys []string
	for i := range l.TaxBreakdown {
		k := taxKey(l.TaxBreakdown[i])
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for i := range r.TaxBreakdown {
		k := taxKey(r.TaxBreakdown[i])
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	return keys
}

func taxByKey(d *model.Document, key string) *model.TaxSubtotal {
	for i := range d.TaxBreakdown {
		if taxKey(d.TaxBreakdown[i]) == key {
			return &d.TaxBreakdown[i]
		}
	}
	return nil
}

func taxRate(t *model.TaxSubtotal) string {
	if t == nil {
		return ""
	}
	return t.Rate.String()
}

func taxAmt(id, label string, l, r *model.TaxSubtotal, get func(*model.TaxSubtotal) model.Amount) term {
	var la, ra model.Amount
	lh := l != nil
	rh := r != nil
	if lh {
		la = get(l)
	}
	if rh {
		ra = get(r)
	}
	return amt(id, label, la, ra, lh, rh)
}
