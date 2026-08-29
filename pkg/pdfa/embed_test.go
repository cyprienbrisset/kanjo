package pdfa

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// createBasePDF génère un PDF d'une page via pdfcpu, pour servir de fond aux tests d'embed.
func createBasePDF(t *testing.T) []byte {
	t.Helper()
	const j = `{"pages":{"1":{"content":{"text":[{"value":"Facture F2026-0042","font":{"name":"Helvetica","size":12},"position":[100,700]}]}}}}`
	var buf bytes.Buffer
	if err := api.Create(nil, strings.NewReader(j), &buf, config()); err != nil {
		t.Fatalf("création du PDF de base: %v", err)
	}
	return buf.Bytes()
}

func TestEmbedThenExtractRoundTrip(t *testing.T) {
	base := createBasePDF(t)
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?><rsm:CrossIndustryInvoice><X>ok</X></rsm:CrossIndustryInvoice>`)

	res, err := EmbedXML(base, xml, "factur-x.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if res.PDFAChecked {
		t.Error("PDFAChecked doit rester false tant que veraPDF n'a pas validé (§17.7)")
	}

	got, name, warn, err := ExtractInvoiceXML(res.PDF)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if name != "factur-x.xml" {
		t.Errorf("nom = %q, veut factur-x.xml", name)
	}
	if warn != "" {
		t.Errorf("avertissement inattendu : %s", warn)
	}
	if !bytes.Equal(got, xml) {
		t.Errorf("XML extrait diffère de l'original.\noriginal: %s\nextrait:  %s", xml, got)
	}
}
