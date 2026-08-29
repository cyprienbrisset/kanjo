package ubl

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// mapToPivot convertit les structures UBL désérialisées en document pivot.
func mapToPivot(x *ublDocument, sourceName string) (*model.Document, error) {
	currency := strings.TrimSpace(x.CurrencyCode)
	if currency == "" {
		return nil, fmt.Errorf("lecture UBL %s: devise (BT-5) absente", sourceName)
	}

	creditNote := strings.EqualFold(x.XMLName.Local, "CreditNote")

	typeCode := model.TypeCode(strings.TrimSpace(x.InvoiceTypeCode))
	if creditNote {
		typeCode = model.TypeCode(strings.TrimSpace(x.CreditNoteType))
	}
	kind := model.KindInvoice
	if creditNote || typeCode.IsCreditNote() {
		kind = model.KindCreditNote
	}

	doc := model.NewDocument(kind)
	doc.ID = strings.TrimSpace(x.ID)
	doc.TypeCode = typeCode
	doc.CurrencyCode = currency

	if v := strings.TrimSpace(x.IssueDate); v != "" {
		issue, err := model.ParseISO(v)
		if err != nil {
			return nil, fmt.Errorf("lecture UBL %s: date d'émission (BT-2): %w", sourceName, err)
		}
		doc.IssueDate = issue
	}
	if v := strings.TrimSpace(x.DueDate); v != "" {
		due, err := model.ParseISO(v)
		if err != nil {
			return nil, fmt.Errorf("lecture UBL %s: échéance (BT-9): %w", sourceName, err)
		}
		doc.DueDate = &due
	}

	for _, n := range x.Notes {
		if strings.TrimSpace(n) == "" {
			continue
		}
		doc.Notes = append(doc.Notes, model.Note{Content: n})
	}

	doc.BuyerReference = strings.TrimSpace(x.BuyerReference)
	doc.PurchaseOrderRef = strings.TrimSpace(x.OrderReference.ID)
	doc.Seller = mapParty(x.SupplierParty.Party)
	doc.Buyer = mapParty(x.CustomerParty.Party)

	if err := mapDelivery(doc, &x.Delivery); err != nil {
		return nil, fmt.Errorf("lecture UBL %s: livraison: %w", sourceName, err)
	}

	doc.PaymentTerms = strings.TrimSpace(x.PaymentTerms.Note)
	mapPayment(doc, &x.PaymentMeans)

	for _, ac := range x.AllowanceCharge {
		mac, err := mapAllowanceCharge(ac, currency)
		if err != nil {
			return nil, fmt.Errorf("lecture UBL %s: remise/charge document: %w", sourceName, err)
		}
		doc.AllowanceCharges = append(doc.AllowanceCharges, mac)
	}

	// EN 16931 autorise un second cac:TaxTotal dans la devise de comptabilisation de la TVA
	// (BT-111). On ne retient que celui exprimé dans la devise du document (BT-5).
	tt := selectTaxTotal(x.TaxTotal, currency)
	for i, ts := range tt.TaxSubtotal {
		sub, err := mapTax(ts, currency)
		if err != nil {
			return nil, fmt.Errorf("lecture UBL %s: ventilation TVA #%d: %w", sourceName, i+1, err)
		}
		doc.TaxBreakdown = append(doc.TaxBreakdown, sub)
	}

	lines := x.InvoiceLines
	if creditNote {
		lines = x.CreditNoteLines
	}
	for i := range lines {
		line, err := mapLine(&lines[i], creditNote, currency)
		if err != nil {
			return nil, fmt.Errorf("lecture UBL %s: ligne #%d: %w", sourceName, i+1, err)
		}
		doc.Lines = append(doc.Lines, line)
	}

	if err := mapTotals(doc, &x.LegalMonetaryTotal, tt.TaxAmount.Value, currency); err != nil {
		return nil, fmt.Errorf("lecture UBL %s: totaux: %w", sourceName, err)
	}

	doc.Provenance = model.NewProvenance(sourceName, "ubl", profileFromURN(x.CustomizationID))
	doc.Provenance.Record("BT-1", "/Invoice/cbc:ID")
	doc.Provenance.Record("BT-5", "/Invoice/cbc:DocumentCurrencyCode")
	return doc, nil
}

func mapParty(p ublParty) model.Party {
	// BT-27/BT-44 (nom) = cac:PartyLegalEntity/cbc:RegistrationName ; repli sur cac:PartyName
	// (qui porte plutôt BT-28/BT-45, le nom commercial).
	name := strings.TrimSpace(p.PartyLegalEntity.RegistrationName)
	trading := strings.TrimSpace(p.PartyName.Name)
	if name == "" {
		name = trading
		trading = ""
	}
	out := model.Party{
		Name:          name,
		TradingName:   trading,
		ID:            strings.TrimSpace(p.Identification.ID),
		LegalID:       strings.TrimSpace(p.PartyLegalEntity.CompanyID.Value),
		LegalIDScheme: strings.TrimSpace(p.PartyLegalEntity.CompanyID.Scheme),
		VATID:         strings.TrimSpace(p.PartyTaxScheme.CompanyID),
		Address:       mapAddress(p.PostalAddress),
	}
	if v := strings.TrimSpace(p.Endpoint.Value); v != "" {
		out.ElectronicAddr = &model.ElectronicAddress{Value: v, Scheme: strings.TrimSpace(p.Endpoint.Scheme)}
	}
	if c := p.Contact; c.Name != "" || c.Phone != "" || c.Email != "" {
		out.Contact = &model.Contact{
			Name:  strings.TrimSpace(c.Name),
			Phone: strings.TrimSpace(c.Phone),
			Email: strings.TrimSpace(c.Email),
		}
	}
	return out
}

func mapAddress(a ublAddress) model.Address {
	return model.Address{
		Line1:       strings.TrimSpace(a.StreetName),
		Line2:       strings.TrimSpace(a.AdditionalName),
		City:        strings.TrimSpace(a.CityName),
		PostalCode:  strings.TrimSpace(a.PostalZone),
		CountryCode: strings.TrimSpace(a.CountryCode),
	}
}

func mapDelivery(doc *model.Document, d *ublDelivery) error {
	name := strings.TrimSpace(d.DeliveryParty.Name)
	addr := mapAddress(d.DeliveryLocation.Address)
	var date *model.Date
	if v := strings.TrimSpace(d.ActualDeliveryDate); v != "" {
		parsed, err := model.ParseISO(v)
		if err != nil {
			return err
		}
		date = &parsed
	}
	if name == "" && addr.Empty() && date == nil {
		return nil
	}
	doc.DeliverTo = &model.DeliveryInfo{
		Name:         name,
		Address:      addr,
		DeliveryDate: date,
	}
	return nil
}

func mapPayment(doc *model.Document, pm *ublPaymentMeans) {
	code := strings.TrimSpace(pm.Code)
	ref := strings.TrimSpace(pm.PayID)
	iban := strings.TrimSpace(pm.Account.ID)
	if code == "" && ref == "" && iban == "" {
		return
	}
	pi := &model.PaymentInstructions{RemittanceInfo: ref}
	if code != "" {
		pi.MeansCode = model.PaymentMeansCode(code)
	}
	if iban != "" {
		pi.CreditTransfers = append(pi.CreditTransfers, model.CreditTransfer{IBAN: iban})
	}
	doc.PaymentInstructions = pi
}

// mapAllowanceCharge convertit une remise/charge de niveau document (BG-20/21).
func mapAllowanceCharge(ac ublAllowanceCharge, currency string) (model.AllowanceCharge, error) {
	amount, err := model.ParseAmount(orZeroStr(ac.Amount), currency)
	if err != nil {
		return model.AllowanceCharge{}, err
	}
	out := model.AllowanceCharge{
		IsCharge:    strings.EqualFold(strings.TrimSpace(ac.ChargeIndicator), "true"),
		Amount:      amount,
		ReasonCode:  strings.TrimSpace(ac.ReasonCode),
		Reason:      strings.TrimSpace(ac.Reason),
		TaxCategory: model.TaxCategoryCode(strings.TrimSpace(ac.Category.ID)),
	}
	if v := strings.TrimSpace(ac.BaseAmount); v != "" {
		b, err := model.ParseAmount(v, currency)
		if err != nil {
			return model.AllowanceCharge{}, err
		}
		out.BaseAmount = &b
	}
	if v := strings.TrimSpace(ac.Percent); v != "" {
		p, err := model.ParseDecimal(v)
		if err == nil {
			out.Percent = &p
		}
	}
	if v := strings.TrimSpace(ac.Category.Percent); v != "" {
		r, err := model.ParseDecimal(v)
		if err == nil {
			out.TaxRate = &r
		}
	}
	return out, nil
}

// selectTaxTotal retient le cac:TaxTotal exprimé dans la devise du document ; à défaut, le
// premier disponible.
func selectTaxTotal(totals []ublTaxTotal, currency string) ublTaxTotal {
	for _, t := range totals {
		if strings.EqualFold(strings.TrimSpace(t.TaxAmount.Currency), currency) {
			return t
		}
	}
	if len(totals) > 0 {
		return totals[0]
	}
	return ublTaxTotal{}
}

func mapTax(ts ublTaxSubtotal, currency string) (model.TaxSubtotal, error) {
	basis, err := model.ParseAmount(orZeroStr(ts.TaxableAmount), currency)
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("base (BT-116): %w", err)
	}
	amount, err := model.ParseAmount(orZeroStr(ts.TaxAmount), currency)
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("montant (BT-117): %w", err)
	}
	rate, err := model.ParseDecimal(orZeroStr(ts.Category.Percent))
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("taux (BT-119): %w", err)
	}
	return model.TaxSubtotal{
		Category:            model.TaxCategoryCode(strings.TrimSpace(ts.Category.ID)),
		Rate:                rate,
		TaxableAmount:       basis,
		TaxAmount:           amount,
		ExemptionReason:     strings.TrimSpace(ts.Category.ExemptionReason),
		ExemptionReasonCode: strings.TrimSpace(ts.Category.ExemptionReasonCode),
	}, nil
}

func mapLine(l *ublLine, creditNote bool, currency string) (model.Line, error) {
	qtyStr := l.Quantity.Value
	unit := l.Quantity.UnitCode
	if creditNote {
		qtyStr = l.CreditedQuantity.Value
		unit = l.CreditedQuantity.UnitCode
	}
	qty, err := model.ParseDecimal(orZeroStr(qtyStr))
	if err != nil {
		return model.Line{}, fmt.Errorf("quantité (BT-129): %w", err)
	}
	netPrice, err := model.ParseAmount(orZeroStr(l.Price.PriceAmount), currency)
	if err != nil {
		return model.Line{}, fmt.Errorf("prix unitaire (BT-146): %w", err)
	}
	netAmount, err := model.ParseAmount(orZeroStr(l.LineExtensionAmount), currency)
	if err != nil {
		return model.Line{}, fmt.Errorf("montant net (BT-131): %w", err)
	}
	out := model.Line{
		ID:               strings.TrimSpace(l.ID),
		Note:             strings.TrimSpace(l.Note),
		Name:             strings.TrimSpace(l.Item.Name),
		Description:      strings.TrimSpace(l.Item.Description),
		SellerAssignedID: strings.TrimSpace(l.Item.SellersItemID),
		BuyerAssignedID:  strings.TrimSpace(l.Item.BuyersItemID),
		StandardID:       strings.TrimSpace(l.Item.StandardItemID.Value),
		StandardScheme:   strings.TrimSpace(l.Item.StandardItemID.Scheme),
		Quantity:         qty,
		UnitCode:         model.UnitCode(strings.TrimSpace(unit)),
		NetPrice:         netPrice,
		NetAmount:        netAmount,
		TaxCategory:      model.TaxCategoryCode(strings.TrimSpace(l.Item.ClassifiedTaxCategory.ID)),
	}
	if r := strings.TrimSpace(l.Item.ClassifiedTaxCategory.Percent); r != "" {
		rate, err := model.ParseDecimal(r)
		if err != nil {
			return model.Line{}, fmt.Errorf("taux ligne (BT-152): %w", err)
		}
		out.TaxRate = &rate
	}
	if bq := strings.TrimSpace(l.Price.BaseQuantity.Value); bq != "" {
		base, err := model.ParseDecimal(bq)
		if err != nil {
			return model.Line{}, fmt.Errorf("quantité de base (BT-149): %w", err)
		}
		out.PriceBaseQty = &base
	}
	for _, ac := range l.AllowanceCharge {
		mac, err := mapAllowanceCharge(ac, currency)
		if err != nil {
			return model.Line{}, fmt.Errorf("remise/charge de ligne (BG-27/28): %w", err)
		}
		out.AllowanceCharges = append(out.AllowanceCharges, mac)
	}
	return out, nil
}

func mapTotals(doc *model.Document, m *ublLegalMonetaryTotal, taxTotal, currency string) error {
	var err error
	if doc.Totals.LineExtensionAmount, err = model.ParseAmount(orZeroStr(m.LineExtensionAmount), currency); err != nil {
		return fmt.Errorf("total lignes (BT-106): %w", err)
	}
	if doc.Totals.TaxExclusiveAmount, err = model.ParseAmount(orZeroStr(m.TaxExclusiveAmount), currency); err != nil {
		return fmt.Errorf("total HT (BT-109): %w", err)
	}
	if doc.Totals.TaxAmount, err = model.ParseAmount(orZeroStr(taxTotal), currency); err != nil {
		return fmt.Errorf("total TVA (BT-110): %w", err)
	}
	if doc.Totals.TaxInclusiveAmount, err = model.ParseAmount(orZeroStr(m.TaxInclusiveAmount), currency); err != nil {
		return fmt.Errorf("total TTC (BT-112): %w", err)
	}
	if doc.Totals.DuePayableAmount, err = model.ParseAmount(orZeroStr(m.PayableAmount), currency); err != nil {
		return fmt.Errorf("net à payer (BT-115): %w", err)
	}
	if v := strings.TrimSpace(m.AllowanceTotal); v != "" {
		a, err := model.ParseAmount(v, currency)
		if err != nil {
			return fmt.Errorf("total remises (BT-107): %w", err)
		}
		doc.Totals.AllowanceTotal = &a
	}
	if v := strings.TrimSpace(m.ChargeTotal); v != "" {
		a, err := model.ParseAmount(v, currency)
		if err != nil {
			return fmt.Errorf("total charges (BT-108): %w", err)
		}
		doc.Totals.ChargeTotal = &a
	}
	if v := strings.TrimSpace(m.PrepaidAmount); v != "" {
		a, err := model.ParseAmount(v, currency)
		if err != nil {
			return fmt.Errorf("acompte (BT-113): %w", err)
		}
		doc.Totals.PrepaidAmount = &a
	}
	return nil
}

// orZeroStr renvoie "0" pour une chaîne vide, sinon la chaîne nettoyée.
func orZeroStr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "0"
	}
	return s
}

// profileFromURN déduit le nom de profil depuis l'URN de personnalisation.
func profileFromURN(urn string) string {
	u := strings.ToLower(urn)
	switch {
	case strings.Contains(u, "xrechnung"):
		return "xrechnung"
	case strings.Contains(u, "extended"):
		return "extended"
	case strings.Contains(u, "basicwl"):
		return "basicwl"
	case strings.Contains(u, "basic"):
		return "basic"
	case strings.Contains(u, "minimum"):
		return "minimum"
	case strings.Contains(u, "en16931"):
		return "en16931"
	default:
		return ""
	}
}
