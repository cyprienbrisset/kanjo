package facturx_test

import (
	"bytes"
	"strings"
	"testing"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/pdfa"
	"github.com/cyprienbrisset/kanjo/pkg/read/facturx"
	"github.com/cyprienbrisset/kanjo/pkg/write"

	_ "github.com/cyprienbrisset/kanjo/pkg/formats" // enregistre le lecteur CII pour le XML embarqué
	wcii "github.com/cyprienbrisset/kanjo/pkg/write/cii"
)

func basePDF(t *testing.T) []byte {
	t.Helper()
	const j = `{"pages":{"1":{"content":{"text":[{"value":"Facture F2026-0100","font":{"name":"Helvetica","size":12},"position":[100,700]}]}}}}`
	var buf bytes.Buffer
	if err := pdfapi.Create(nil, strings.NewReader(j), &buf, nil); err != nil {
		t.Fatalf("création PDF de base: %v", err)
	}
	return buf.Bytes()
}

func sampleCIIXML(t *testing.T) []byte {
	t.Helper()
	rate := model.MustParseDecimal("20")
	d := model.NewDocument(model.KindInvoice)
	d.ID = "F2026-0100"
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
		TaxableAmount: model.MustParseAmount("100.00", "EUR"),
		TaxAmount:     model.MustParseAmount("20.00", "EUR"),
	}}
	d.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("100.00", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("100.00", "EUR"),
		TaxAmount:           model.MustParseAmount("20.00", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("120.00", "EUR"),
		DuePayableAmount:    model.MustParseAmount("120.00", "EUR"),
	}
	b, err := wcii.Write(d, write.Options{Profile: write.ProfileEN16931})
	if err != nil {
		t.Fatalf("préparation CII: %v", err)
	}
	return b
}

// TestReadFacturX fabrique un PDF Factur-X (PDF/A-3 + CII embarqué) puis le relit.
func TestReadFacturX(t *testing.T) {
	res, err := pdfa.EmbedXML(basePDF(t), sampleCIIXML(t), "factur-x.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	doc, err := facturx.Read(res.PDF, "facture.pdf")
	if err != nil {
		t.Fatalf("lecture Factur-X: %v", err)
	}
	if doc.ID != "F2026-0100" {
		t.Errorf("identifiant = %q, veut F2026-0100", doc.ID)
	}
	if doc.Provenance == nil || doc.Provenance.SourceFormat != "facturx" {
		t.Errorf("format porteur non marqué facturx : %+v", doc.Provenance)
	}
}

// TestReadFacturXNonConformingName vérifie l'avertissement sur un nom de pièce jointe non standard.
func TestReadFacturXNonConformingName(t *testing.T) {
	res, err := pdfa.EmbedXML(basePDF(t), sampleCIIXML(t), "autre-nom.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	doc, err := facturx.Read(res.PDF, "facture.pdf")
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if len(doc.Provenance.Warnings) == 0 {
		t.Error("un avertissement était attendu pour un nom de pièce jointe non conforme")
	}
}

func TestReadFacturXNotAPDF(t *testing.T) {
	if _, err := facturx.Read([]byte("pas un pdf"), "x.pdf"); err == nil {
		t.Error("un contenu non-PDF devrait échouer")
	}
}
