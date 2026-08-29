// Package cii écrit un document pivot au format CII UN/CEFACT CrossIndustryInvoice D16B.
// L'écriture utilise un arbre XML à préfixes explicites (rsm:/ram:/udt:).
package cii

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func init() { write.Register("cii", Write) }

// Espaces de noms CII D16B.
const (
	nsRSM = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	nsRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	nsUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
	nsQDT = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
)

// profileURN renvoie l'URN d'identification de la spécification pour un profil donné.
func profileURN(p write.Profile) string {
	switch p {
	case write.ProfileMinimum:
		return "urn:factur-x.eu:1p0:minimum"
	case write.ProfileBasicWL:
		return "urn:factur-x.eu:1p0:basicwl"
	case write.ProfileBasic:
		return "urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:basic"
	case write.ProfileExtended:
		return "urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended"
	default:
		return "urn:cen.eu:en16931:2017"
	}
}

// Write sérialise le document pivot en CII.
func Write(doc *model.Document, opts write.Options) ([]byte, error) {
	if opts.Profile == "" {
		opts.Profile = write.ProfileEN16931
	}
	root := el("rsm:CrossIndustryInvoice")
	root.attrs = []attr{
		{"xmlns:rsm", nsRSM},
		{"xmlns:ram", nsRAM},
		{"xmlns:udt", nsUDT},
		{"xmlns:qdt", nsQDT},
	}
	root.with(
		buildContext(opts),
		buildExchangedDocument(doc),
		buildTransaction(doc),
	)
	return []byte(render(root)), nil
}

func buildContext(opts write.Options) *node {
	spec := profileURN(opts.Profile)
	if opts.CustomizationID != "" {
		spec = opts.CustomizationID
	}
	return el("rsm:ExchangedDocumentContext",
		el("ram:GuidelineSpecifiedDocumentContextParameter",
			leaf("ram:ID", spec),
		),
	)
}

func buildExchangedDocument(doc *model.Document) *node {
	n := el("rsm:ExchangedDocument",
		leaf("ram:ID", doc.ID),
		leaf("ram:TypeCode", string(doc.TypeCode)),
		el("ram:IssueDateTime",
			leafA("udt:DateTimeString", doc.IssueDate.Compact(), attr{"format", "102"}),
		),
	)
	for _, note := range doc.Notes {
		nn := el("ram:IncludedNote", leaf("ram:Content", note.Content))
		nn.with(leaf("ram:SubjectCode", note.SubjectCode))
		n.with(nn)
	}
	return n
}

func buildTransaction(doc *model.Document) *node {
	t := el("rsm:SupplyChainTradeTransaction")
	for i := range doc.Lines {
		t.with(buildLine(doc, &doc.Lines[i]))
	}
	t.with(
		buildAgreement(doc),
		buildDelivery(doc),
		buildSettlement(doc),
	)
	return t
}

func buildLine(doc *model.Document, l *model.Line) *node {
	assoc := el("ram:AssociatedDocumentLineDocument", leaf("ram:LineID", l.ID))
	if l.Note != "" {
		assoc.with(el("ram:IncludedNote", leaf("ram:Content", l.Note)))
	}

	product := el("ram:SpecifiedTradeProduct")
	if l.StandardID != "" {
		product.with(leafA("ram:GlobalID", l.StandardID, attr{"schemeID", l.StandardScheme}))
	}
	product.with(
		leaf("ram:SellerAssignedID", l.SellerAssignedID),
		leaf("ram:BuyerAssignedID", l.BuyerAssignedID),
		leaf("ram:Name", l.Name),
		leaf("ram:Description", l.Description),
	)

	agreement := el("ram:SpecifiedLineTradeAgreement")
	if l.GrossPrice != nil {
		gp := el("ram:GrossPriceProductTradePrice", leaf("ram:ChargeAmount", l.GrossPrice.String()))
		if l.PriceDiscount != nil {
			gp.with(el("ram:AppliedTradeAllowanceCharge",
				el("ram:ChargeIndicator", leaf("udt:Indicator", "false")),
				leaf("ram:ActualAmount", l.PriceDiscount.String()),
			))
		}
		agreement.with(gp)
	}
	np := el("ram:NetPriceProductTradePrice", leaf("ram:ChargeAmount", l.NetPrice.String()))
	if l.PriceBaseQty != nil {
		np.with(leafA("ram:BasisQuantity", l.PriceBaseQty.String(), attr{"unitCode", string(l.UnitCode)}))
	}
	agreement.with(np)

	delivery := el("ram:SpecifiedLineTradeDelivery",
		leafA("ram:BilledQuantity", l.Quantity.String(), attr{"unitCode", string(l.UnitCode)}),
	)

	tax := el("ram:ApplicableTradeTax",
		leaf("ram:TypeCode", "VAT"),
		leaf("ram:CategoryCode", string(l.TaxCategory)),
	)
	if l.TaxRate != nil {
		tax.with(leaf("ram:RateApplicablePercent", l.TaxRate.String()))
	}
	settlement := el("ram:SpecifiedLineTradeSettlement", tax)
	for i := range l.AllowanceCharges { // BG-27/28 remises/charges de ligne
		settlement.with(buildAllowanceCharge(&l.AllowanceCharges[i]))
	}
	settlement.with(el("ram:SpecifiedTradeSettlementLineMonetarySummation",
		leaf("ram:LineTotalAmount", l.NetAmount.String()),
	))

	return el("ram:IncludedSupplyChainTradeLineItem",
		assoc, product, agreement, delivery, settlement,
	)
}

func buildAgreement(doc *model.Document) *node {
	a := el("ram:ApplicableHeaderTradeAgreement")
	a.with(leaf("ram:BuyerReference", doc.BuyerReference))
	a.with(buildParty("ram:SellerTradeParty", doc.Seller))
	a.with(buildParty("ram:BuyerTradeParty", doc.Buyer))
	if doc.PurchaseOrderRef != "" {
		a.with(el("ram:BuyerOrderReferencedDocument", leaf("ram:IssuerAssignedID", doc.PurchaseOrderRef)))
	}
	return a
}

func buildParty(tag string, p model.Party) *node {
	party := el(tag)
	party.with(leaf("ram:ID", p.ID)) // BT-29 identifiant de partie
	party.with(leaf("ram:Name", p.Name))
	if p.LegalID != "" {
		lo := el("ram:SpecifiedLegalOrganization")
		lo.with(leafA("ram:ID", p.LegalID, attr{"schemeID", p.LegalIDScheme}))
		if p.TradingName != "" {
			lo.with(leaf("ram:TradingBusinessName", p.TradingName))
		}
		party.with(lo)
	}
	if p.Contact != nil {
		c := el("ram:DefinedTradeContact",
			leaf("ram:PersonName", p.Contact.Name))
		if p.Contact.Phone != "" {
			c.with(el("ram:TelephoneUniversalCommunication", leaf("ram:CompleteNumber", p.Contact.Phone)))
		}
		if p.Contact.Email != "" {
			c.with(el("ram:EmailURIUniversalCommunication", leaf("ram:URIID", p.Contact.Email)))
		}
		party.with(c)
	}
	party.with(buildAddress(p.Address))
	if p.ElectronicAddr != nil && p.ElectronicAddr.Value != "" {
		party.with(el("ram:URIUniversalCommunication",
			leafA("ram:URIID", p.ElectronicAddr.Value, attr{"schemeID", p.ElectronicAddr.Scheme})))
	}
	if p.VATID != "" {
		party.with(el("ram:SpecifiedTaxRegistration",
			leafA("ram:ID", p.VATID, attr{"schemeID", "VA"})))
	}
	return party
}

func buildAddress(a model.Address) *node {
	if a.Empty() {
		return nil
	}
	return el("ram:PostalTradeAddress",
		leaf("ram:PostcodeCode", a.PostalCode),
		leaf("ram:LineOne", a.Line1),
		leaf("ram:LineTwo", a.Line2),
		leaf("ram:CityName", a.City),
		leaf("ram:CountryID", a.CountryCode),
	)
}

func buildDelivery(doc *model.Document) *node {
	d := el("ram:ApplicableHeaderTradeDelivery")
	if doc.DeliverTo != nil {
		ship := el("ram:ShipToTradeParty", leaf("ram:Name", doc.DeliverTo.Name))
		ship.with(buildAddress(doc.DeliverTo.Address))
		d.with(ship)
		if doc.DeliverTo.DeliveryDate != nil {
			d.with(el("ram:ActualDeliverySupplyChainEvent",
				el("ram:OccurrenceDateTime",
					leafA("udt:DateTimeString", doc.DeliverTo.DeliveryDate.Compact(), attr{"format", "102"}),
				)))
		}
	}
	return d
}

func buildSettlement(doc *model.Document) *node {
	s := el("ram:ApplicableHeaderTradeSettlement")
	if doc.PaymentInstructions != nil {
		s.with(leaf("ram:PaymentReference", doc.PaymentInstructions.RemittanceInfo))
	}
	s.with(leaf("ram:InvoiceCurrencyCode", doc.CurrencyCode))

	if pi := doc.PaymentInstructions; pi != nil {
		pm := el("ram:SpecifiedTradeSettlementPaymentMeans", leaf("ram:TypeCode", string(pm0(pi.MeansCode))))
		for _, ct := range pi.CreditTransfers {
			if ct.IBAN != "" {
				pm.with(el("ram:PayeePartyCreditorFinancialAccount", leaf("ram:IBANID", ct.IBAN)))
			}
		}
		s.with(pm)
	}

	for _, ts := range doc.TaxBreakdown {
		s.with(el("ram:ApplicableTradeTax",
			leaf("ram:CalculatedAmount", ts.TaxAmount.String()),
			leaf("ram:TypeCode", "VAT"),
			leaf("ram:ExemptionReason", ts.ExemptionReason),
			leaf("ram:BasisAmount", ts.TaxableAmount.String()),
			leaf("ram:CategoryCode", string(ts.Category)),
			leaf("ram:ExemptionReasonCode", ts.ExemptionReasonCode),
			leaf("ram:RateApplicablePercent", ts.Rate.String()),
		))
	}

	for i := range doc.AllowanceCharges {
		s.with(buildAllowanceCharge(&doc.AllowanceCharges[i]))
	}

	if doc.DueDate != nil || doc.PaymentTerms != "" {
		pt := el("ram:SpecifiedTradePaymentTerms")
		pt.with(leaf("ram:Description", doc.PaymentTerms))
		if doc.DueDate != nil {
			pt.with(el("ram:DueDateDateTime",
				leafA("udt:DateTimeString", doc.DueDate.Compact(), attr{"format", "102"})))
		}
		s.with(pt)
	}

	s.with(buildMonetarySummation(doc))
	return s
}

func buildMonetarySummation(doc *model.Document) *node {
	t := doc.Totals
	sum := el("ram:SpecifiedTradeSettlementHeaderMonetarySummation",
		leaf("ram:LineTotalAmount", t.LineExtensionAmount.String()),
	)
	if t.ChargeTotal != nil {
		sum.with(leaf("ram:ChargeTotalAmount", t.ChargeTotal.String()))
	}
	if t.AllowanceTotal != nil {
		sum.with(leaf("ram:AllowanceTotalAmount", t.AllowanceTotal.String()))
	}
	sum.with(
		leaf("ram:TaxBasisTotalAmount", t.TaxExclusiveAmount.String()),
		leafA("ram:TaxTotalAmount", t.TaxAmount.String(), attr{"currencyID", doc.CurrencyCode}),
		leaf("ram:GrandTotalAmount", t.TaxInclusiveAmount.String()),
	)
	if t.PrepaidAmount != nil {
		sum.with(leaf("ram:TotalPrepaidAmount", t.PrepaidAmount.String()))
	}
	sum.with(leaf("ram:DuePayableAmount", t.DuePayableAmount.String()))
	return sum
}

func buildAllowanceCharge(ac *model.AllowanceCharge) *node {
	ind := "false"
	if ac.IsCharge {
		ind = "true"
	}
	n := el("ram:SpecifiedTradeAllowanceCharge",
		el("ram:ChargeIndicator", leaf("udt:Indicator", ind)))
	if ac.Percent != nil {
		n.with(leaf("ram:CalculationPercent", ac.Percent.String()))
	}
	if ac.BaseAmount != nil {
		n.with(leaf("ram:BasisAmount", ac.BaseAmount.String()))
	}
	n.with(leaf("ram:ActualAmount", ac.Amount.String()))
	n.with(leaf("ram:ReasonCode", ac.ReasonCode))
	n.with(leaf("ram:Reason", ac.Reason))
	if ac.TaxCategory != "" {
		ct := el("ram:CategoryTradeTax",
			leaf("ram:TypeCode", "VAT"),
			leaf("ram:CategoryCode", string(ac.TaxCategory)))
		if ac.TaxRate != nil {
			ct.with(leaf("ram:RateApplicablePercent", ac.TaxRate.String()))
		}
		n.with(ct)
	}
	return n
}

// pm0 renvoie un code moyen de paiement par défaut si absent (virement).
func pm0(c model.PaymentMeansCode) model.PaymentMeansCode {
	if c == "" {
		return model.PayCredit
	}
	return c
}
