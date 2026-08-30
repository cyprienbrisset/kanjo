// Package orderx lit le format Order-X (bon de commande hybride UN/CEFACT SCRDM — CrossIndustryOrder)
// vers le modèle pivot. Order-X partage la structure CII (ExchangedDocument,
// SupplyChainTradeTransaction) ; seules diffèrent la racine (SCRDMCCBDACIOMESSAGE), la devise
// (OrderCurrencyCode) et la quantité de ligne (RequestedQuantity).
//
// Le pivot étant centré facture, la commande y est représentée avec Kind=order et TypeCode 220.
// Les champs sans équivalent (échéancier de livraison détaillé, conditions de commande…) ne sont
// pas remontés dans cette version (§17.7, rien inventé).
package orderx

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/internal/xmlsafe"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

func init() { read.Register(read.FormatOrderX, Read) }

// Read désérialise un flux Order-X en document pivot.
func Read(data []byte, sourceName string) (*model.Document, error) {
	var x order
	if err := xmlsafe.Decode(data, &x); err != nil {
		return nil, fmt.Errorf("lecture Order-X %s: %w", sourceName, err)
	}
	return mapToPivot(&x, sourceName)
}

// --- Structures XML (appariées par nom local ; le préfixe rsm/ram est ignoré) ---

type order struct {
	Context struct {
		Guideline struct {
			ID string `xml:"ID"`
		} `xml:"GuidelineSpecifiedDocumentContextParameter"`
	} `xml:"ExchangedDocumentContext"`
	Doc struct {
		ID            string `xml:"ID"`
		TypeCode      string `xml:"TypeCode"`
		IssueDateTime dtStr  `xml:"IssueDateTime>DateTimeString"`
		Notes         []struct {
			Content string `xml:"Content"`
		} `xml:"IncludedNote"`
	} `xml:"ExchangedDocument"`
	Transaction struct {
		Lines     []orderLine `xml:"IncludedSupplyChainTradeLineItem"`
		Agreement struct {
			BuyerReference string  `xml:"BuyerReference"`
			Seller         oxParty `xml:"SellerTradeParty"`
			Buyer          oxParty `xml:"BuyerTradeParty"`
		} `xml:"ApplicableHeaderTradeAgreement"`
		Delivery struct {
			ShipTo struct {
				Name    string    `xml:"Name"`
				Address oxAddress `xml:"PostalTradeAddress"`
			} `xml:"ShipToTradeParty"`
			Requested dtStr `xml:"RequestedDeliverySupplyChainEvent>OccurrenceDateTime>DateTimeString"`
		} `xml:"ApplicableHeaderTradeDelivery"`
		Settlement struct {
			Currency string `xml:"OrderCurrencyCode"`
		} `xml:"ApplicableHeaderTradeSettlement"`
	} `xml:"SupplyChainTradeTransaction"`
}

type orderLine struct {
	Doc struct {
		LineID string `xml:"LineID"`
	} `xml:"AssociatedDocumentLineDocument"`
	Product struct {
		SellerAssignedID string `xml:"SellerAssignedID"`
		Name             string `xml:"Name"`
		Description      string `xml:"Description"`
	} `xml:"SpecifiedTradeProduct"`
	Agreement struct {
		Net struct {
			ChargeAmount string `xml:"ChargeAmount"`
		} `xml:"NetPriceProductTradePrice"`
	} `xml:"SpecifiedLineTradeAgreement"`
	Delivery struct {
		Requested qtyEl `xml:"RequestedQuantity"`
	} `xml:"SpecifiedLineTradeDelivery"`
}

type dtStr struct {
	Format string `xml:"format,attr"`
	Value  string `xml:",chardata"`
}

type qtyEl struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type oxParty struct {
	Name    string    `xml:"Name"`
	Address oxAddress `xml:"PostalTradeAddress"`
	TaxReg  []struct {
		ID     string `xml:",chardata"`
		Scheme string `xml:"schemeID,attr"`
	} `xml:"SpecifiedTaxRegistration>ID"`
}

type oxAddress struct {
	PostalCode  string `xml:"PostcodeCode"`
	Line1       string `xml:"LineOne"`
	City        string `xml:"CityName"`
	CountryCode string `xml:"CountryID"`
}

func mapToPivot(x *order, sourceName string) (*model.Document, error) {
	doc := model.NewDocument(model.KindOrder)
	doc.ID = strings.TrimSpace(x.Doc.ID)
	if tc := strings.TrimSpace(x.Doc.TypeCode); tc != "" {
		doc.TypeCode = model.TypeCode(tc)
	} else {
		doc.TypeCode = model.TypeCode("220") // commande
	}
	if v := strings.TrimSpace(x.Doc.IssueDateTime.Value); v != "" {
		if d, err := model.ParseDate(v); err == nil {
			doc.IssueDate = d
		}
	}
	doc.CurrencyCode = strings.TrimSpace(x.Transaction.Settlement.Currency)

	doc.Seller = party(x.Transaction.Agreement.Seller)
	doc.Buyer = party(x.Transaction.Agreement.Buyer)

	for _, l := range x.Transaction.Lines {
		line := model.Line{
			ID:       strings.TrimSpace(l.Doc.LineID),
			Name:     strings.TrimSpace(l.Product.Name),
			UnitCode: model.UnitCode(strings.TrimSpace(l.Delivery.Requested.UnitCode)),
		}
		if q := strings.TrimSpace(l.Delivery.Requested.Value); q != "" {
			if dq, err := model.ParseDecimal(q); err == nil {
				line.Quantity = dq
				line.QuantityPresent = true
			}
		}
		if p := strings.TrimSpace(l.Agreement.Net.ChargeAmount); p != "" {
			if a, err := model.ParseAmount(p, doc.CurrencyCode); err == nil {
				line.NetPrice = a
			}
		}
		doc.Lines = append(doc.Lines, line)
	}

	for _, n := range x.Doc.Notes {
		if c := strings.TrimSpace(n.Content); c != "" {
			doc.Notes = append(doc.Notes, model.Note{Content: c})
		}
	}

	guideline := strings.TrimSpace(x.Context.Guideline.ID)
	doc.Provenance = model.NewProvenance(sourceName, "orderx", guideline)
	doc.Provenance.SpecIdentifier = guideline // BT-24 équivalent Order-X
	return doc, nil
}

func party(p oxParty) model.Party {
	out := model.Party{
		Name: strings.TrimSpace(p.Name),
		Address: model.Address{
			Line1:       strings.TrimSpace(p.Address.Line1),
			City:        strings.TrimSpace(p.Address.City),
			PostalCode:  strings.TrimSpace(p.Address.PostalCode),
			CountryCode: strings.TrimSpace(p.Address.CountryCode),
		},
	}
	for _, r := range p.TaxReg {
		if strings.EqualFold(strings.TrimSpace(r.Scheme), "VA") {
			out.VATID = strings.TrimSpace(r.ID)
		}
	}
	return out
}
