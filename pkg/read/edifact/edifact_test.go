package edifact

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// corpus renvoie le chemin d'un échantillon EDIFACT réel (pydifact, MIT).
func corpus(name string) string {
	return filepath.Join("..", "..", "..", "testdata", "corpus", "edifact", name)
}

func readSample(t *testing.T, name string) *model.Document {
	t.Helper()
	data, err := os.ReadFile(corpus(name))
	if err != nil {
		t.Fatalf("lecture fichier %s: %v", name, err)
	}
	doc, err := Read(data, name)
	if err != nil {
		t.Fatalf("Read(%s): %v", name, err)
	}
	return doc
}

// TestInvoice1 vérifie la cartographie d'un INVOIC D.97A réel (avoir).
func TestInvoice1(t *testing.T) {
	doc := readSample(t, "invoice1.edi")

	if doc.Kind != model.KindCreditNote {
		t.Errorf("Kind = %q, attendu creditNote", doc.Kind)
	}
	if doc.TypeCode != model.TypeCreditNote {
		t.Errorf("TypeCode = %q, attendu 381", doc.TypeCode)
	}
	if doc.ID != "1060113800026" {
		t.Errorf("ID = %q", doc.ID)
	}
	if doc.CurrencyCode != "EUR" {
		t.Errorf("devise = %q", doc.CurrencyCode)
	}
	if doc.IssueDate.Year != 1999 || doc.IssueDate.Month != 10 || doc.IssueDate.Day != 6 {
		t.Errorf("date = %+v, attendu 1999-10-06", doc.IssueDate)
	}
	// SU (fournisseur) → vendeur ; BT (à facturer) → acheteur, avec leurs identifiants TVA.
	if doc.Seller.VATID != "123844750" {
		t.Errorf("TVA vendeur = %q", doc.Seller.VATID)
	}
	if doc.Buyer.Name != "VAUXHALL MOTORS LTD" || doc.Buyer.VATID != "382324067" {
		t.Errorf("acheteur = %+v", doc.Buyer)
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("nombre de lignes = %d, attendu 1", len(doc.Lines))
	}
	l := doc.Lines[0]
	if l.Quantity.String() != "54" {
		t.Errorf("quantité = %q", l.Quantity.String())
	}
	if l.NetAmount.String() != "1960.29" {
		t.Errorf("montant net ligne = %q", l.NetAmount.String())
	}
	// MOA+77 → total TTC.
	if doc.Totals.TaxInclusiveAmount.String() != "1960.29" {
		t.Errorf("TTC = %q", doc.Totals.TaxInclusiveAmount.String())
	}
	// TAX+7+VAT sans catégorie explicite, taux 0 → taux zéro déduit.
	if len(doc.TaxBreakdown) == 0 || doc.TaxBreakdown[0].Category != model.TaxZeroRated {
		t.Errorf("ventilation TVA = %+v", doc.TaxBreakdown)
	}
	// Provenance : identifiant de spécification issu de l'en-tête UNH.
	if doc.Provenance == nil || doc.Provenance.SpecIdentifier != "INVOIC:D:97A:UN" {
		t.Errorf("provenance = %+v", doc.Provenance)
	}
}

// TestInvoice2 couvre le cas limite d'un séparateur de composant en fin d'élément (RFF+VA:...:).
func TestInvoice2(t *testing.T) {
	doc := readSample(t, "invoice2.edi")
	if doc.Seller.VATID != "123844750" {
		t.Errorf("TVA vendeur = %q", doc.Seller.VATID)
	}
	if len(doc.Lines) != 1 || doc.Lines[0].NetAmount.String() != "1960.29" {
		t.Errorf("lignes = %+v", doc.Lines)
	}
}

// TestMessageNonInvoic refuse proprement un message non-INVOIC (ORDERS) sans paniquer.
func TestMessageNonInvoic(t *testing.T) {
	data, err := os.ReadFile(corpus("order.edi"))
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if _, err := Read(data, "order.edi"); err == nil {
		t.Fatal("un message ORDERS doit être refusé")
	}
}

// TestTokenizeUNA vérifie la prise en compte du segment de service UNA (séparateurs personnalisés)
// et du caractère d'échappement.
func TestTokenizeUNA(t *testing.T) {
	// UNA redéfinit : composant '|', donnée '#', décimal ',', échappement '?', réservé ' ', segment '\n'.
	src := "UNA|#,? \n" + "UNH#1#INVOIC|D|01B|UN\n" + "FTX#AAA###montant ?# net\n"
	segs, del, err := tokenize([]byte(src))
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	if del.data != '#' || del.component != '|' {
		t.Fatalf("délimiteurs mal lus : %+v", del)
	}
	if len(segs) != 2 || segs[0].tag != "UNH" || segs[1].tag != "FTX" {
		t.Fatalf("segments = %+v", segs)
	}
	if segs[0].comp(1, 0) != "INVOIC" || segs[0].comp(1, 2) != "01B" {
		t.Errorf("UNH mal éclaté : %+v", segs[0].elements)
	}
	// « ?# » est un « # » littéral échappé, pas un séparateur.
	if got := segs[1].comp(3, 0); got != "montant # net" {
		t.Errorf("échappement non respecté : %q", got)
	}
}

// TestEmpty refuse un flux vide.
func TestEmpty(t *testing.T) {
	if _, _, err := tokenize([]byte("   \n")); err == nil {
		t.Fatal("un flux vide doit produire une erreur")
	}
}
