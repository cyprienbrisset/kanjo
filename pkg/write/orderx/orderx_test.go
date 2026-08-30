package orderx_test

import (
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	rorderx "github.com/cyprienbrisset/kanjo/pkg/read/orderx"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	worderx "github.com/cyprienbrisset/kanjo/pkg/write/orderx"
)

func sampleOrder() *model.Document {
	d := model.NewDocument(model.KindOrder)
	d.ID = "PO-2026-0042"
	d.TypeCode = model.TypeCode("220")
	d.IssueDate, _ = model.ParseISO("2026-09-14")
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{
		Name: "Fournitures Bureau SARL", VATID: "FR40123456824",
		Address: model.Address{City: "Lyon", PostalCode: "69002", CountryCode: "FR"},
	}
	d.Buyer = model.Party{
		Name:    "Acme Industries SA",
		Address: model.Address{City: "Paris", PostalCode: "75008", CountryCode: "FR"},
	}
	d.Lines = []model.Line{
		{ID: "1", Name: "Clavier mécanique", Quantity: model.DecimalFromInt(10),
			UnitCode: model.UnitPiece, NetPrice: model.MustParseAmount("89.90", "EUR")},
		{ID: "2", Name: "Souris ergonomique", Quantity: model.DecimalFromInt(5),
			UnitCode: model.UnitPiece, NetPrice: model.MustParseAmount("45.00", "EUR")},
	}
	return d
}

// TestWriteRoundTrip vérifie qu'une commande sérialisée en Order-X puis relue conserve ses champs.
func TestWriteRoundTrip(t *testing.T) {
	src := sampleOrder()
	data, err := worderx.Write(src, write.DefaultOptions())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	s := string(data)
	for _, want := range []string{
		"<rsm:SCRDMCCBDACIOMESSAGE", "<ram:TypeCode>220</ram:TypeCode>",
		"<ram:ID>PO-2026-0042</ram:ID>", "<ram:OrderCurrencyCode>EUR</ram:OrderCurrencyCode>",
		`<ram:RequestedQuantity unitCode="C62">10</ram:RequestedQuantity>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("sortie sans %q\n---\n%s", want, s)
		}
	}

	got, err := rorderx.Read(data, "roundtrip.xml")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Kind != model.KindOrder || got.TypeCode != model.TypeCode("220") {
		t.Errorf("kind/type = %q / %q", got.Kind, got.TypeCode)
	}
	if got.ID != src.ID || got.CurrencyCode != "EUR" || got.IssueDate != src.IssueDate {
		t.Errorf("entête = %q / %q / %+v", got.ID, got.CurrencyCode, got.IssueDate)
	}
	if got.Seller.Name != src.Seller.Name || got.Seller.VATID != src.Seller.VATID {
		t.Errorf("vendeur = %+v", got.Seller)
	}
	if got.Buyer.Name != src.Buyer.Name || got.Buyer.Address.City != "Paris" {
		t.Errorf("acheteur = %+v", got.Buyer)
	}
	if len(got.Lines) != 2 {
		t.Fatalf("lignes = %d", len(got.Lines))
	}
	if got.Lines[1].Name != "Souris ergonomique" || got.Lines[1].Quantity.String() != "5" ||
		got.Lines[1].NetPrice.String() != "45.00" {
		t.Errorf("ligne 2 = %+v", got.Lines[1])
	}
}
