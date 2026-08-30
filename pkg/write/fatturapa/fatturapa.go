// Package fatturapa sérialise un document pivot vers le format italien FatturaPA
// (FatturaElettronica v1.2). C'est le symétrique du lecteur pkg/read/fatturapa : un aller-retour
// pivot→FatturaPA→pivot préserve en-tête, parties, lignes, ventilation de TVA, totaux et paiement.
//
// Limite assumée (§17.7) : les extensions purement italiennes non modélisées dans le pivot (bollo,
// ritenuta, cassa, CIG/CUP…) ne sont pas émises. La Natura est déduite de la catégorie de TVA
// EN 16931 pour les opérations à taux zéro.
package fatturapa

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func init() { write.Register("fatturapa", Write) }

const nsFatturaPA = "http://ivaservizi.agenziaentrate.gov.it/docs/xsd/fatture/v1.2"

// Write produit un document FatturaElettronica v1.2 (profil FPR12, entre particuliers/entreprises).
func Write(doc *model.Document, _ write.Options) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("fatturapa: document nil")
	}

	tipo := "TD01" // fattura
	if doc.TypeCode.IsCreditNote() {
		tipo = "TD04" // nota di credito
	}
	cur := doc.CurrencyCode
	if cur == "" {
		cur = "EUR"
	}

	r := root{
		Xmlns:    nsFatturaPA,
		Versione: "FPR12",
	}
	r.Header.Cedente = party(doc.Seller)
	r.Header.Cessionario = party(doc.Buyer)

	r.Body.DatiGenerali.Documento = documento{
		TipoDocumento: tipo,
		Divisa:        cur,
		Data:          isoDate(doc.IssueDate),
		Numero:        doc.ID,
		ImportoTotale: amt2(doc.Totals.TaxInclusiveAmount),
	}

	for _, l := range doc.Lines {
		line := dettaglioLinea{
			NumeroLinea:    l.ID,
			Descrizione:    l.Name,
			Quantita:       dec2(l.Quantity),
			UnitaMisura:    string(l.UnitCode),
			PrezzoUnitario: amt2(l.NetPrice),
			PrezzoTotale:   amt2(l.NetAmount),
		}
		if l.TaxRate != nil {
			line.AliquotaIVA = dec2(*l.TaxRate)
		} else {
			line.AliquotaIVA = "0.00"
		}
		if n := natura(l.TaxCategory); n != "" {
			line.Natura = n
		}
		r.Body.DatiBeniServizi.Linee = append(r.Body.DatiBeniServizi.Linee, line)
	}

	for _, ts := range doc.TaxBreakdown {
		rp := datiRiepilogo{
			AliquotaIVA:       dec2(ts.Rate),
			ImponibileImporto: amt2(ts.TaxableAmount),
			Imposta:           amt2(ts.TaxAmount),
			EsigibilitaIVA:    "I", // IVA ad esigibilità immediata
		}
		if n := natura(ts.Category); n != "" {
			rp.Natura = n
		}
		r.Body.DatiBeniServizi.Riepilogo = append(r.Body.DatiBeniServizi.Riepilogo, rp)
	}

	// Paiement : échéance + IBAN si présents.
	if doc.DueDate != nil || hasIBAN(doc) {
		det := dettaglioPagamento{ModalitaPagamento: "MP05", Importo: amt2(doc.Totals.DuePayableAmount)}
		if doc.DueDate != nil {
			det.DataScadenza = isoDate(*doc.DueDate)
		}
		if iban := firstIBAN(doc); iban != "" {
			det.IBAN = iban
		}
		r.Body.DatiPagamento = append(r.Body.DatiPagamento,
			datiPagamento{CondizioniPagamento: "TP02", Dettaglio: []dettaglioPagamento{det}})
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, fmt.Errorf("fatturapa: sérialisation: %w", err)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func party(p model.Party) anagrafica {
	var a anagrafica
	country, code := splitVAT(p.VATID)
	a.DatiAnagrafici.IdFiscaleIVA.IdPaese = country
	a.DatiAnagrafici.IdFiscaleIVA.IdCodice = code
	a.DatiAnagrafici.CodiceFiscale = p.TaxID
	a.DatiAnagrafici.Anagrafica.Denominazione = p.Name
	a.Sede.Indirizzo = p.Address.Line1
	a.Sede.CAP = p.Address.PostalCode
	a.Sede.Comune = p.Address.City
	a.Sede.Nazione = p.Address.CountryCode
	return a
}

// natura déduit le code Natura FatturaPA d'une catégorie de TVA EN 16931 à taux zéro.
func natura(cat model.TaxCategoryCode) string {
	switch cat {
	case model.TaxReverseCharge:
		return "N6" // inversione contabile (autoliquidation)
	case model.TaxExempt:
		return "N4" // esenti
	case model.TaxIntraCommunity:
		return "N3.2" // non imponibili — cessioni intracomunitarie
	case model.TaxExport:
		return "N3.1" // non imponibili — esportazioni
	case model.TaxOutsideScope:
		return "N2.2" // non soggette
	default:
		return "" // TaxStandard / TaxZeroRated : pas de Natura
	}
}

func splitVAT(vatid string) (country, code string) {
	v := strings.TrimSpace(vatid)
	if len(v) >= 2 && isAlpha(v[0]) && isAlpha(v[1]) {
		return strings.ToUpper(v[:2]), v[2:]
	}
	return "", v
}

func isAlpha(b byte) bool { return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') }

func hasIBAN(doc *model.Document) bool { return firstIBAN(doc) != "" }

func firstIBAN(doc *model.Document) string {
	if doc.PaymentInstructions == nil {
		return ""
	}
	for _, ct := range doc.PaymentInstructions.CreditTransfers {
		if strings.TrimSpace(ct.IBAN) != "" {
			return ct.IBAN
		}
	}
	return ""
}

func isoDate(d model.Date) string {
	if d.IsZero() {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, int(d.Month), d.Day)
}

func amt2(a model.Amount) string { return a.Rescale(2).String() }

func dec2(d model.Decimal) string { return d.Rescale(2).String() }
