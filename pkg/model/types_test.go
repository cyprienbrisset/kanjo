package model

import (
	"testing"
	"time"
)

func TestCodesValidAndLabel(t *testing.T) {
	if !TypeCommercialInvoice.Valid() || TypeCode("999").Valid() {
		t.Error("TypeCode.Valid incorrect")
	}
	if TypeCommercialInvoice.Label(LangFR) != "Facture commerciale" {
		t.Errorf("label FR = %q", TypeCommercialInvoice.Label(LangFR))
	}
	if TypeCommercialInvoice.Label(LangEN) != "Commercial invoice" {
		t.Errorf("label EN = %q", TypeCommercialInvoice.Label(LangEN))
	}
	// Repli sur le code brut pour un code sans libellé.
	if got := TypeCode("XYZ").Label(LangFR); got != "XYZ" {
		t.Errorf("repli label = %q", got)
	}
	if !TypeCreditNote.IsCreditNote() || TypeCommercialInvoice.IsCreditNote() {
		t.Error("IsCreditNote incorrect")
	}

	if !TaxStandard.Valid() || TaxCategoryCode("Q").Valid() {
		t.Error("TaxCategoryCode.Valid incorrect")
	}
	if !TaxStandard.RequiresRate() || TaxExempt.RequiresRate() {
		t.Error("RequiresRate incorrect")
	}
	if TaxReverseCharge.Label(LangFR) != "Autoliquidation" {
		t.Errorf("label TVA = %q", TaxReverseCharge.Label(LangFR))
	}

	if !PayCredit.Valid() || PaymentMeansCode("").Valid() {
		t.Error("PaymentMeansCode.Valid incorrect")
	}
	if PayCredit.Label(LangFR) != "Virement" {
		t.Errorf("label paiement = %q", PayCredit.Label(LangFR))
	}
	if !UnitPiece.Valid() || UnitCode("").Valid() {
		t.Error("UnitCode.Valid incorrect")
	}
}

func TestEnumsValid(t *testing.T) {
	if !KindInvoice.Valid() || DocumentKind("x").Valid() {
		t.Error("DocumentKind.Valid incorrect")
	}
	if !OpGoods.Valid() || !OperationCat("").Valid() || OperationCat("z").Valid() {
		t.Error("OperationCat.Valid incorrect")
	}
}

func TestDateHelpers(t *testing.T) {
	d, _ := NewDate(2026, time.September, 1)
	if d.IsZero() {
		t.Error("date non nulle considérée nulle")
	}
	if (Date{}).IsZero() != true {
		t.Error("date nulle non détectée")
	}
	if d.Compact() != "20260901" || d.String() != "2026-09-01" {
		t.Errorf("formats date : %s / %s", d.Compact(), d.String())
	}
	if _, err := NewDate(2026, time.February, 30); err == nil {
		t.Error("30 février devrait être invalide")
	}
}

func TestAmountHelpers(t *testing.T) {
	z := ZeroAmount("EUR")
	if !z.IsZero() {
		t.Error("ZeroAmount non nul")
	}
	a := MustParseAmount("10.00", "EUR")
	if a.Neg().String() != "-10.00" {
		t.Errorf("Neg = %s", a.Neg())
	}
	sum, err := SumAmounts("EUR", a, MustParseAmount("5.50", "EUR"), MustParseAmount("0.50", "EUR"))
	if err != nil {
		t.Fatal(err)
	}
	if sum.String() != "16.00" {
		t.Errorf("SumAmounts = %s", sum)
	}
	if _, err := SumAmounts("EUR", MustParseAmount("1.00", "USD")); err == nil {
		t.Error("somme de devises incompatibles devrait échouer")
	}
	if _, err := a.Cmp(MustParseAmount("1.00", "USD")); err == nil {
		t.Error("Cmp de devises incompatibles devrait échouer")
	}
}

func TestDecimalHelpers(t *testing.T) {
	if !NewDecimal(0, 2).IsZero() {
		t.Error("décimal nul non détecté")
	}
	if DecimalFromInt(5).String() != "5" {
		t.Errorf("DecimalFromInt = %s", DecimalFromInt(5))
	}
}

func TestExtensionsAndAddress(t *testing.T) {
	if !(Extensions{}).Empty() {
		t.Error("Extensions vides non détectées")
	}
	e := Extensions{FR: &FrenchCTC{SellerSIREN: "501234567"}}
	if e.Empty() {
		t.Error("Extensions non vides considérées vides")
	}
	if !(Address{}).Empty() {
		t.Error("adresse vide non détectée")
	}
}

func TestProvenanceRecord(t *testing.T) {
	p := NewProvenance("f.xml", "cii", "en16931")
	p.Record("BT-1", "/x/ram:ID")
	if p.FieldOrigins["BT-1"] != "/x/ram:ID" {
		t.Error("Record n'a pas enregistré l'origine")
	}
	p.Warn(ReadWarning{Code: "W-ENC-001", Message: "encodage converti"})
	if len(p.Warnings) != 1 {
		t.Error("Warn n'a pas ajouté d'avertissement")
	}
	// Sûr sur pointeur nil.
	var nilp *Provenance
	nilp.Record("BT-2", "x")
	nilp.Warn(ReadWarning{})
}

func TestDocumentAccessors(t *testing.T) {
	d := NewDocument(KindInvoice)
	d.CurrencyCode = "EUR"
	d.TypeCode = TypeCreditNote
	if d.Currency() != "EUR" {
		t.Error("Currency incorrect")
	}
	if !d.IsCreditNote() {
		t.Error("IsCreditNote (par TypeCode) incorrect")
	}
	d.Lines = []Line{{ID: "1", NetAmount: MustParseAmount("100.00", "EUR")}, {ID: "2", NetAmount: MustParseAmount("50.00", "EUR")}}
	if d.LineByID("2") == nil || d.LineByID("9") != nil {
		t.Error("LineByID incorrect")
	}
	sum, err := d.SumLineNetAmounts()
	if err != nil || sum.String() != "150.00" {
		t.Errorf("SumLineNetAmounts = %s (%v)", sum, err)
	}
	d.TaxBreakdown = []TaxSubtotal{{TaxAmount: MustParseAmount("20.00", "EUR")}, {TaxAmount: MustParseAmount("10.00", "EUR")}}
	tsum, err := d.SumTaxAmounts()
	if err != nil || tsum.String() != "30.00" {
		t.Errorf("SumTaxAmounts = %s (%v)", tsum, err)
	}
}

func TestAllowanceSigned(t *testing.T) {
	charge := AllowanceCharge{IsCharge: true, Amount: MustParseAmount("5.00", "EUR")}
	if charge.Signed().String() != "5.00" {
		t.Errorf("charge signée = %s", charge.Signed())
	}
	rebate := AllowanceCharge{IsCharge: false, Amount: MustParseAmount("5.00", "EUR")}
	if rebate.Signed().String() != "-5.00" {
		t.Errorf("remise signée = %s", rebate.Signed())
	}
}

func TestTotalsWithAllowancesChargesPrepaid(t *testing.T) {
	al := MustParseAmount("10.00", "EUR")
	ch := MustParseAmount("4.00", "EUR")
	pp := MustParseAmount("100.00", "EUR")
	tot := Totals{
		LineExtensionAmount: MustParseAmount("1000.00", "EUR"),
		AllowanceTotal:      &al,
		ChargeTotal:         &ch,
		TaxExclusiveAmount:  MustParseAmount("994.00", "EUR"), // 1000 - 10 + 4
		TaxAmount:           MustParseAmount("198.80", "EUR"),
		TaxInclusiveAmount:  MustParseAmount("1192.80", "EUR"),
		PrepaidAmount:       &pp,
		DuePayableAmount:    MustParseAmount("1092.80", "EUR"), // 1192.80 - 100
	}
	ht, _ := tot.ComputeTaxExclusive("EUR")
	if !ht.Equal(tot.TaxExclusiveAmount) {
		t.Errorf("HT calculé = %s", ht)
	}
	due, _ := tot.ComputeDuePayable("EUR")
	if !due.Equal(tot.DuePayableAmount) {
		t.Errorf("net à payer calculé = %s", due)
	}
}

func TestAttachmentHasData(t *testing.T) {
	if (Attachment{}).HasEmbeddedData() {
		t.Error("pièce jointe vide considérée avec données")
	}
	if !(Attachment{Data: []byte("x")}).HasEmbeddedData() {
		t.Error("pièce jointe avec données non détectée")
	}
}
