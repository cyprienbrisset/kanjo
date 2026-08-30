package edifact_test

import (
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	redifact "github.com/cyprienbrisset/kanjo/pkg/read/edifact"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	wedifact "github.com/cyprienbrisset/kanjo/pkg/write/edifact"
)

// sampleDoc construit une facture pivot cohérente (2 lignes, TVA 20 %).
func sampleDoc() *model.Document {
	rate := model.MustParseDecimal("20")
	d := model.NewDocument(model.KindInvoice)
	d.ID = "F2026-0042"
	d.TypeCode = model.TypeCommercialInvoice
	d.IssueDate, _ = model.ParseISO("2026-09-14")
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{
		Name: "SA Meunier", VATID: "FR81627131848",
		Address: model.Address{Line1: "1 rue du Test", City: "Bordeaux", PostalCode: "69000", CountryCode: "FR"},
	}
	d.Buyer = model.Party{
		Name: "Société Cliente", VATID: "FR12345678901",
		Address: model.Address{Line1: "2 avenue Client", City: "Paris", PostalCode: "75000", CountryCode: "FR"},
	}
	d.Lines = []model.Line{
		{ID: "1", Name: "Audit de sécurité", Quantity: model.DecimalFromInt(1), UnitCode: model.UnitPiece,
			NetPrice: model.MustParseAmount("116.94", "EUR"), TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("116.94", "EUR")},
		{ID: "2", Name: "Licence annuelle", Quantity: model.DecimalFromInt(3), UnitCode: model.UnitPiece,
			NetPrice: model.MustParseAmount("60.89", "EUR"), TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("182.67", "EUR")},
	}
	d.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("299.61", "EUR"), TaxAmount: model.MustParseAmount("59.92", "EUR"),
	}}
	d.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("299.61", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("299.61", "EUR"),
		TaxAmount:           model.MustParseAmount("59.92", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("359.53", "EUR"),
		DuePayableAmount:    model.MustParseAmount("359.53", "EUR"),
	}
	return d
}

// TestWriteRoundTrip vérifie qu'un document pivot sérialisé en EDIFACT puis relu conserve les
// champs transportables par INVOIC.
func TestWriteRoundTrip(t *testing.T) {
	src := sampleDoc()
	data, err := wedifact.Write(src, write.DefaultOptions())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Interchange bien formé.
	s := string(data)
	for _, seg := range []string{"UNB+", "UNH+1+INVOIC:D:96A:UN'", "BGM+380+F2026-0042", "UNT+", "UNZ+1+"} {
		if !strings.Contains(s, seg) {
			t.Errorf("segment attendu absent : %q\n---\n%s", seg, s)
		}
	}

	got, err := redifact.Read(data, "roundtrip.edi")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.ID != src.ID {
		t.Errorf("ID = %q, veut %q", got.ID, src.ID)
	}
	if got.TypeCode != src.TypeCode {
		t.Errorf("TypeCode = %q, veut %q", got.TypeCode, src.TypeCode)
	}
	if got.CurrencyCode != src.CurrencyCode {
		t.Errorf("devise = %q", got.CurrencyCode)
	}
	if got.IssueDate != src.IssueDate {
		t.Errorf("date = %+v, veut %+v", got.IssueDate, src.IssueDate)
	}
	if got.Seller.Name != src.Seller.Name || got.Seller.VATID != src.Seller.VATID {
		t.Errorf("vendeur = %+v", got.Seller)
	}
	if got.Buyer.Name != src.Buyer.Name || got.Buyer.VATID != src.Buyer.VATID {
		t.Errorf("acheteur = %+v", got.Buyer)
	}
	if got.Seller.Address.City != "Bordeaux" || got.Seller.Address.CountryCode != "FR" {
		t.Errorf("adresse vendeur = %+v", got.Seller.Address)
	}
	if len(got.Lines) != 2 {
		t.Fatalf("lignes = %d, veut 2", len(got.Lines))
	}
	l := got.Lines[1]
	if l.Name != "Licence annuelle" || l.Quantity.String() != "3" || l.NetAmount.String() != "182.67" {
		t.Errorf("ligne 2 = %+v", l)
	}
	if l.TaxCategory != model.TaxStandard || l.TaxRate == nil || l.TaxRate.String() != "20" {
		t.Errorf("TVA ligne 2 = %v / %v", l.TaxCategory, l.TaxRate)
	}
	if got.Totals.TaxInclusiveAmount.String() != "359.53" {
		t.Errorf("TTC = %q", got.Totals.TaxInclusiveAmount.String())
	}
	if got.Totals.DuePayableAmount.String() != "359.53" {
		t.Errorf("net à payer = %q", got.Totals.DuePayableAmount.String())
	}
	if got.Totals.TaxAmount.String() != "59.92" {
		t.Errorf("TVA totale = %q", got.Totals.TaxAmount.String())
	}
}

// TestWriteEscaping vérifie que les séparateurs de service présents dans une valeur sont échappés
// et correctement re-décodés (aller-retour sur un nom contenant « + » et « : »).
func TestWriteEscaping(t *testing.T) {
	d := sampleDoc()
	d.Seller.Name = "A+B : C"
	data, err := wedifact.Write(d, write.DefaultOptions())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.Contains(string(data), "A?+B ?: C") {
		t.Errorf("nom non échappé dans la sortie :\n%s", data)
	}
	got, err := redifact.Read(data, "esc.edi")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Seller.Name != "A+B : C" {
		t.Errorf("nom après aller-retour = %q, veut %q", got.Seller.Name, "A+B : C")
	}
}
