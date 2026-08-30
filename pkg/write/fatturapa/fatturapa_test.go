package fatturapa_test

import (
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	rfatturapa "github.com/cyprienbrisset/kanjo/pkg/read/fatturapa"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	wfatturapa "github.com/cyprienbrisset/kanjo/pkg/write/fatturapa"
)

func sampleDoc() *model.Document {
	rate := model.MustParseDecimal("22")
	d := model.NewDocument(model.KindInvoice)
	d.ID = "IT2026-0007"
	d.TypeCode = model.TypeCommercialInvoice
	d.IssueDate, _ = model.ParseISO("2026-09-14")
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{
		Name: "Rossi SRL", VATID: "IT12345678900",
		Address: model.Address{Line1: "Via Roma 1", City: "Roma", PostalCode: "00100", CountryCode: "IT"},
	}
	d.Buyer = model.Party{
		Name: "Bianchi SPA", VATID: "IT98765432100",
		Address: model.Address{Line1: "Corso Milano 2", City: "Milano", PostalCode: "20100", CountryCode: "IT"},
	}
	d.Lines = []model.Line{
		{ID: "1", Name: "Consulenza", Quantity: model.DecimalFromInt(1), UnitCode: model.UnitPiece,
			NetPrice: model.MustParseAmount("100.00", "EUR"), TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("100.00", "EUR")},
	}
	d.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("100.00", "EUR"), TaxAmount: model.MustParseAmount("22.00", "EUR"),
	}}
	d.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("100.00", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("100.00", "EUR"),
		TaxAmount:           model.MustParseAmount("22.00", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("122.00", "EUR"),
		DuePayableAmount:    model.MustParseAmount("122.00", "EUR"),
	}
	return d
}

// TestWriteRoundTrip vérifie qu'un pivot sérialisé en FatturaPA puis relu conserve ses champs.
func TestWriteRoundTrip(t *testing.T) {
	src := sampleDoc()
	data, err := wfatturapa.Write(src, write.DefaultOptions())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	s := string(data)
	for _, want := range []string{
		`<p:FatturaElettronica`, `versione="FPR12"`, `<TipoDocumento>TD01</TipoDocumento>`,
		`<Numero>IT2026-0007</Numero>`, `<IdPaese>IT</IdPaese>`, `<IdCodice>12345678900</IdCodice>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("sortie sans %q\n---\n%s", want, s)
		}
	}

	got, err := rfatturapa.Read(data, "roundtrip.xml")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.ID != src.ID {
		t.Errorf("ID = %q", got.ID)
	}
	if got.TypeCode != model.TypeCommercialInvoice {
		t.Errorf("TypeCode = %q", got.TypeCode)
	}
	if got.CurrencyCode != "EUR" || got.IssueDate != src.IssueDate {
		t.Errorf("devise/date = %q / %+v", got.CurrencyCode, got.IssueDate)
	}
	if got.Seller.Name != "Rossi SRL" || got.Seller.VATID != "IT12345678900" {
		t.Errorf("vendeur = %+v", got.Seller)
	}
	if got.Seller.Address.City != "Roma" || got.Seller.Address.CountryCode != "IT" {
		t.Errorf("adresse vendeur = %+v", got.Seller.Address)
	}
	if got.Buyer.Name != "Bianchi SPA" || got.Buyer.VATID != "IT98765432100" {
		t.Errorf("acheteur = %+v", got.Buyer)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("lignes = %d", len(got.Lines))
	}
	l := got.Lines[0]
	if l.Name != "Consulenza" || l.NetAmount.String() != "100.00" || l.TaxCategory != model.TaxStandard {
		t.Errorf("ligne = %+v", l)
	}
	if got.Totals.TaxInclusiveAmount.String() != "122.00" {
		t.Errorf("TTC = %q", got.Totals.TaxInclusiveAmount.String())
	}
	if got.Totals.TaxAmount.String() != "22.00" {
		t.Errorf("TVA = %q", got.Totals.TaxAmount.String())
	}
}

// TestWriteCreditNote vérifie qu'un avoir devient TD04.
func TestWriteCreditNote(t *testing.T) {
	d := sampleDoc()
	d.Kind = model.KindCreditNote
	d.TypeCode = model.TypeCreditNote
	data, err := wfatturapa.Write(d, write.DefaultOptions())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.Contains(string(data), "<TipoDocumento>TD04</TipoDocumento>") {
		t.Errorf("un avoir doit être TD04")
	}
	got, err := rfatturapa.Read(data, "cn.xml")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.TypeCode != model.TypeCreditNote {
		t.Errorf("TypeCode relu = %q, veut avoir", got.TypeCode)
	}
}
