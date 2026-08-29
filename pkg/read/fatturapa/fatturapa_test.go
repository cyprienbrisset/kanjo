package fatturapa_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/read/fatturapa"

	_ "github.com/cyprienbrisset/kanjo/pkg/formats" // enregistre le registre pour ReadBytes/Detect
)

const sample = `<?xml version="1.0" encoding="UTF-8"?>
<p:FatturaElettronica versione="FPR12" xmlns:p="http://ivaservizi.agenziaentrate.gov.it/docs/xsd/fatture/v1.2">
  <FatturaElettronicaHeader>
    <CedentePrestatore>
      <DatiAnagrafici>
        <IdFiscaleIVA><IdPaese>IT</IdPaese><IdCodice>12345678901</IdCodice></IdFiscaleIVA>
        <Anagrafica><Denominazione>Rossi SRL</Denominazione></Anagrafica>
      </DatiAnagrafici>
      <Sede><Indirizzo>Via Roma 1</Indirizzo><CAP>00100</CAP><Comune>Roma</Comune><Nazione>IT</Nazione></Sede>
    </CedentePrestatore>
    <CessionarioCommittente>
      <DatiAnagrafici>
        <IdFiscaleIVA><IdPaese>IT</IdPaese><IdCodice>98765432109</IdCodice></IdFiscaleIVA>
        <Anagrafica><Denominazione>Bianchi SPA</Denominazione></Anagrafica>
      </DatiAnagrafici>
      <Sede><Indirizzo>Corso Milano 2</Indirizzo><CAP>20100</CAP><Comune>Milano</Comune><Nazione>IT</Nazione></Sede>
    </CessionarioCommittente>
  </FatturaElettronicaHeader>
  <FatturaElettronicaBody>
    <DatiGenerali>
      <DatiGeneraliDocumento>
        <TipoDocumento>TD01</TipoDocumento>
        <Divisa>EUR</Divisa>
        <Data>2026-08-12</Data>
        <Numero>F2026-0100</Numero>
        <ImportoTotaleDocumento>122.00</ImportoTotaleDocumento>
      </DatiGeneraliDocumento>
    </DatiGenerali>
    <DatiBeniServizi>
      <DettaglioLinee>
        <NumeroLinea>1</NumeroLinea>
        <Descrizione>Consulenza</Descrizione>
        <Quantita>1.00</Quantita>
        <PrezzoUnitario>100.00</PrezzoUnitario>
        <PrezzoTotale>100.00</PrezzoTotale>
        <AliquotaIVA>22.00</AliquotaIVA>
      </DettaglioLinee>
      <DatiRiepilogo>
        <AliquotaIVA>22.00</AliquotaIVA>
        <ImponibileImporto>100.00</ImponibileImporto>
        <Imposta>22.00</Imposta>
      </DatiRiepilogo>
    </DatiBeniServizi>
  </FatturaElettronicaBody>
</p:FatturaElettronica>`

func TestDetectFatturaPA(t *testing.T) {
	if f := read.Detect([]byte(sample)); f != read.FormatFatturaPA {
		t.Errorf("format détecté = %s, veut fatturapa", f)
	}
}

func TestReadFatturaPA(t *testing.T) {
	doc, err := fatturapa.Read([]byte(sample), "fattura.xml")
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if doc.ID != "F2026-0100" {
		t.Errorf("ID = %q, veut F2026-0100", doc.ID)
	}
	if doc.CurrencyCode != "EUR" {
		t.Errorf("devise = %q, veut EUR", doc.CurrencyCode)
	}
	if doc.Seller.Name != "Rossi SRL" || doc.Seller.VATID != "IT12345678901" {
		t.Errorf("vendeur = %q / %q", doc.Seller.Name, doc.Seller.VATID)
	}
	if doc.Buyer.Name != "Bianchi SPA" {
		t.Errorf("acheteur = %q", doc.Buyer.Name)
	}
	if len(doc.Lines) != 1 || doc.Lines[0].Name != "Consulenza" {
		t.Fatalf("lignes = %+v", doc.Lines)
	}
	if doc.Lines[0].TaxCategory != model.TaxStandard {
		t.Errorf("catégorie ligne = %q, veut S", doc.Lines[0].TaxCategory)
	}
	if len(doc.TaxBreakdown) != 1 {
		t.Fatalf("ventilations = %d", len(doc.TaxBreakdown))
	}
	if got := doc.Totals.LineExtensionAmount.String(); got != "100.00" {
		t.Errorf("total lignes = %s, veut 100.00", got)
	}
	if got := doc.Totals.TaxAmount.String(); got != "22.00" {
		t.Errorf("total TVA = %s, veut 22.00", got)
	}
	if got := doc.Totals.TaxInclusiveAmount.String(); got != "122.00" {
		t.Errorf("total TTC = %s, veut 122.00", got)
	}
}

func TestReadFatturaPAViaRegistry(t *testing.T) {
	rd, err := read.ReadBytes([]byte(sample), "fattura.xml")
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if rd.Format != read.FormatFatturaPA {
		t.Errorf("format = %s, veut fatturapa", rd.Format)
	}
	if rd.Doc.ID != "F2026-0100" {
		t.Errorf("ID = %q", rd.Doc.ID)
	}
}
