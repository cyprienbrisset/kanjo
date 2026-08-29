package zugferd1_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/read/zugferd1"
)

// Extrait minimal représentatif de ZUGFeRD 1.0 (CII D14B).
const sample = `<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryDocument xmlns:rsm="urn:ferd:CrossIndustryDocument:invoice:1p0"
 xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:12"
 xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:15">
 <rsm:SpecifiedExchangedDocumentContext>
  <ram:GuidelineSpecifiedDocumentContextParameter><ram:ID>urn:ferd:CrossIndustryDocument:invoice:1p0:basic</ram:ID></ram:GuidelineSpecifiedDocumentContextParameter>
 </rsm:SpecifiedExchangedDocumentContext>
 <rsm:HeaderExchangedDocument>
  <ram:ID>RE-2014-03</ram:ID>
  <ram:TypeCode>380</ram:TypeCode>
  <ram:IssueDateTime><udt:DateTimeString format="102">20140711</udt:DateTimeString></ram:IssueDateTime>
 </rsm:HeaderExchangedDocument>
 <rsm:SpecifiedSupplyChainTradeTransaction>
  <ram:ApplicableSupplyChainTradeAgreement>
   <ram:SellerTradeParty>
    <ram:Name>Lieferant GmbH</ram:Name>
    <ram:PostalTradeAddress><ram:PostcodeCode>80333</ram:PostcodeCode><ram:CityName>München</ram:CityName><ram:CountryID>DE</ram:CountryID></ram:PostalTradeAddress>
    <ram:SpecifiedTaxRegistration><ram:ID schemeID="VA">DE123456789</ram:ID></ram:SpecifiedTaxRegistration>
   </ram:SellerTradeParty>
   <ram:BuyerTradeParty><ram:Name>Kunden AG</ram:Name>
    <ram:PostalTradeAddress><ram:CountryID>DE</ram:CountryID></ram:PostalTradeAddress></ram:BuyerTradeParty>
  </ram:ApplicableSupplyChainTradeAgreement>
  <ram:IncludedSupplyChainTradeLineItem>
   <ram:AssociatedDocumentLineDocument><ram:LineID>1</ram:LineID></ram:AssociatedDocumentLineDocument>
   <ram:SpecifiedTradeProduct><ram:Name>Beratung</ram:Name></ram:SpecifiedTradeProduct>
   <ram:SpecifiedSupplyChainTradeDelivery><ram:BilledQuantity unitCode="C62">1.0</ram:BilledQuantity></ram:SpecifiedSupplyChainTradeDelivery>
   <ram:SpecifiedSupplyChainTradeSettlement>
    <ram:SpecifiedTradeSettlementMonetarySummation><ram:LineTotalAmount currencyID="EUR">845.00</ram:LineTotalAmount></ram:SpecifiedTradeSettlementMonetarySummation>
   </ram:SpecifiedSupplyChainTradeSettlement>
  </ram:IncludedSupplyChainTradeLineItem>
  <ram:ApplicableSupplyChainTradeSettlement>
   <ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
   <ram:ApplicableTradeTax>
    <ram:CalculatedAmount currencyID="EUR">160.55</ram:CalculatedAmount>
    <ram:TypeCode>VAT</ram:TypeCode>
    <ram:BasisAmount currencyID="EUR">845.00</ram:BasisAmount>
    <ram:CategoryCode>S</ram:CategoryCode>
    <ram:ApplicablePercent>19.0</ram:ApplicablePercent>
   </ram:ApplicableTradeTax>
   <ram:SpecifiedTradeSettlementMonetarySummation>
    <ram:LineTotalAmount currencyID="EUR">845.00</ram:LineTotalAmount>
    <ram:TaxBasisTotalAmount currencyID="EUR">845.00</ram:TaxBasisTotalAmount>
    <ram:TaxTotalAmount currencyID="EUR">160.55</ram:TaxTotalAmount>
    <ram:GrandTotalAmount currencyID="EUR">1005.55</ram:GrandTotalAmount>
    <ram:DuePayableAmount currencyID="EUR">1005.55</ram:DuePayableAmount>
   </ram:SpecifiedTradeSettlementMonetarySummation>
  </ram:ApplicableSupplyChainTradeSettlement>
 </rsm:SpecifiedSupplyChainTradeTransaction>
</rsm:CrossIndustryDocument>`

func TestDetectedAsZUGFeRD1(t *testing.T) {
	if f := read.Detect([]byte(sample)); f != read.FormatZUGFeRD1 {
		t.Errorf("format détecté = %s, veut zugferd1", f)
	}
}

func TestReadZUGFeRD1(t *testing.T) {
	doc, err := zugferd1.Read([]byte(sample), "z1.xml")
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID != "RE-2014-03" {
		t.Errorf("ID = %q", doc.ID)
	}
	if doc.CurrencyCode != "EUR" || doc.IssueDate.ISO() != "2014-07-11" {
		t.Errorf("devise/date : %s / %s", doc.CurrencyCode, doc.IssueDate.ISO())
	}
	if doc.Seller.Name != "Lieferant GmbH" || doc.Seller.VATID != "DE123456789" {
		t.Errorf("vendeur : %+v", doc.Seller)
	}
	if len(doc.Lines) != 1 || doc.Lines[0].Name != "Beratung" {
		t.Errorf("lignes : %+v", doc.Lines)
	}
	if len(doc.TaxBreakdown) != 1 || doc.TaxBreakdown[0].Category != "S" {
		t.Errorf("TVA : %+v", doc.TaxBreakdown)
	}
	if doc.Totals.TaxInclusiveAmount.String() != "1005.55" {
		t.Errorf("TTC = %s", doc.Totals.TaxInclusiveAmount)
	}
	if doc.Provenance == nil || doc.Provenance.SourceFormat != "zugferd1" {
		t.Errorf("provenance : %+v", doc.Provenance)
	}
}
