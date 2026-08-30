// Package edifact sérialise un document pivot en message UN/EDIFACT INVOIC (D.96A), en syntaxe
// texte ISO 9735 avec les séparateurs de service par défaut (UNA:+.? '). C'est le symétrique du
// lecteur pkg/read/edifact : un aller-retour pivot→EDIFACT→pivot préserve les champs porteurs de
// sens (en-tête, parties, lignes, catégories/taux de TVA, totaux).
//
// Limite assumée (§17.7, rule 14) : EDIFACT INVOIC ne transporte pas toute la richesse d'EN 16931
// (p. ex. montants par sous-catégorie de la ventilation). Aucune donnée n'est inventée ; ce qui
// n'a pas d'équivalent segmentaire n'est simplement pas émis.
package edifact

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func init() { write.Register("edifact", Write) }

// Séparateurs de service par défaut (UNA:+.? ').
const (
	compSep = ':'
	dataSep = '+'
	release = '?'
	segTerm = '\''
)

// Write produit un interchange EDIFACT INVOIC complet (UNB…UNZ) pour le document donné.
func Write(doc *model.Document, _ write.Options) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("edifact: document nil")
	}
	w := &builder{}

	ctrlRef := nonEmptyOr(doc.ID, "1")
	// UNB : en-tête d'interchange. Date/heure tirées de la date de facture (le pivot ne porte pas
	// d'horodatage d'échange). Expéditeur/destinataire : identifiants TVA à défaut de mieux.
	w.raw(fmt.Sprintf("UNB+UNOC:3+%s+%s+%s:0000+%s",
		esc(nonEmptyOr(doc.Seller.VATID, doc.Seller.Name, "SENDER")),
		esc(nonEmptyOr(doc.Buyer.VATID, doc.Buyer.Name, "RECIPIENT")),
		dateYYMMDD(doc.IssueDate), esc(ctrlRef)))

	// À partir d'ici, on compte les segments du message (UNH..UNT inclus).
	w.count = 0
	w.seg("UNH", comps("1"), comps("INVOIC", "D", "96A", "UN"))

	typeCode := string(doc.TypeCode)
	if typeCode == "" {
		typeCode = string(model.TypeCommercialInvoice)
	}
	w.seg("BGM", comps(typeCode), comps(nonEmptyOr(doc.ID, "")), comps("9"))
	if !doc.IssueDate.IsZero() {
		w.seg("DTM", comps("137", dateCCYYMMDD(doc.IssueDate), "102"))
	}

	// Parties : SU = fournisseur (vendeur), BY = acheteur, avec leur identifiant TVA en RFF+VA.
	writeParty(w, "SU", doc.Seller)
	writeParty(w, "BY", doc.Buyer)

	if doc.CurrencyCode != "" {
		w.seg("CUX", comps("2", doc.CurrencyCode, "4"))
	}

	// Lignes.
	for i, l := range doc.Lines {
		lineNo := l.ID
		if lineNo == "" {
			lineNo = fmt.Sprintf("%d", i+1)
		}
		w.seg("LIN", comps(lineNo), comps(""), comps(nonEmptyOr(l.Name, lineNo), "IN"))
		if l.Name != "" {
			w.seg("IMD", comps("F"), comps(""), comps("", "", "", l.Name))
		}
		w.seg("QTY", comps("47", l.Quantity.String(), string(l.UnitCode)))
		if !l.NetAmount.IsZero() {
			w.seg("MOA", comps("203", l.NetAmount.String()))
		}
		if !l.NetPrice.IsZero() {
			w.seg("PRI", comps("AAA", l.NetPrice.String()))
		}
		if l.TaxCategory != "" {
			rate := "0"
			if l.TaxRate != nil {
				rate = l.TaxRate.String()
			}
			w.seg("TAX", comps("7"), comps("VAT"), comps(""), comps(""), comps("", "", "", rate), comps(string(l.TaxCategory)))
		}
	}

	// Sommaire.
	w.raw2("UNS", "S")
	writeMOA(w, "79", doc.Totals.LineExtensionAmount)
	writeMOA(w, "125", doc.Totals.TaxExclusiveAmount)
	writeMOA(w, "124", doc.Totals.TaxAmount)
	writeMOA(w, "77", doc.Totals.TaxInclusiveAmount)
	writeMOA(w, "9", doc.Totals.DuePayableAmount)
	// Ventilation de TVA (catégorie + taux ; les montants par sous-catégorie ne sont pas
	// transportables tels quels en INVOIC — cf. limite assumée du format).
	for _, ts := range doc.TaxBreakdown {
		w.seg("TAX", comps("7"), comps("VAT"), comps(""), comps(""), comps("", "", "", ts.Rate.String()), comps(string(ts.Category)))
	}

	// UNT : nombre de segments du message (UNH..UNT inclus) + référence de message.
	w.seg("UNT", comps(fmt.Sprintf("%d", w.count+1)), comps("1"))
	w.raw(fmt.Sprintf("UNZ+1+%s", esc(ctrlRef)))

	return []byte(w.b.String()), nil
}

// --- Construction de segments ---

type builder struct {
	b     strings.Builder
	count int // segments du message courant (UNH..UNT)
}

// comps assemble des composants en un élément de données.
func comps(c ...string) []string { return c }

// seg écrit un segment (tag + éléments) et incrémente le compteur de message.
func (w *builder) seg(tag string, elems ...[]string) {
	parts := make([]string, 0, len(elems)+1)
	parts = append(parts, tag)
	for _, e := range elems {
		cc := make([]string, len(e))
		for i, c := range e {
			cc[i] = esc(c)
		}
		parts = append(parts, strings.Join(cc, string(compSep)))
	}
	w.b.WriteString(strings.Join(parts, string(dataSep)))
	w.b.WriteByte(segTerm)
	w.b.WriteByte('\n')
	w.count++
}

// raw écrit un segment déjà formaté (UNB/UNZ, hors comptage message).
func (w *builder) raw(s string) {
	w.b.WriteString(s)
	w.b.WriteByte(segTerm)
	w.b.WriteByte('\n')
}

// raw2 écrit un segment simple compté dans le message (ex. UNS+S).
func (w *builder) raw2(tag, el string) {
	w.seg(tag, comps(el))
}

func writeParty(w *builder, role string, p model.Party) {
	if p.Name == "" && p.VATID == "" {
		return
	}
	a := p.Address
	w.seg("NAD", comps(role), comps(""), comps(""),
		comps(p.Name), comps(a.Line1), comps(a.City), comps(""), comps(a.PostalCode), comps(a.CountryCode))
	if p.VATID != "" {
		w.seg("RFF", comps("VA", p.VATID))
	}
}

func writeMOA(w *builder, qualifier string, a model.Amount) {
	if a.IsZero() {
		return
	}
	w.seg("MOA", comps(qualifier, a.String()))
}

// esc préfixe le caractère d'échappement devant tout séparateur de service présent dans une valeur.
func esc(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == compSep || c == dataSep || c == release || c == segTerm {
			b.WriteByte(release)
		}
		b.WriteByte(c)
	}
	return b.String()
}

func dateCCYYMMDD(d model.Date) string {
	return fmt.Sprintf("%04d%02d%02d", d.Year, int(d.Month), d.Day)
}

func dateYYMMDD(d model.Date) string {
	if d.IsZero() {
		return "000101"
	}
	return fmt.Sprintf("%02d%02d%02d", d.Year%100, int(d.Month), d.Day)
}

func nonEmptyOr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
