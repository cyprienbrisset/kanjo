package cii

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// mapToPivot convertit les structures CII désérialisées en document pivot.
func mapToPivot(x *ciiInvoice, sourceName string) (*model.Document, error) {
	currency := strings.TrimSpace(x.Transaction.Settlement.Currency)
	if currency == "" {
		return nil, fmt.Errorf("lecture CII %s: devise (BT-5) absente", sourceName)
	}

	typeCode := model.TypeCode(strings.TrimSpace(x.Doc.TypeCode))
	kind := model.KindInvoice
	if typeCode.IsCreditNote() {
		kind = model.KindCreditNote
	}
	doc := model.NewDocument(kind)
	doc.ID = strings.TrimSpace(x.Doc.ID)
	doc.TypeCode = typeCode
	doc.CurrencyCode = currency

	issue, err := x.Doc.IssueDateTime.parse()
	if err != nil {
		return nil, fmt.Errorf("lecture CII %s: date d'émission (BT-2): %w", sourceName, err)
	}
	if issue != nil {
		doc.IssueDate = *issue
	}

	for _, n := range x.Doc.Notes {
		if strings.TrimSpace(n.Content) == "" {
			continue
		}
		doc.Notes = append(doc.Notes, model.Note{Content: n.Content, SubjectCode: n.SubjectCode})
	}

	ag := x.Transaction.Agreement
	doc.BuyerReference = strings.TrimSpace(ag.BuyerReference)
	doc.PurchaseOrderRef = strings.TrimSpace(ag.Order.ID)
	doc.Seller = mapParty(ag.Seller)
	doc.Buyer = mapParty(ag.Buyer)

	if err := mapDelivery(doc, &x.Transaction.Delivery); err != nil {
		return nil, fmt.Errorf("lecture CII %s: livraison: %w", sourceName, err)
	}

	set := x.Transaction.Settlement
	doc.PaymentTerms = strings.TrimSpace(set.PaymentTerms.Description)
	if due, err := set.PaymentTerms.DueDate.parse(); err != nil {
		return nil, fmt.Errorf("lecture CII %s: échéance (BT-9): %w", sourceName, err)
	} else if due != nil {
		doc.DueDate = due
	}
	if err := mapPayment(doc, &set); err != nil {
		return nil, fmt.Errorf("lecture CII %s: paiement: %w", sourceName, err)
	}

	for _, a := range set.Allowances {
		mac, err := mapAllowance(a, currency)
		if err != nil {
			return nil, fmt.Errorf("lecture CII %s: remise/charge document: %w", sourceName, err)
		}
		doc.AllowanceCharges = append(doc.AllowanceCharges, mac)
	}

	for i, tx := range set.Tax {
		ts, err := mapTax(tx, currency)
		if err != nil {
			return nil, fmt.Errorf("lecture CII %s: ventilation TVA #%d: %w", sourceName, i+1, err)
		}
		doc.TaxBreakdown = append(doc.TaxBreakdown, ts)
		// BT-8 (code de date d'exigibilité TVA) : porté par la ventilation en CII ; on retient le premier.
		if doc.TaxPointDateCode == "" {
			doc.TaxPointDateCode = strings.TrimSpace(tx.DueDateTypeCode)
		}
	}

	for i := range x.Transaction.Lines {
		line, err := mapLine(&x.Transaction.Lines[i], currency)
		if err != nil {
			return nil, fmt.Errorf("lecture CII %s: ligne #%d: %w", sourceName, i+1, err)
		}
		doc.Lines = append(doc.Lines, line)
	}

	if err := mapTotals(doc, &set, currency); err != nil {
		return nil, fmt.Errorf("lecture CII %s: totaux: %w", sourceName, err)
	}

	doc.Provenance = model.NewProvenance(sourceName, "cii", profileFromURN(x.Context.Guideline.ID))
	doc.Provenance.SpecIdentifier = strings.TrimSpace(x.Context.Guideline.ID)
	doc.Provenance.Record("BT-1", "/rsm:CrossIndustryInvoice/rsm:ExchangedDocument/ram:ID")
	doc.Provenance.Record("BT-5", "/rsm:CrossIndustryInvoice/rsm:SupplyChainTradeTransaction/ram:ApplicableHeaderTradeSettlement/ram:InvoiceCurrencyCode")
	return doc, nil
}

func mapParty(p ciiParty) model.Party {
	id := strings.TrimSpace(p.DirectID)
	if id == "" {
		id = strings.TrimSpace(p.GlobalID.Value)
	}
	out := model.Party{
		Name:          strings.TrimSpace(p.Name),
		ID:            id,
		TradingName:   strings.TrimSpace(p.LegalOrg.TradingName),
		LegalID:       strings.TrimSpace(p.LegalOrg.ID.Value),
		LegalIDScheme: strings.TrimSpace(p.LegalOrg.ID.Scheme),
		Address:       mapAddress(p.Address),
	}
	for _, r := range p.TaxReg {
		if strings.EqualFold(strings.TrimSpace(r.Scheme), "VA") {
			out.VATID = strings.TrimSpace(r.ID)
		} else if r.Scheme == "FC" {
			out.TaxID = strings.TrimSpace(r.ID)
		}
	}
	if c := p.Contact; c.Name != "" || c.Phone != "" || c.Email != "" {
		out.Contact = &model.Contact{Name: c.Name, Phone: c.Phone, Email: c.Email}
	}
	if v := strings.TrimSpace(p.URI.Value); v != "" {
		out.ElectronicAddr = &model.ElectronicAddress{Value: v, Scheme: strings.TrimSpace(p.URI.Scheme)}
	}
	return out
}

func mapAddress(a ciiAddress) model.Address {
	return model.Address{
		Line1:       strings.TrimSpace(a.Line1),
		Line2:       strings.TrimSpace(a.Line2),
		City:        strings.TrimSpace(a.City),
		PostalCode:  strings.TrimSpace(a.PostalCode),
		CountryCode: strings.TrimSpace(a.CountryCode),
	}
}

func mapDelivery(doc *model.Document, d *ciiDelivery) error {
	hasParty := strings.TrimSpace(d.ShipTo.Name) != "" || !mapAddress(d.ShipTo.Address).Empty()
	date, err := d.Event.Date.parse()
	if err != nil {
		return err
	}
	if !hasParty && date == nil {
		return nil
	}
	doc.DeliverTo = &model.DeliveryInfo{
		Name:         strings.TrimSpace(d.ShipTo.Name),
		Address:      mapAddress(d.ShipTo.Address),
		DeliveryDate: date,
	}
	return nil
}

func mapPayment(doc *model.Document, s *ciiSettlement) error {
	if len(s.PaymentMeans) == 0 && s.PaymentReference == "" {
		return nil
	}
	pi := &model.PaymentInstructions{RemittanceInfo: strings.TrimSpace(s.PaymentReference)}
	for _, pm := range s.PaymentMeans {
		if tc := strings.TrimSpace(pm.TypeCode); tc != "" {
			pi.MeansCode = model.PaymentMeansCode(tc)
		}
		if iban := strings.TrimSpace(pm.IBAN); iban != "" {
			pi.CreditTransfers = append(pi.CreditTransfers, model.CreditTransfer{IBAN: iban})
		}
	}
	doc.PaymentInstructions = pi
	return nil
}

func mapTax(tx ciiHeaderTax, currency string) (model.TaxSubtotal, error) {
	basis, err := model.ParseAmount(orZeroStr(tx.BasisAmount), currency)
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("base (BT-116): %w", err)
	}
	amount, err := model.ParseAmount(orZeroStr(tx.CalculatedAmount), currency)
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("montant (BT-117): %w", err)
	}
	rate, err := model.ParseDecimal(orZeroStr(tx.Rate))
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("taux (BT-119): %w", err)
	}
	return model.TaxSubtotal{
		Category:            model.TaxCategoryCode(strings.TrimSpace(tx.CategoryCode)),
		Rate:                rate,
		RatePresent:         strings.TrimSpace(tx.Rate) != "",
		TaxableAmount:       basis,
		TaxAmount:           amount,
		ExemptionReason:     strings.TrimSpace(tx.ExemptionReason),
		ExemptionReasonCode: strings.TrimSpace(tx.ExemptionReasonCode),
	}, nil
}

func mapAllowance(a ciiAllowance, currency string) (model.AllowanceCharge, error) {
	amount, err := model.ParseAmount(orZeroStr(a.ActualAmount), currency)
	if err != nil {
		return model.AllowanceCharge{}, err
	}
	out := model.AllowanceCharge{
		IsCharge:    strings.EqualFold(strings.TrimSpace(a.Indicator), "true"),
		Amount:      amount,
		ReasonCode:  strings.TrimSpace(a.ReasonCode),
		Reason:      strings.TrimSpace(a.Reason),
		TaxCategory: model.TaxCategoryCode(strings.TrimSpace(a.Category.CategoryCode)),
	}
	if v := strings.TrimSpace(a.BasisAmount); v != "" {
		b, err := model.ParseAmount(v, currency)
		if err != nil {
			return model.AllowanceCharge{}, err
		}
		out.BaseAmount = &b
	}
	if v := strings.TrimSpace(a.Percent); v != "" {
		if p, err := model.ParseDecimal(v); err == nil {
			out.Percent = &p
		}
	}
	if v := strings.TrimSpace(a.Category.Rate); v != "" {
		if r, err := model.ParseDecimal(v); err == nil {
			out.TaxRate = &r
		}
	}
	return out, nil
}

func mapLine(l *ciiLine, currency string) (model.Line, error) {
	qty, err := model.ParseDecimal(orZeroStr(l.Delivery.BilledQuantity.Value))
	if err != nil {
		return model.Line{}, fmt.Errorf("quantité (BT-129): %w", err)
	}
	netPrice, err := model.ParseAmount(orZeroStr(l.Agreement.Net.ChargeAmount), currency)
	if err != nil {
		return model.Line{}, fmt.Errorf("prix net (BT-146): %w", err)
	}
	netAmount, err := model.ParseAmount(orZeroStr(l.Settlement.Sum.LineTotalAmount), currency)
	if err != nil {
		return model.Line{}, fmt.Errorf("montant net (BT-131): %w", err)
	}
	out := model.Line{
		ID:                   strings.TrimSpace(l.Doc.LineID),
		Note:                 strings.TrimSpace(l.Doc.Note.Content),
		Name:                 strings.TrimSpace(l.Product.Name),
		Description:          strings.TrimSpace(l.Product.Description),
		SellerAssignedID:     strings.TrimSpace(l.Product.SellerAssignedID),
		BuyerAssignedID:      strings.TrimSpace(l.Product.BuyerAssignedID),
		StandardID:           strings.TrimSpace(l.Product.GlobalID.Value),
		StandardScheme:       strings.TrimSpace(l.Product.GlobalID.Scheme),
		ClassificationID:     strings.TrimSpace(l.Product.Classification.Code.Value),
		ClassificationScheme: strings.TrimSpace(l.Product.Classification.Code.ListID),
		Quantity:             qty,
		QuantityPresent:      strings.TrimSpace(l.Delivery.BilledQuantity.Value) != "",
		UnitCode:             model.UnitCode(strings.TrimSpace(l.Delivery.BilledQuantity.UnitCode)),
		NetPrice:             netPrice,
		NetAmount:            netAmount,
		TaxCategory:          model.TaxCategoryCode(strings.TrimSpace(l.Settlement.Tax.CategoryCode)),
	}
	if r := strings.TrimSpace(l.Settlement.Tax.Rate); r != "" {
		rate, err := model.ParseDecimal(r)
		if err != nil {
			return model.Line{}, fmt.Errorf("taux ligne (BT-152): %w", err)
		}
		out.TaxRate = &rate
	}
	if bq := strings.TrimSpace(l.Agreement.Net.BasisQuantity.Value); bq != "" {
		base, err := model.ParseDecimal(bq)
		if err != nil {
			return model.Line{}, fmt.Errorf("quantité de base (BT-149): %w", err)
		}
		out.PriceBaseQty = &base
	}
	for _, a := range l.Settlement.Allowances {
		mac, err := mapAllowance(a, currency)
		if err != nil {
			return model.Line{}, fmt.Errorf("remise/charge de ligne (BG-27/28): %w", err)
		}
		out.AllowanceCharges = append(out.AllowanceCharges, mac)
	}
	if g := strings.TrimSpace(l.Agreement.Gross.ChargeAmount); g != "" {
		gross, err := model.ParseAmount(g, currency)
		if err != nil {
			return model.Line{}, fmt.Errorf("prix brut (BT-148): %w", err)
		}
		out.GrossPrice = &gross
		if d := strings.TrimSpace(l.Agreement.Gross.Allowance.Amount); d != "" {
			disc, err := model.ParseAmount(d, currency)
			if err != nil {
				return model.Line{}, fmt.Errorf("remise prix (BT-147): %w", err)
			}
			out.PriceDiscount = &disc
		}
	}
	return out, nil
}

func mapTotals(doc *model.Document, s *ciiSettlement, currency string) error {
	sum := s.Sum
	var err error
	if doc.Totals.LineExtensionAmount, err = model.ParseAmount(orZeroStr(sum.LineTotal), currency); err != nil {
		return fmt.Errorf("total lignes (BT-106): %w", err)
	}
	if doc.Totals.TaxExclusiveAmount, err = model.ParseAmount(orZeroStr(sum.TaxBasisTotal), currency); err != nil {
		return fmt.Errorf("total HT (BT-109): %w", err)
	}
	if doc.Totals.TaxAmount, err = model.ParseAmount(orZeroStr(selectTaxTotal(sum.TaxTotals, currency)), currency); err != nil {
		return fmt.Errorf("total TVA (BT-110): %w", err)
	}
	if doc.Totals.TaxInclusiveAmount, err = model.ParseAmount(orZeroStr(sum.GrandTotal), currency); err != nil {
		return fmt.Errorf("total TTC (BT-112): %w", err)
	}
	if doc.Totals.DuePayableAmount, err = model.ParseAmount(orZeroStr(sum.DuePayable), currency); err != nil {
		return fmt.Errorf("net à payer (BT-115): %w", err)
	}
	if v := strings.TrimSpace(sum.AllowanceTotal); v != "" {
		a, err := model.ParseAmount(v, currency)
		if err != nil {
			return fmt.Errorf("total remises (BT-107): %w", err)
		}
		doc.Totals.AllowanceTotal = &a
	}
	if v := strings.TrimSpace(sum.ChargeTotal); v != "" {
		a, err := model.ParseAmount(v, currency)
		if err != nil {
			return fmt.Errorf("total charges (BT-108): %w", err)
		}
		doc.Totals.ChargeTotal = &a
	}
	if v := strings.TrimSpace(sum.Prepaid); v != "" {
		a, err := model.ParseAmount(v, currency)
		if err != nil {
			return fmt.Errorf("acompte (BT-113): %w", err)
		}
		doc.Totals.PrepaidAmount = &a
	}
	return nil
}

// selectTaxTotal retient le TaxTotalAmount exprimé dans la devise du document (BT-110) ;
// ignore un éventuel second montant dans la devise de comptabilisation de la TVA (BT-111).
func selectTaxTotal(vals []currencyValue, currency string) string {
	for _, v := range vals {
		if strings.EqualFold(strings.TrimSpace(v.Currency), currency) {
			return v.Value
		}
	}
	if len(vals) > 0 {
		return vals[len(vals)-1].Value // à défaut de devise, le dernier (souvent la devise doc)
	}
	return "0"
}

// orZeroStr renvoie "0" pour une chaîne vide, sinon la chaîne nettoyée.
func orZeroStr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "0"
	}
	return s
}

// profileFromURN déduit le nom de profil depuis l'URN de la spécification.
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
