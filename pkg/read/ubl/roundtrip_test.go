package ubl_test

import (
	"bytes"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	rubl "github.com/cyprienbrisset/kanjo/pkg/read/ubl"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	wubl "github.com/cyprienbrisset/kanjo/pkg/write/ubl"
)

// buildSample construit une facture EN 16931 représentative, ne peuplant que des champs
// pris en charge par le reader et le writer UBL, afin de tester l'aller-retour lossless (O1).
func buildSample() *model.Document {
	rate := model.MustParseDecimal("20")
	due, _ := model.ParseISO("2026-09-11")
	doc := model.NewDocument(model.KindInvoice)
	doc.ID = "F2026-0042"
	doc.IssueDate, _ = model.ParseISO("2026-08-12")
	doc.TypeCode = model.TypeCommercialInvoice
	doc.CurrencyCode = "EUR"
	doc.BuyerReference = "SERVICE-COMPTA"
	doc.PurchaseOrderRef = "CMD-2026-99"
	doc.Notes = []model.Note{{Content: "Merci de votre confiance."}}

	doc.Seller = model.Party{
		Name:          "SAS Martin",
		LegalID:       "501234567",
		LegalIDScheme: "0002",
		VATID:         "FR12501234567",
		Address: model.Address{
			Line1: "12 rue des Comptes", PostalCode: "75001", City: "Paris", CountryCode: "FR",
		},
		Contact: &model.Contact{Name: "Jean Martin", Email: "compta@martin.fr"},
	}
	doc.Buyer = model.Party{
		Name: "Société Cliente",
		Address: model.Address{
			Line1: "1 avenue du SaaS", PostalCode: "69002", City: "Lyon", CountryCode: "FR",
		},
	}
	doc.DeliverTo = &model.DeliveryInfo{
		Name:    "Société Cliente Entrepôt",
		Address: model.Address{City: "Lyon", PostalCode: "69003", CountryCode: "FR"},
	}

	doc.Lines = []model.Line{
		{
			ID: "1", Name: "Prestation de conseil", Quantity: model.DecimalFromInt(2),
			UnitCode: model.UnitPiece, NetPrice: model.MustParseAmount("500.00", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("1000.00", "EUR"),
		},
		{
			ID: "2", Name: "Licence annuelle", Quantity: model.DecimalFromInt(1),
			UnitCode: model.UnitPiece, NetPrice: model.MustParseAmount("249.90", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("249.90", "EUR"),
		},
	}
	doc.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:     model.MustParseAmount("249.98", "EUR"),
	}}
	doc.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:           model.MustParseAmount("249.98", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("1499.88", "EUR"),
		DuePayableAmount:    model.MustParseAmount("1499.88", "EUR"),
	}
	doc.PaymentTerms = "Paiement à 30 jours"
	doc.DueDate = &due
	doc.PaymentInstructions = &model.PaymentInstructions{
		MeansCode:       model.PayCredit,
		RemittanceInfo:  "F2026-0042",
		CreditTransfers: []model.CreditTransfer{{IBAN: "FR7630006000011234567890189"}},
	}
	return doc
}

func jsonOf(t *testing.T, doc *model.Document) []byte {
	t.Helper()
	b, err := write.WriteBytes("json", doc, write.Options{Indent: true})
	if err != nil {
		t.Fatalf("sérialisation JSON: %v", err)
	}
	return b
}

func TestUBLRoundTripLossless(t *testing.T) {
	original := buildSample()

	xml, err := wubl.Write(original, write.Options{Profile: write.ProfileEN16931})
	if err != nil {
		t.Fatalf("écriture UBL: %v", err)
	}

	reparsed, err := rubl.Read(xml, "roundtrip.xml")
	if err != nil {
		t.Fatalf("relecture UBL: %v\n--- XML ---\n%s", err, xml)
	}

	want := jsonOf(t, original)
	got := jsonOf(t, reparsed)
	if !bytes.Equal(want, got) {
		t.Errorf("aller-retour non lossless.\n=== attendu ===\n%s\n=== obtenu ===\n%s\n=== XML ===\n%s", want, got, xml)
	}
}

func TestUBLDetectedAsInvoice(t *testing.T) {
	xml, _ := wubl.Write(buildSample(), write.Options{Profile: write.ProfileEN16931})
	if f := read.Detect(xml); f != read.FormatUBLInvoice {
		t.Errorf("format détecté = %s, veut %s", f, read.FormatUBLInvoice)
	}
}

func buildCreditNoteSample() *model.Document {
	doc := buildSample()
	doc.Kind = model.KindCreditNote
	doc.TypeCode = model.TypeCreditNote
	doc.ID = "AV2026-0007"
	return doc
}

func TestUBLCreditNoteRoundTripLossless(t *testing.T) {
	original := buildCreditNoteSample()

	xml, err := wubl.Write(original, write.Options{Profile: write.ProfileEN16931})
	if err != nil {
		t.Fatalf("écriture UBL avoir: %v", err)
	}

	reparsed, err := rubl.Read(xml, "roundtrip-cn.xml")
	if err != nil {
		t.Fatalf("relecture UBL avoir: %v\n--- XML ---\n%s", err, xml)
	}

	want := jsonOf(t, original)
	got := jsonOf(t, reparsed)
	if !bytes.Equal(want, got) {
		t.Errorf("aller-retour avoir non lossless.\n=== attendu ===\n%s\n=== obtenu ===\n%s\n=== XML ===\n%s", want, got, xml)
	}
}

func TestUBLCreditNoteDetected(t *testing.T) {
	xml, _ := wubl.Write(buildCreditNoteSample(), write.Options{Profile: write.ProfileEN16931})
	if f := read.Detect(xml); f != read.FormatUBLCreditNote {
		t.Errorf("format détecté = %s, veut %s", f, read.FormatUBLCreditNote)
	}
}
