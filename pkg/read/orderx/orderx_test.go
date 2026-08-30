package orderx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// TestReadOrderComfort lit un Order-X (profil comfort) réaliste écrit à la main.
func TestReadOrderComfort(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "order-comfort.xml"))
	if err != nil {
		t.Fatalf("lecture fixture: %v", err)
	}
	doc, err := Read(data, "order-comfort.xml")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if doc.Kind != model.KindOrder {
		t.Errorf("Kind = %q, veut order", doc.Kind)
	}
	if doc.TypeCode != model.TypeCode("220") {
		t.Errorf("TypeCode = %q, veut 220", doc.TypeCode)
	}
	if doc.ID != "PO-2026-0007" {
		t.Errorf("ID = %q", doc.ID)
	}
	if doc.CurrencyCode != "EUR" {
		t.Errorf("devise = %q", doc.CurrencyCode)
	}
	if doc.IssueDate.Year != 2026 || doc.IssueDate.Month != 9 || doc.IssueDate.Day != 14 {
		t.Errorf("date = %+v", doc.IssueDate)
	}
	if doc.Seller.Name != "Fournitures Bureau SARL" || doc.Seller.VATID != "FR40123456824" {
		t.Errorf("vendeur = %+v", doc.Seller)
	}
	if doc.Buyer.Name != "Acme Industries SA" || doc.Buyer.Address.City != "Paris" {
		t.Errorf("acheteur = %+v", doc.Buyer)
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("lignes = %d", len(doc.Lines))
	}
	l := doc.Lines[0]
	if l.Name != "Clavier mécanique" || l.Quantity.String() != "10" || l.UnitCode != model.UnitPiece {
		t.Errorf("ligne = %+v", l)
	}
	if l.NetPrice.String() != "89.90" {
		t.Errorf("prix net = %q", l.NetPrice.String())
	}
	if doc.Provenance == nil || doc.Provenance.SpecIdentifier != "urn:order-x.eu:1p0:comfort" {
		t.Errorf("provenance = %+v", doc.Provenance)
	}
}
