// Package ubl écrit un document pivot au format UBL 2.1 (Invoice ou CreditNote).
// L'écriture utilise un arbre XML à préfixes explicites (cac:/cbc:).
package ubl

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func init() { write.Register("ubl", Write) }

// Espaces de noms UBL 2.1.
const (
	nsInvoice    = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	nsCreditNote = "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2"
	nsCAC        = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	nsCBC        = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
)

// customizationURN renvoie l'identifiant de personnalisation pour un profil donné.
func customizationURN(p write.Profile) string {
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

// Write sérialise le document pivot en UBL 2.1.
func Write(doc *model.Document, opts write.Options) ([]byte, error) {
	if opts.Profile == "" {
		opts.Profile = write.ProfileEN16931
	}
	creditNote := doc.IsCreditNote()

	rootName := "Invoice"
	rootNS := nsInvoice
	if creditNote {
		rootName = "CreditNote"
		rootNS = nsCreditNote
	}
	root := el(rootName)
	root.attrs = []attr{
		{"xmlns", rootNS},
		{"xmlns:cac", nsCAC},
		{"xmlns:cbc", nsCBC},
	}

	custom := customizationURN(opts.Profile)
	if opts.CustomizationID != "" {
		custom = opts.CustomizationID
	}
	root.with(leaf("cbc:CustomizationID", custom))
	if opts.ProfileID != "" {
		root.with(leaf("cbc:ProfileID", opts.ProfileID))
	}
	root.with(
		leaf("cbc:ID", doc.ID),
		leaf("cbc:IssueDate", doc.IssueDate.ISO()),
	)
	if doc.DueDate != nil {
		root.with(leaf("cbc:DueDate", doc.DueDate.ISO()))
	}
	if creditNote {
		root.with(leaf("cbc:CreditNoteTypeCode", string(doc.TypeCode)))
	} else {
		root.with(leaf("cbc:InvoiceTypeCode", string(doc.TypeCode)))
	}
	for _, note := range doc.Notes {
		root.with(leaf("cbc:Note", note.Content))
	}
	root.with(
		leaf("cbc:DocumentCurrencyCode", doc.CurrencyCode),
		leaf("cbc:BuyerReference", doc.BuyerReference),
	)
	if doc.PurchaseOrderRef != "" {
		root.with(el("cac:OrderReference", leaf("cbc:ID", doc.PurchaseOrderRef)))
	}

	root.with(
		el("cac:AccountingSupplierParty", buildParty(doc.Seller)),
		el("cac:AccountingCustomerParty", buildParty(doc.Buyer)),
	)
	if d := buildDelivery(doc); d != nil {
		root.with(d)
	}
	if pm := buildPaymentMeans(doc); pm != nil {
		root.with(pm)
	}
	if doc.PaymentTerms != "" {
		root.with(el("cac:PaymentTerms", leaf("cbc:Note", doc.PaymentTerms)))
	}
	for i := range doc.AllowanceCharges {
		root.with(buildAllowanceCharge(&doc.AllowanceCharges[i]))
	}
	root.with(buildTaxTotal(doc))
	root.with(buildLegalMonetaryTotal(doc))
	for i := range doc.Lines {
		root.with(buildLine(doc, &doc.Lines[i], creditNote))
	}

	return []byte(render(root)), nil
}

func buildParty(p model.Party) *node {
	party := el("cac:Party")
	// BT-34 : adresse électronique = cbc:EndpointID ; BT-29 : identifiant = cac:PartyIdentification.
	if p.ElectronicAddr != nil && p.ElectronicAddr.Value != "" {
		if p.ElectronicAddr.Scheme != "" {
			party.with(leafA("cbc:EndpointID", p.ElectronicAddr.Value, attr{"schemeID", p.ElectronicAddr.Scheme}))
		} else {
			party.with(leaf("cbc:EndpointID", p.ElectronicAddr.Value))
		}
	}
	if p.ID != "" {
		party.with(el("cac:PartyIdentification", leaf("cbc:ID", p.ID)))
	}
	// cac:PartyName/cbc:Name porte le nom commercial (BT-28/BT-45) ; le nom principal
	// (BT-27/BT-44) va dans cac:PartyLegalEntity/cbc:RegistrationName ci-dessous.
	if p.TradingName != "" {
		party.with(el("cac:PartyName", leaf("cbc:Name", p.TradingName)))
	}
	party.with(buildAddress(p.Address))
	if p.VATID != "" {
		party.with(el("cac:PartyTaxScheme",
			leaf("cbc:CompanyID", p.VATID),
			el("cac:TaxScheme", leaf("cbc:ID", "VAT")),
		))
	}
	legal := el("cac:PartyLegalEntity", leaf("cbc:RegistrationName", legalName(p)))
	if p.LegalID != "" {
		if p.LegalIDScheme != "" {
			legal.with(leafA("cbc:CompanyID", p.LegalID, attr{"schemeID", p.LegalIDScheme}))
		} else {
			legal.with(leaf("cbc:CompanyID", p.LegalID))
		}
	}
	if !legal.empty() {
		party.with(legal)
	}
	if p.Contact != nil && (p.Contact.Name != "" || p.Contact.Phone != "" || p.Contact.Email != "") {
		party.with(el("cac:Contact",
			leaf("cbc:Name", p.Contact.Name),
			leaf("cbc:Telephone", p.Contact.Phone),
			leaf("cbc:ElectronicMail", p.Contact.Email),
		))
	}
	return party
}

// legalName renvoie le nom légal (BT-27) ; on retombe sur le nom courant si le nom légal
// dédié n'est pas distinct dans le pivot.
func legalName(p model.Party) string {
	if p.TradingName != "" {
		return p.Name
	}
	return p.Name
}

func buildAddress(a model.Address) *node {
	if a.Empty() {
		return nil
	}
	addr := el("cac:PostalAddress",
		leaf("cbc:StreetName", a.Line1),
		leaf("cbc:AdditionalStreetName", a.Line2),
		leaf("cbc:CityName", a.City),
		leaf("cbc:PostalZone", a.PostalCode),
	)
	if a.CountryCode != "" {
		addr.with(el("cac:Country", leaf("cbc:IdentificationCode", a.CountryCode)))
	}
	return addr
}

func buildDelivery(doc *model.Document) *node {
	if doc.DeliverTo == nil {
		return nil
	}
	d := el("cac:Delivery")
	if doc.DeliverTo.DeliveryDate != nil {
		d.with(leaf("cbc:ActualDeliveryDate", doc.DeliverTo.DeliveryDate.ISO()))
	}
	loc := el("cac:DeliveryLocation")
	if addr := buildAddress(doc.DeliverTo.Address); addr != nil {
		loc.with(el("cac:Address").with(addr.children...))
	}
	if !loc.empty() {
		d.with(loc)
	}
	if doc.DeliverTo.Name != "" {
		d.with(el("cac:DeliveryParty",
			el("cac:PartyName", leaf("cbc:Name", doc.DeliverTo.Name))))
	}
	if d.empty() {
		return nil
	}
	return d
}

func buildPaymentMeans(doc *model.Document) *node {
	pi := doc.PaymentInstructions
	if pi == nil {
		return nil
	}
	pm := el("cac:PaymentMeans",
		leaf("cbc:PaymentMeansCode", string(pi.MeansCode)),
		leaf("cbc:PaymentID", pi.RemittanceInfo),
	)
	for _, ct := range pi.CreditTransfers {
		if ct.IBAN != "" {
			pm.with(el("cac:PayeeFinancialAccount", leaf("cbc:ID", ct.IBAN)))
		}
	}
	if pm.empty() {
		return nil
	}
	return pm
}

func buildAllowanceCharge(ac *model.AllowanceCharge) *node {
	cur := ac.Amount.Currency
	ind := "false"
	if ac.IsCharge {
		ind = "true"
	}
	n := el("cac:AllowanceCharge", leaf("cbc:ChargeIndicator", ind))
	if ac.Percent != nil {
		n.with(leaf("cbc:MultiplierFactorNumeric", ac.Percent.String()))
	}
	n.with(leafA("cbc:Amount", ac.Amount.String(), attr{"currencyID", cur}))
	if ac.BaseAmount != nil {
		n.with(leafA("cbc:BaseAmount", ac.BaseAmount.String(), attr{"currencyID", cur}))
	}
	n.with(leaf("cbc:AllowanceChargeReasonCode", ac.ReasonCode))
	n.with(leaf("cbc:AllowanceChargeReason", ac.Reason))
	if ac.TaxCategory != "" {
		cat := el("cac:TaxCategory", leaf("cbc:ID", string(ac.TaxCategory)))
		if ac.TaxRate != nil {
			cat.with(leaf("cbc:Percent", ac.TaxRate.String()))
		}
		cat.with(el("cac:TaxScheme", leaf("cbc:ID", "VAT")))
		n.with(cat)
	}
	return n
}

func buildTaxTotal(doc *model.Document) *node {
	cur := doc.CurrencyCode
	tt := el("cac:TaxTotal",
		leafA("cbc:TaxAmount", doc.Totals.TaxAmount.String(), attr{"currencyID", cur}),
	)
	for _, ts := range doc.TaxBreakdown {
		sub := el("cac:TaxSubtotal",
			leafA("cbc:TaxableAmount", ts.TaxableAmount.String(), attr{"currencyID", cur}),
			leafA("cbc:TaxAmount", ts.TaxAmount.String(), attr{"currencyID", cur}),
			el("cac:TaxCategory",
				leaf("cbc:ID", string(ts.Category)),
				leaf("cbc:Percent", ts.Rate.String()),
				leaf("cbc:TaxExemptionReasonCode", ts.ExemptionReasonCode),
				leaf("cbc:TaxExemptionReason", ts.ExemptionReason),
				el("cac:TaxScheme", leaf("cbc:ID", "VAT")),
			),
		)
		tt.with(sub)
	}
	return tt
}

func buildLegalMonetaryTotal(doc *model.Document) *node {
	cur := doc.CurrencyCode
	t := doc.Totals
	lmt := el("cac:LegalMonetaryTotal",
		leafA("cbc:LineExtensionAmount", t.LineExtensionAmount.String(), attr{"currencyID", cur}),
		leafA("cbc:TaxExclusiveAmount", t.TaxExclusiveAmount.String(), attr{"currencyID", cur}),
		leafA("cbc:TaxInclusiveAmount", t.TaxInclusiveAmount.String(), attr{"currencyID", cur}),
	)
	if t.AllowanceTotal != nil {
		lmt.with(leafA("cbc:AllowanceTotalAmount", t.AllowanceTotal.String(), attr{"currencyID", cur}))
	}
	if t.ChargeTotal != nil {
		lmt.with(leafA("cbc:ChargeTotalAmount", t.ChargeTotal.String(), attr{"currencyID", cur}))
	}
	if t.PrepaidAmount != nil {
		lmt.with(leafA("cbc:PrepaidAmount", t.PrepaidAmount.String(), attr{"currencyID", cur}))
	}
	lmt.with(leafA("cbc:PayableAmount", t.DuePayableAmount.String(), attr{"currencyID", cur}))
	return lmt
}

func buildLine(doc *model.Document, l *model.Line, creditNote bool) *node {
	cur := doc.CurrencyCode
	lineTag := "cac:InvoiceLine"
	qtyTag := "cbc:InvoicedQuantity"
	if creditNote {
		lineTag = "cac:CreditNoteLine"
		qtyTag = "cbc:CreditedQuantity"
	}
	line := el(lineTag,
		leaf("cbc:ID", l.ID),
	)
	if l.Note != "" {
		line.with(leaf("cbc:Note", l.Note))
	}
	line.with(
		leafA(qtyTag, l.Quantity.String(), attr{"unitCode", string(l.UnitCode)}),
		leafA("cbc:LineExtensionAmount", l.NetAmount.String(), attr{"currencyID", cur}),
	)
	for i := range l.AllowanceCharges { // BG-27/28 remises/charges de ligne
		line.with(buildAllowanceCharge(&l.AllowanceCharges[i]))
	}

	item := el("cac:Item",
		leaf("cbc:Description", l.Description),
		leaf("cbc:Name", l.Name),
	)
	if l.BuyerAssignedID != "" {
		item.with(el("cac:BuyersItemIdentification", leaf("cbc:ID", l.BuyerAssignedID)))
	}
	if l.SellerAssignedID != "" {
		item.with(el("cac:SellersItemIdentification", leaf("cbc:ID", l.SellerAssignedID)))
	}
	if l.StandardID != "" {
		if l.StandardScheme != "" {
			item.with(el("cac:StandardItemIdentification",
				leafA("cbc:ID", l.StandardID, attr{"schemeID", l.StandardScheme})))
		} else {
			item.with(el("cac:StandardItemIdentification", leaf("cbc:ID", l.StandardID)))
		}
	}
	cat := el("cac:ClassifiedTaxCategory", leaf("cbc:ID", string(l.TaxCategory)))
	if l.TaxRate != nil {
		cat.with(leaf("cbc:Percent", l.TaxRate.String()))
	}
	cat.with(el("cac:TaxScheme", leaf("cbc:ID", "VAT")))
	item.with(cat)
	line.with(item)

	price := el("cac:Price",
		leafA("cbc:PriceAmount", l.NetPrice.String(), attr{"currencyID", cur}),
	)
	if l.PriceBaseQty != nil {
		price.with(leafA("cbc:BaseQuantity", l.PriceBaseQty.String(), attr{"unitCode", string(l.UnitCode)}))
	}
	line.with(price)

	return line
}
