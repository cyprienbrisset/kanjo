package ubl_test

import (
	"bytes"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	wubl "github.com/cyprienbrisset/kanjo/pkg/write/ubl"
)

func sampleDoc(kind model.DocumentKind) *model.Document {
	rate := model.MustParseDecimal("20")
	d := model.NewDocument(kind)
	d.ID = "F2026-0042"
	d.TypeCode = model.TypeCommercialInvoice
	if kind == model.KindCreditNote {
		d.TypeCode = model.TypeCreditNote
	}
	d.IssueDate, _ = model.ParseISO("2026-08-12")
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{Name: "SAS Martin", VATID: "FR12501234567", Address: model.Address{CountryCode: "FR"}}
	d.Buyer = model.Party{Name: "Société Cliente", Address: model.Address{CountryCode: "FR"}}
	d.Lines = []model.Line{{
		ID: "1", Name: "Conseil", Quantity: model.DecimalFromInt(1), UnitCode: model.UnitPiece,
		NetPrice: model.MustParseAmount("100.00", "EUR"), TaxCategory: model.TaxStandard,
		TaxRate: &rate, NetAmount: model.MustParseAmount("100.00", "EUR"),
	}}
	d.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("100.00", "EUR"), TaxAmount: model.MustParseAmount("20.00", "EUR"),
	}}
	d.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("100.00", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("100.00", "EUR"),
		TaxAmount:           model.MustParseAmount("20.00", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("120.00", "EUR"),
		DuePayableAmount:    model.MustParseAmount("120.00", "EUR"),
	}
	return d
}

func TestWriteUBLInvoice(t *testing.T) {
	out, err := wubl.Write(sampleDoc(model.KindInvoice), write.Options{Profile: write.ProfileEN16931})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, want := range []string{"<Invoice", "F2026-0042", "EUR", "SAS Martin", "Conseil"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("sortie UBL ne contient pas %q", want)
		}
	}
}

func TestWriteUBLCreditNote(t *testing.T) {
	out, err := wubl.Write(sampleDoc(model.KindCreditNote), write.Options{Profile: write.ProfileEN16931})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Contains(out, []byte("<CreditNote")) {
		t.Errorf("un avoir doit produire un élément CreditNote :\n%s", out)
	}
}
