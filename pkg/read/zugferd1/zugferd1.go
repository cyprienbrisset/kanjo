// Package zugferd1 lit le format ZUGFeRD 1.0 (rsm:CrossIndustryDocument, syntaxe CII D14B),
// format hérité franco-allemand antérieur à Factur-X. Il se distingue de la D16B (CII actuel)
// par ses conteneurs (HeaderExchangedDocument, SpecifiedSupplyChainTradeTransaction,
// Applicable/Specified-SupplyChainTrade-*) et sa ventilation de TVA au niveau en-tête.
//
// Tout le parsing passe par internal/xmlsafe (§17.1).
package zugferd1

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/internal/xmlsafe"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

func init() { read.Register(read.FormatZUGFeRD1, Read) }

// Read désérialise un flux ZUGFeRD 1.0 (CII D14B) en document pivot.
func Read(data []byte, sourceName string) (*model.Document, error) {
	var x ciDoc
	if err := xmlsafe.Decode(data, &x); err != nil {
		return nil, fmt.Errorf("lecture ZUGFeRD 1.0 %s: %w", sourceName, err)
	}
	return mapToPivot(&x, sourceName)
}

// --- Structures XML D14B (tags par nom local) ---

type dateTimeStr struct {
	Value string `xml:",chardata"`
}

func (d dateTimeStr) date() (*model.Date, error) {
	v := strings.TrimSpace(d.Value)
	if v == "" {
		return nil, nil
	}
	dt, err := model.ParseDate(v)
	if err != nil {
		return nil, err
	}
	return &dt, nil
}

type quantityEl struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type ciDoc struct {
	Context struct {
		Guideline struct {
			ID string `xml:"ID"`
		} `xml:"GuidelineSpecifiedDocumentContextParameter"`
	} `xml:"SpecifiedExchangedDocumentContext"`
	Header struct {
		ID       string      `xml:"ID"`
		TypeCode string      `xml:"TypeCode"`
		Issue    dateTimeStr `xml:"IssueDateTime>DateTimeString"`
		Notes    []struct {
			Content string `xml:"Content"`
		} `xml:"IncludedNote"`
	} `xml:"HeaderExchangedDocument"`
	Tx struct {
		Agreement struct {
			BuyerRef string `xml:"BuyerReference"`
			Seller   party  `xml:"SellerTradeParty"`
			Buyer    party  `xml:"BuyerTradeParty"`
		} `xml:"ApplicableSupplyChainTradeAgreement"`
		Settlement struct {
			Currency string      `xml:"InvoiceCurrencyCode"`
			Tax      []headerTax `xml:"ApplicableTradeTax"`
			Sum      struct {
				LineTotal     string `xml:"LineTotalAmount"`
				ChargeTotal   string `xml:"ChargeTotalAmount"`
				AllowTotal    string `xml:"AllowanceTotalAmount"`
				TaxBasisTotal string `xml:"TaxBasisTotalAmount"`
				TaxTotal      string `xml:"TaxTotalAmount"`
				GrandTotal    string `xml:"GrandTotalAmount"`
				Prepaid       string `xml:"TotalPrepaidAmount"`
				DuePayable    string `xml:"DuePayableAmount"`
			} `xml:"SpecifiedTradeSettlementMonetarySummation"`
		} `xml:"ApplicableSupplyChainTradeSettlement"`
		Lines []lineItem `xml:"IncludedSupplyChainTradeLineItem"`
	} `xml:"SpecifiedSupplyChainTradeTransaction"`
}

type headerTax struct {
	CalculatedAmount string `xml:"CalculatedAmount"`
	BasisAmount      string `xml:"BasisAmount"`
	CategoryCode     string `xml:"CategoryCode"`
	Percent          string `xml:"ApplicablePercent"`
}

type party struct {
	Name     string `xml:"Name"`
	LegalOrg struct {
		ID string `xml:"ID"`
	} `xml:"SpecifiedLegalOrganization"`
	Address struct {
		PostalCode  string `xml:"PostcodeCode"`
		Line1       string `xml:"LineOne"`
		City        string `xml:"CityName"`
		CountryCode string `xml:"CountryID"`
	} `xml:"PostalTradeAddress"`
	TaxReg []struct {
		ID     string `xml:",chardata"`
		Scheme string `xml:"schemeID,attr"`
	} `xml:"SpecifiedTaxRegistration>ID"`
}

type lineItem struct {
	Doc struct {
		LineID string `xml:"LineID"`
	} `xml:"AssociatedDocumentLineDocument"`
	Product struct {
		Name string `xml:"Name"`
	} `xml:"SpecifiedTradeProduct"`
	Agreement struct {
		Net struct {
			ChargeAmount string `xml:"ChargeAmount"`
		} `xml:"NetPriceProductTradePrice"`
	} `xml:"SpecifiedSupplyChainTradeAgreement"`
	Delivery struct {
		Qty quantityEl `xml:"BilledQuantity"`
	} `xml:"SpecifiedSupplyChainTradeDelivery"`
	Settlement struct {
		Tax struct {
			CategoryCode string `xml:"CategoryCode"`
			Percent      string `xml:"ApplicablePercent"`
		} `xml:"ApplicableTradeTax"`
		Sum struct {
			LineTotal string `xml:"LineTotalAmount"`
		} `xml:"SpecifiedTradeSettlementMonetarySummation"`
	} `xml:"SpecifiedSupplyChainTradeSettlement"`
}
