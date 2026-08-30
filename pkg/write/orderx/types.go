package orderx

import "encoding/xml"

// Structures de sérialisation Order-X (SCRDMCCBDACIOMESSAGE). Les préfixes rsm/ram/udt sont émis
// littéralement ; le lecteur apparie par nom local, donc le round-trip est garanti.

type root struct {
	XMLName  xml.Name `xml:"rsm:SCRDMCCBDACIOMESSAGE"`
	XMLNSrsm string   `xml:"xmlns:rsm,attr"`
	XMLNSram string   `xml:"xmlns:ram,attr"`
	XMLNSudt string   `xml:"xmlns:udt,attr"`

	Context struct {
		Guideline struct {
			ID string `xml:"ram:ID"`
		} `xml:"ram:GuidelineSpecifiedDocumentContextParameter"`
	} `xml:"rsm:ExchangedDocumentContext"`

	Doc struct {
		ID            string `xml:"ram:ID"`
		TypeCode      string `xml:"ram:TypeCode"`
		IssueDateTime struct {
			DateTime dateTimeString `xml:"udt:DateTimeString"`
		} `xml:"ram:IssueDateTime"`
	} `xml:"rsm:ExchangedDocument"`

	Transaction struct {
		Lines     []lineItem `xml:"ram:IncludedSupplyChainTradeLineItem"`
		Agreement struct {
			Seller *tradeParty `xml:"ram:SellerTradeParty"`
			Buyer  *tradeParty `xml:"ram:BuyerTradeParty"`
		} `xml:"ram:ApplicableHeaderTradeAgreement"`
		Delivery   struct{} `xml:"ram:ApplicableHeaderTradeDelivery"`
		Settlement struct {
			Currency string `xml:"ram:OrderCurrencyCode"`
		} `xml:"ram:ApplicableHeaderTradeSettlement"`
	} `xml:"rsm:SupplyChainTradeTransaction"`
}

type dateTimeString struct {
	Format string `xml:"format,attr"`
	Value  string `xml:",chardata"`
}

type lineItem struct {
	Doc struct {
		LineID string `xml:"ram:LineID"`
	} `xml:"ram:AssociatedDocumentLineDocument"`
	Product struct {
		Name string `xml:"ram:Name"`
	} `xml:"ram:SpecifiedTradeProduct"`
	Agreement struct {
		Net struct {
			ChargeAmount string `xml:"ram:ChargeAmount,omitempty"`
		} `xml:"ram:NetPriceProductTradePrice"`
	} `xml:"ram:SpecifiedLineTradeAgreement"`
	Delivery struct {
		Requested quantity `xml:"ram:RequestedQuantity"`
	} `xml:"ram:SpecifiedLineTradeDelivery"`
}

type quantity struct {
	UnitCode string `xml:"unitCode,attr,omitempty"`
	Value    string `xml:",chardata"`
}

type tradeParty struct {
	Name    string `xml:"ram:Name"`
	Address struct {
		PostalCode  string `xml:"ram:PostcodeCode,omitempty"`
		Line1       string `xml:"ram:LineOne,omitempty"`
		City        string `xml:"ram:CityName,omitempty"`
		CountryCode string `xml:"ram:CountryID,omitempty"`
	} `xml:"ram:PostalTradeAddress"`
	TaxReg *taxRegistration `xml:"ram:SpecifiedTaxRegistration"`
}

type taxRegistration struct {
	ID struct {
		Scheme string `xml:"schemeID,attr"`
		Value  string `xml:",chardata"`
	} `xml:"ram:ID"`
}
