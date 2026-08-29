// Package cii lit le format CII UN/CEFACT CrossIndustryInvoice D16B vers le pivot.
// Tout le parsing passe par internal/xmlsafe (jamais encoding/xml directement, §17.1).
package cii

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/internal/xmlsafe"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

func init() { read.Register(read.FormatCII, Read) }

// Read désérialise un flux CII en document pivot.
func Read(data []byte, sourceName string) (*model.Document, error) {
	var x ciiInvoice
	if err := xmlsafe.Decode(data, &x); err != nil {
		return nil, fmt.Errorf("lecture CII %s: %w", sourceName, err)
	}
	return mapToPivot(&x, sourceName)
}

// --- Structures XML (tags par nom local ; encoding/xml ignore le préfixe) ---

type ciiInvoice struct {
	Context struct {
		Guideline struct {
			ID string `xml:"ID"`
		} `xml:"GuidelineSpecifiedDocumentContextParameter"`
	} `xml:"ExchangedDocumentContext"`
	Doc struct {
		ID            string      `xml:"ID"`
		TypeCode      string      `xml:"TypeCode"`
		IssueDateTime dateTimeStr `xml:"IssueDateTime>DateTimeString"`
		Notes         []struct {
			Content     string `xml:"Content"`
			SubjectCode string `xml:"SubjectCode"`
		} `xml:"IncludedNote"`
	} `xml:"ExchangedDocument"`
	Transaction struct {
		Lines      []ciiLine     `xml:"IncludedSupplyChainTradeLineItem"`
		Agreement  ciiAgreement  `xml:"ApplicableHeaderTradeAgreement"`
		Delivery   ciiDelivery   `xml:"ApplicableHeaderTradeDelivery"`
		Settlement ciiSettlement `xml:"ApplicableHeaderTradeSettlement"`
	} `xml:"SupplyChainTradeTransaction"`
}

type dateTimeStr struct {
	Format string `xml:"format,attr"`
	Value  string `xml:",chardata"`
}

func (d dateTimeStr) parse() (*model.Date, error) {
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

type ciiLine struct {
	Doc struct {
		LineID string `xml:"LineID"`
		Note   struct {
			Content string `xml:"Content"`
		} `xml:"IncludedNote"`
	} `xml:"AssociatedDocumentLineDocument"`
	Product struct {
		GlobalID struct {
			Scheme string `xml:"schemeID,attr"`
			Value  string `xml:",chardata"`
		} `xml:"GlobalID"`
		SellerAssignedID string `xml:"SellerAssignedID"`
		BuyerAssignedID  string `xml:"BuyerAssignedID"`
		Name             string `xml:"Name"`
		Description      string `xml:"Description"`
	} `xml:"SpecifiedTradeProduct"`
	Agreement struct {
		Gross struct {
			ChargeAmount string `xml:"ChargeAmount"`
			Allowance    struct {
				Indicator string `xml:"ChargeIndicator>Indicator"`
				Amount    string `xml:"ActualAmount"`
			} `xml:"AppliedTradeAllowanceCharge"`
		} `xml:"GrossPriceProductTradePrice"`
		Net struct {
			ChargeAmount  string     `xml:"ChargeAmount"`
			BasisQuantity quantityEl `xml:"BasisQuantity"`
		} `xml:"NetPriceProductTradePrice"`
	} `xml:"SpecifiedLineTradeAgreement"`
	Delivery struct {
		BilledQuantity quantityEl `xml:"BilledQuantity"`
	} `xml:"SpecifiedLineTradeDelivery"`
	Settlement struct {
		Tax struct {
			CategoryCode string `xml:"CategoryCode"`
			Rate         string `xml:"RateApplicablePercent"`
		} `xml:"ApplicableTradeTax"`
		Allowances []ciiAllowance `xml:"SpecifiedTradeAllowanceCharge"` // BG-27/28
		Sum        struct {
			LineTotalAmount string `xml:"LineTotalAmount"`
		} `xml:"SpecifiedTradeSettlementLineMonetarySummation"`
	} `xml:"SpecifiedLineTradeSettlement"`
}

type quantityEl struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type currencyValue struct {
	Currency string `xml:"currencyID,attr"`
	Value    string `xml:",chardata"`
}

type ciiAgreement struct {
	BuyerReference string   `xml:"BuyerReference"`
	Seller         ciiParty `xml:"SellerTradeParty"`
	Buyer          ciiParty `xml:"BuyerTradeParty"`
	Order          struct {
		ID string `xml:"IssuerAssignedID"`
	} `xml:"BuyerOrderReferencedDocument"`
}

type ciiParty struct {
	DirectID string `xml:"ID"`
	GlobalID struct {
		Value string `xml:",chardata"`
	} `xml:"GlobalID"`
	Name     string `xml:"Name"`
	LegalOrg struct {
		ID struct {
			Scheme string `xml:"schemeID,attr"`
			Value  string `xml:",chardata"`
		} `xml:"ID"`
		TradingName string `xml:"TradingBusinessName"`
	} `xml:"SpecifiedLegalOrganization"`
	Contact struct {
		Name  string `xml:"PersonName"`
		Phone string `xml:"TelephoneUniversalCommunication>CompleteNumber"`
		Email string `xml:"EmailURIUniversalCommunication>URIID"`
	} `xml:"DefinedTradeContact"`
	Address ciiAddress `xml:"PostalTradeAddress"`
	URI     struct {
		Scheme string `xml:"schemeID,attr"`
		Value  string `xml:",chardata"`
	} `xml:"URIUniversalCommunication>URIID"`
	TaxReg []struct {
		ID     string `xml:",chardata"`
		Scheme string `xml:"schemeID,attr"`
	} `xml:"SpecifiedTaxRegistration>ID"`
}

type ciiAllowance struct {
	Indicator    string `xml:"ChargeIndicator>Indicator"`
	Percent      string `xml:"CalculationPercent"`
	BasisAmount  string `xml:"BasisAmount"`
	ActualAmount string `xml:"ActualAmount"`
	ReasonCode   string `xml:"ReasonCode"`
	Reason       string `xml:"Reason"`
	Category     struct {
		CategoryCode string `xml:"CategoryCode"`
		Rate         string `xml:"RateApplicablePercent"`
	} `xml:"CategoryTradeTax"`
}

type ciiHeaderTax struct {
	CalculatedAmount    string `xml:"CalculatedAmount"`
	BasisAmount         string `xml:"BasisAmount"`
	CategoryCode        string `xml:"CategoryCode"`
	Rate                string `xml:"RateApplicablePercent"`
	ExemptionReason     string `xml:"ExemptionReason"`
	ExemptionReasonCode string `xml:"ExemptionReasonCode"`
}

type ciiAddress struct {
	PostalCode  string `xml:"PostcodeCode"`
	Line1       string `xml:"LineOne"`
	Line2       string `xml:"LineTwo"`
	City        string `xml:"CityName"`
	CountryCode string `xml:"CountryID"`
}

type ciiDelivery struct {
	ShipTo struct {
		Name    string     `xml:"Name"`
		Address ciiAddress `xml:"PostalTradeAddress"`
	} `xml:"ShipToTradeParty"`
	Event struct {
		Date dateTimeStr `xml:"OccurrenceDateTime>DateTimeString"`
	} `xml:"ActualDeliverySupplyChainEvent"`
}

type ciiSettlement struct {
	PaymentReference string `xml:"PaymentReference"`
	Currency         string `xml:"InvoiceCurrencyCode"`
	PaymentMeans     []struct {
		TypeCode string `xml:"TypeCode"`
		IBAN     string `xml:"PayeePartyCreditorFinancialAccount>IBANID"`
	} `xml:"SpecifiedTradeSettlementPaymentMeans"`
	Tax          []ciiHeaderTax `xml:"ApplicableTradeTax"`
	Allowances   []ciiAllowance `xml:"SpecifiedTradeAllowanceCharge"`
	PaymentTerms struct {
		Description string      `xml:"Description"`
		DueDate     dateTimeStr `xml:"DueDateDateTime>DateTimeString"`
	} `xml:"SpecifiedTradePaymentTerms"`
	Sum struct {
		LineTotal      string          `xml:"LineTotalAmount"`
		ChargeTotal    string          `xml:"ChargeTotalAmount"`
		AllowanceTotal string          `xml:"AllowanceTotalAmount"`
		TaxBasisTotal  string          `xml:"TaxBasisTotalAmount"`
		TaxTotals      []currencyValue `xml:"TaxTotalAmount"` // peut apparaître 2× (devise doc + devise TVA, BT-111)
		GrandTotal     string          `xml:"GrandTotalAmount"`
		Prepaid        string          `xml:"TotalPrepaidAmount"`
		DuePayable     string          `xml:"DuePayableAmount"`
	} `xml:"SpecifiedTradeSettlementHeaderMonetarySummation"`
}
