// Package orderx sérialise un document pivot (Kind=order) vers le format Order-X (bon de commande
// hybride UN/CEFACT SCRDM — CrossIndustryOrder). C'est le symétrique du lecteur pkg/read/orderx.
//
// Order-X réutilise la structure CII avec quelques spécificités : racine SCRDMCCBDACIOMESSAGE,
// devise OrderCurrencyCode, quantité de ligne RequestedQuantity. Les champs facture sans objet pour
// une commande (TVA, totaux à payer…) ne sont pas émis (§17.7).
package orderx

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func init() { write.Register("orderx", Write) }

const (
	nsRSM = "urn:un:unece:uncefact:data:standard:SCRDMCCBDACIOMESSAGEStructure:100"
	nsRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:128"
	nsUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:128"
	// Profil Order-X par défaut (comfort).
	guidelineComfort = "urn:order-x.eu:1p0:comfort"
)

// Write produit un message Order-X (SCRDMCCBDACIOMESSAGE) pour la commande donnée.
func Write(doc *model.Document, _ write.Options) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("orderx: document nil")
	}

	typeCode := string(doc.TypeCode)
	if typeCode == "" {
		typeCode = "220" // commande
	}

	r := root{
		XMLNSrsm: nsRSM, XMLNSram: nsRAM, XMLNSudt: nsUDT,
	}
	r.Context.Guideline.ID = guidelineComfort
	r.Doc.ID = doc.ID
	r.Doc.TypeCode = typeCode
	if !doc.IssueDate.IsZero() {
		r.Doc.IssueDateTime.DateTime.Format = "102"
		r.Doc.IssueDateTime.DateTime.Value = fmt.Sprintf("%04d%02d%02d", doc.IssueDate.Year, int(doc.IssueDate.Month), doc.IssueDate.Day)
	}

	for _, l := range doc.Lines {
		line := lineItem{}
		line.Doc.LineID = l.ID
		line.Product.Name = l.Name
		if !l.NetPrice.IsZero() {
			line.Agreement.Net.ChargeAmount = l.NetPrice.String()
		}
		line.Delivery.Requested.UnitCode = string(l.UnitCode)
		line.Delivery.Requested.Value = l.Quantity.String()
		r.Transaction.Lines = append(r.Transaction.Lines, line)
	}

	r.Transaction.Agreement.Seller = party(doc.Seller)
	r.Transaction.Agreement.Buyer = party(doc.Buyer)
	r.Transaction.Settlement.Currency = doc.CurrencyCode

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, fmt.Errorf("orderx: sérialisation: %w", err)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func party(p model.Party) *tradeParty {
	if p.Name == "" && p.VATID == "" {
		return nil
	}
	tp := &tradeParty{Name: p.Name}
	tp.Address.PostalCode = p.Address.PostalCode
	tp.Address.Line1 = p.Address.Line1
	tp.Address.City = p.Address.City
	tp.Address.CountryCode = p.Address.CountryCode
	if p.VATID != "" {
		tp.TaxReg = &taxRegistration{}
		tp.TaxReg.ID.Scheme = "VA"
		tp.TaxReg.ID.Value = p.VATID
	}
	return tp
}
