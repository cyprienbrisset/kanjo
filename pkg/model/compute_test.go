package model

import "testing"

func TestLineComputeNetAmount(t *testing.T) {
	l := Line{
		Quantity:  DecimalFromInt(3),
		NetPrice:  MustParseAmount("19.99", "EUR"),
		NetAmount: MustParseAmount("59.97", "EUR"),
	}
	got, err := l.ComputeNetAmount("EUR")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "59.97" {
		t.Errorf("net ligne = %s, veut 59.97", got)
	}
}

func TestLineComputeWithBaseQuantityAndDiscount(t *testing.T) {
	base := MustParseDecimal("100")
	l := Line{
		Quantity:     MustParseDecimal("250"),
		NetPrice:     MustParseAmount("12.00", "EUR"), // 12,00 € pour 100 unités
		PriceBaseQty: &base,
		AllowanceCharges: []AllowanceCharge{
			{IsCharge: false, Amount: MustParseAmount("0.50", "EUR")},
		},
	}
	// 12,00 / 100 = 0,12 → × 250 = 30,00 → − 0,50 = 29,50
	got, err := l.ComputeNetAmount("EUR")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "29.50" {
		t.Errorf("net ligne = %s, veut 29.50", got)
	}
}

func TestTaxSubtotalComputeTaxAmount(t *testing.T) {
	ts := TaxSubtotal{
		Category:      TaxStandard,
		Rate:          MustParseDecimal("20"),
		TaxableAmount: MustParseAmount("1249.90", "EUR"),
	}
	got := ts.ComputeTaxAmount()
	if got.String() != "249.98" {
		t.Errorf("TVA = %s, veut 249.98", got)
	}
}

func TestTotalsCoherence(t *testing.T) {
	tot := Totals{
		LineExtensionAmount: MustParseAmount("1249.90", "EUR"),
		TaxExclusiveAmount:  MustParseAmount("1249.90", "EUR"),
		TaxAmount:           MustParseAmount("249.98", "EUR"),
		TaxInclusiveAmount:  MustParseAmount("1499.88", "EUR"),
		DuePayableAmount:    MustParseAmount("1499.88", "EUR"),
	}
	ttc, err := tot.ComputeTaxInclusive()
	if err != nil {
		t.Fatal(err)
	}
	if !ttc.Equal(tot.TaxInclusiveAmount) {
		t.Errorf("TTC calculé = %s, veut %s", ttc, tot.TaxInclusiveAmount)
	}
	ht, err := tot.ComputeTaxExclusive("EUR")
	if err != nil {
		t.Fatal(err)
	}
	if !ht.Equal(tot.TaxExclusiveAmount) {
		t.Errorf("HT calculé = %s, veut %s", ht, tot.TaxExclusiveAmount)
	}
	due, err := tot.ComputeDuePayable("EUR")
	if err != nil {
		t.Fatal(err)
	}
	if !due.Equal(tot.DuePayableAmount) {
		t.Errorf("net à payer calculé = %s, veut %s", due, tot.DuePayableAmount)
	}
}
