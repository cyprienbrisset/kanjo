package xrechnung_test

import (
	"bytes"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	rubl "github.com/cyprienbrisset/kanjo/pkg/read/ubl"
	"github.com/cyprienbrisset/kanjo/pkg/write"

	// Enregistre le writer xrechnung (et, en cascade, ubl/cii/json).
	_ "github.com/cyprienbrisset/kanjo/pkg/write/xrechnung"
)

// customizationXRechnung30 est l'identifiant de spécification attendu de XRechnung 3.0.
const customizationXRechnung30 = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_3.0"

// buildSample construit une facture EN 16931 représentative (copie locale de la fixture du
// round-trip CII), ne peuplant que des champs pris en charge par les readers/writers.
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

func TestXRechnungUBLCustomizationID(t *testing.T) {
	out, err := write.WriteBytes("xrechnung", buildSample(), write.Options{Syntax: "ubl"})
	if err != nil {
		t.Fatalf("écriture xrechnung/ubl: %v", err)
	}
	if !bytes.Contains(out, []byte(customizationXRechnung30)) {
		t.Errorf("CustomizationID XRechnung absent de la sortie UBL:\n%s", out)
	}
	if !bytes.Contains(out, []byte("cbc:CustomizationID")) {
		t.Errorf("sortie UBL attendue (cbc:CustomizationID absent):\n%s", out)
	}
}

func TestXRechnungCIICustomizationID(t *testing.T) {
	out, err := write.WriteBytes("xrechnung", buildSample(), write.Options{Syntax: "cii"})
	if err != nil {
		t.Fatalf("écriture xrechnung/cii: %v", err)
	}
	if !bytes.Contains(out, []byte(customizationXRechnung30)) {
		t.Errorf("CustomizationID XRechnung absent de la sortie CII:\n%s", out)
	}
	if !bytes.Contains(out, []byte("rsm:CrossIndustryInvoice")) {
		t.Errorf("sortie CII attendue:\n%s", out)
	}
}

// TestXRechnungUBLRoundTrip prouve que la sortie xrechnung/ubl reste relisable par le reader
// UBL et que le pivot JSON est inchangé (aucune structure cassée par l'override).
func TestXRechnungUBLRoundTrip(t *testing.T) {
	original := buildSample()
	out, err := write.WriteBytes("xrechnung", original, write.Options{Syntax: "ubl"})
	if err != nil {
		t.Fatalf("écriture xrechnung/ubl: %v", err)
	}
	reparsed, err := rubl.Read(out, "xrechnung-ubl.xml")
	if err != nil {
		t.Fatalf("relecture UBL: %v\n--- XML ---\n%s", err, out)
	}
	want := jsonOf(t, original)
	got := jsonOf(t, reparsed)
	if !bytes.Equal(want, got) {
		t.Errorf("aller-retour non lossless.\n=== attendu ===\n%s\n=== obtenu ===\n%s", want, got)
	}
}
