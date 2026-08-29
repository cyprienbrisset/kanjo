package cii_test

import (
	"bytes"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	wcii "github.com/cyprienbrisset/kanjo/pkg/write/cii"
)

func sampleDoc() *model.Document {
	rate := model.MustParseDecimal("20")
	d := model.NewDocument(model.KindInvoice)
	d.ID = "F2026-0042"
	d.TypeCode = model.TypeCommercialInvoice
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

func TestWriteCII(t *testing.T) {
	out, err := wcii.Write(sampleDoc(), write.Options{Profile: write.ProfileEN16931})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, want := range []string{
		"CrossIndustryInvoice", "F2026-0042", "EUR", "SAS Martin",
		"Société Cliente", "Conseil", "100.00", "20.00",
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("sortie CII ne contient pas %q", want)
		}
	}
}

func TestWriteCIIProfiles(t *testing.T) {
	for _, p := range []write.Profile{write.ProfileEN16931, write.ProfileBasic, write.ProfileExtended} {
		out, err := wcii.Write(sampleDoc(), write.Options{Profile: p})
		if err != nil {
			t.Errorf("profil %s: %v", p, err)
			continue
		}
		if len(out) == 0 {
			t.Errorf("profil %s: sortie vide", p)
		}
	}
}
