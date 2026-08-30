// Package ubl lit le format UBL 2.1 (Invoice et CreditNote) vers le pivot.
// Tout le parsing passe par internal/xmlsafe (jamais encoding/xml directement, §17.1).
package ubl

import (
	"encoding/xml"
	"fmt"

	"github.com/cyprienbrisset/kanjo/internal/xmlsafe"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

func init() {
	read.Register(read.FormatUBLInvoice, Read)
	read.Register(read.FormatUBLCreditNote, Read)
}

// Read désérialise un flux UBL 2.1 en document pivot.
func Read(data []byte, sourceName string) (*model.Document, error) {
	var x ublDocument
	if err := xmlsafe.Decode(data, &x); err != nil {
		return nil, fmt.Errorf("lecture UBL %s: %w", sourceName, err)
	}
	return mapToPivot(&x, sourceName)
}

// --- Structures XML (tags par nom local ; encoding/xml ignore le préfixe) ---

// ublDocument couvre à la fois Invoice et CreditNote : les deux syntaxes partagent la
// quasi-totalité de leurs éléments (par nom local). Les particularités (InvoiceTypeCode vs
// CreditNoteTypeCode, InvoiceLine vs CreditNoteLine) sont captées par des champs distincts.
type ublDocument struct {
	XMLName         xml.Name
	CustomizationID string   `xml:"CustomizationID"`
	ID              string   `xml:"ID"`
	IssueDate       string   `xml:"IssueDate"`
	DueDate         string   `xml:"DueDate"`
	InvoiceTypeCode string   `xml:"InvoiceTypeCode"`
	CreditNoteType  string   `xml:"CreditNoteTypeCode"`
	Notes           []string `xml:"Note"`
	CurrencyCode    string   `xml:"DocumentCurrencyCode"`
	TaxPointCode    string   `xml:"InvoicePeriod>DescriptionCode"` // BT-8 code de date d'exigibilité TVA
	BuyerReference  string   `xml:"BuyerReference"`
	OrderReference  struct {
		ID string `xml:"ID"`
	} `xml:"OrderReference"`
	SupplierParty struct {
		Party ublParty `xml:"Party"`
	} `xml:"AccountingSupplierParty"`
	CustomerParty struct {
		Party ublParty `xml:"Party"`
	} `xml:"AccountingCustomerParty"`
	Delivery     ublDelivery     `xml:"Delivery"`
	PaymentMeans ublPaymentMeans `xml:"PaymentMeans"`
	PaymentTerms struct {
		Note string `xml:"Note"`
	} `xml:"PaymentTerms"`
	AllowanceCharge    []ublAllowanceCharge  `xml:"AllowanceCharge"`
	TaxTotal           []ublTaxTotal         `xml:"TaxTotal"`
	LegalMonetaryTotal ublLegalMonetaryTotal `xml:"LegalMonetaryTotal"`
	InvoiceLines       []ublLine             `xml:"InvoiceLine"`
	CreditNoteLines    []ublLine             `xml:"CreditNoteLine"`
}

type ublParty struct {
	Endpoint struct {
		Value  string `xml:",chardata"`
		Scheme string `xml:"schemeID,attr"`
	} `xml:"EndpointID"`
	Identification struct {
		ID string `xml:"ID"`
	} `xml:"PartyIdentification"`
	PartyName struct {
		Name string `xml:"Name"`
	} `xml:"PartyName"`
	PostalAddress  ublAddress `xml:"PostalAddress"`
	PartyTaxScheme struct {
		CompanyID string `xml:"CompanyID"`
		Scheme    string `xml:"TaxScheme>ID"`
	} `xml:"PartyTaxScheme"`
	PartyLegalEntity struct {
		RegistrationName string `xml:"RegistrationName"`
		CompanyID        struct {
			Value  string `xml:",chardata"`
			Scheme string `xml:"schemeID,attr"`
		} `xml:"CompanyID"`
	} `xml:"PartyLegalEntity"`
	Contact struct {
		Name  string `xml:"Name"`
		Phone string `xml:"Telephone"`
		Email string `xml:"ElectronicMail"`
	} `xml:"Contact"`
}

type ublAddress struct {
	StreetName     string `xml:"StreetName"`
	AdditionalName string `xml:"AdditionalStreetName"`
	CityName       string `xml:"CityName"`
	PostalZone     string `xml:"PostalZone"`
	CountryCode    string `xml:"Country>IdentificationCode"`
}

type ublDelivery struct {
	ActualDeliveryDate string `xml:"ActualDeliveryDate"`
	DeliveryLocation   struct {
		Address ublAddress `xml:"Address"`
	} `xml:"DeliveryLocation"`
	DeliveryParty struct {
		Name string `xml:"PartyName>Name"`
	} `xml:"DeliveryParty"`
}

type ublPaymentMeans struct {
	Code    string `xml:"PaymentMeansCode"`
	PayID   string `xml:"PaymentID"`
	Account struct {
		ID string `xml:"ID"`
	} `xml:"PayeeFinancialAccount"`
}

type ublAllowanceCharge struct {
	ChargeIndicator string `xml:"ChargeIndicator"` // true = charge, false = remise
	ReasonCode      string `xml:"AllowanceChargeReasonCode"`
	Reason          string `xml:"AllowanceChargeReason"`
	Percent         string `xml:"MultiplierFactorNumeric"`
	Amount          string `xml:"Amount"`
	BaseAmount      string `xml:"BaseAmount"`
	Category        struct {
		ID      string `xml:"ID"`
		Percent string `xml:"Percent"`
	} `xml:"TaxCategory"`
}

type ublTaxTotal struct {
	TaxAmount struct {
		Value    string `xml:",chardata"`
		Currency string `xml:"currencyID,attr"`
	} `xml:"TaxAmount"`
	TaxSubtotal []ublTaxSubtotal `xml:"TaxSubtotal"`
}

type ublTaxSubtotal struct {
	TaxableAmount string `xml:"TaxableAmount"`
	TaxAmount     string `xml:"TaxAmount"`
	Category      struct {
		ID                  string `xml:"ID"`
		Percent             string `xml:"Percent"`
		ExemptionReason     string `xml:"TaxExemptionReason"`
		ExemptionReasonCode string `xml:"TaxExemptionReasonCode"`
	} `xml:"TaxCategory"`
}

type ublLegalMonetaryTotal struct {
	LineExtensionAmount string `xml:"LineExtensionAmount"`
	TaxExclusiveAmount  string `xml:"TaxExclusiveAmount"`
	TaxInclusiveAmount  string `xml:"TaxInclusiveAmount"`
	AllowanceTotal      string `xml:"AllowanceTotalAmount"`
	ChargeTotal         string `xml:"ChargeTotalAmount"`
	PrepaidAmount       string `xml:"PrepaidAmount"`
	PayableAmount       string `xml:"PayableAmount"`
}

type ublLine struct {
	ID       string `xml:"ID"`
	Note     string `xml:"Note"`
	Quantity struct {
		Value    string `xml:",chardata"`
		UnitCode string `xml:"unitCode,attr"`
	} `xml:"InvoicedQuantity"`
	CreditedQuantity struct {
		Value    string `xml:",chardata"`
		UnitCode string `xml:"unitCode,attr"`
	} `xml:"CreditedQuantity"`
	LineExtensionAmount string               `xml:"LineExtensionAmount"`
	AllowanceCharge     []ublAllowanceCharge `xml:"AllowanceCharge"` // BG-27/28
	Item                struct {
		Name           string `xml:"Name"`
		Description    string `xml:"Description"`
		SellersItemID  string `xml:"SellersItemIdentification>ID"`
		BuyersItemID   string `xml:"BuyersItemIdentification>ID"`
		StandardItemID struct {
			Value  string `xml:",chardata"`
			Scheme string `xml:"schemeID,attr"`
		} `xml:"StandardItemIdentification>ID"`
		Classification struct {
			ListID string `xml:"listID,attr"`
			Value  string `xml:",chardata"`
		} `xml:"CommodityClassification>ItemClassificationCode"` // BT-158
		ClassifiedTaxCategory struct {
			ID      string `xml:"ID"`
			Percent string `xml:"Percent"`
		} `xml:"ClassifiedTaxCategory"`
	} `xml:"Item"`
	Price struct {
		PriceAmount  string `xml:"PriceAmount"`
		BaseQuantity struct {
			Value    string `xml:",chardata"`
			UnitCode string `xml:"unitCode,attr"`
		} `xml:"BaseQuantity"`
	} `xml:"Price"`
}
