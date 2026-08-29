package zugferd1

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

func mapToPivot(x *ciDoc, sourceName string) (*model.Document, error) {
	currency := strings.TrimSpace(x.Tx.Settlement.Currency)
	if currency == "" {
		return nil, fmt.Errorf("lecture ZUGFeRD 1.0 %s: devise (BT-5) absente", sourceName)
	}
	typeCode := model.TypeCode(strings.TrimSpace(x.Header.TypeCode))
	kind := model.KindInvoice
	if typeCode.IsCreditNote() {
		kind = model.KindCreditNote
	}
	doc := model.NewDocument(kind)
	doc.ID = strings.TrimSpace(x.Header.ID)
	doc.TypeCode = typeCode
	doc.CurrencyCode = currency

	if d, err := x.Header.Issue.date(); err != nil {
		return nil, fmt.Errorf("lecture ZUGFeRD 1.0 %s: date d'émission: %w", sourceName, err)
	} else if d != nil {
		doc.IssueDate = *d
	}
	for _, n := range x.Header.Notes {
		if c := strings.TrimSpace(n.Content); c != "" {
			doc.Notes = append(doc.Notes, model.Note{Content: c})
		}
	}

	doc.BuyerReference = strings.TrimSpace(x.Tx.Agreement.BuyerRef)
	doc.Seller = mapParty(x.Tx.Agreement.Seller)
	doc.Buyer = mapParty(x.Tx.Agreement.Buyer)

	for i, tx := range x.Tx.Settlement.Tax {
		ts, err := mapTax(tx, currency)
		if err != nil {
			return nil, fmt.Errorf("lecture ZUGFeRD 1.0 %s: TVA #%d: %w", sourceName, i+1, err)
		}
		doc.TaxBreakdown = append(doc.TaxBreakdown, ts)
	}

	for i := range x.Tx.Lines {
		l, err := mapLine(&x.Tx.Lines[i], currency)
		if err != nil {
			return nil, fmt.Errorf("lecture ZUGFeRD 1.0 %s: ligne #%d: %w", sourceName, i+1, err)
		}
		doc.Lines = append(doc.Lines, l)
	}

	sum := x.Tx.Settlement.Sum
	var err error
	if doc.Totals.LineExtensionAmount, err = amt(sum.LineTotal, currency); err != nil {
		return nil, wrap(sourceName, "BT-106", err)
	}
	if doc.Totals.TaxExclusiveAmount, err = amt(sum.TaxBasisTotal, currency); err != nil {
		return nil, wrap(sourceName, "BT-109", err)
	}
	if doc.Totals.TaxAmount, err = amt(sum.TaxTotal, currency); err != nil {
		return nil, wrap(sourceName, "BT-110", err)
	}
	if doc.Totals.TaxInclusiveAmount, err = amt(sum.GrandTotal, currency); err != nil {
		return nil, wrap(sourceName, "BT-112", err)
	}
	if doc.Totals.DuePayableAmount, err = amt(sum.DuePayable, currency); err != nil {
		return nil, wrap(sourceName, "BT-115", err)
	}
	if v := strings.TrimSpace(sum.AllowTotal); v != "" {
		a, _ := amt(v, currency)
		doc.Totals.AllowanceTotal = &a
	}
	if v := strings.TrimSpace(sum.ChargeTotal); v != "" {
		a, _ := amt(v, currency)
		doc.Totals.ChargeTotal = &a
	}
	if v := strings.TrimSpace(sum.Prepaid); v != "" {
		a, _ := amt(v, currency)
		doc.Totals.PrepaidAmount = &a
	}

	doc.Provenance = model.NewProvenance(sourceName, "zugferd1", profileFromURN(x.Context.Guideline.ID))
	return doc, nil
}

func mapParty(p party) model.Party {
	out := model.Party{
		Name:    strings.TrimSpace(p.Name),
		LegalID: strings.TrimSpace(p.LegalOrg.ID),
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

func mapTax(tx headerTax, currency string) (model.TaxSubtotal, error) {
	basis, err := amt(tx.BasisAmount, currency)
	if err != nil {
		return model.TaxSubtotal{}, err
	}
	amount, err := amt(tx.CalculatedAmount, currency)
	if err != nil {
		return model.TaxSubtotal{}, err
	}
	rate, err := dec(tx.Percent)
	if err != nil {
		return model.TaxSubtotal{}, err
	}
	return model.TaxSubtotal{
		Category:      model.TaxCategoryCode(strings.TrimSpace(tx.CategoryCode)),
		Rate:          rate,
		TaxableAmount: basis,
		TaxAmount:     amount,
	}, nil
}

func mapLine(l *lineItem, currency string) (model.Line, error) {
	qty, err := dec(orZero(l.Delivery.Qty.Value))
	if err != nil {
		return model.Line{}, err
	}
	netAmount, err := amt(l.Settlement.Sum.LineTotal, currency)
	if err != nil {
		return model.Line{}, err
	}
	out := model.Line{
		ID:          strings.TrimSpace(l.Doc.LineID),
		Name:        strings.TrimSpace(l.Product.Name),
		Quantity:    qty,
		UnitCode:    model.UnitCode(strings.TrimSpace(l.Delivery.Qty.UnitCode)),
		NetAmount:   netAmount,
		TaxCategory: model.TaxCategoryCode(strings.TrimSpace(l.Settlement.Tax.CategoryCode)),
	}
	if p := strings.TrimSpace(l.Agreement.Net.ChargeAmount); p != "" {
		out.NetPrice, _ = amt(p, currency)
	}
	if r := strings.TrimSpace(l.Settlement.Tax.Percent); r != "" {
		rate, err := dec(r)
		if err == nil {
			out.TaxRate = &rate
		}
	}
	return out, nil
}

// --- helpers ---

func amt(s, currency string) (model.Amount, error) { return model.ParseAmount(orZero(s), currency) }
func dec(s string) (model.Decimal, error)          { return model.ParseDecimal(orZero(s)) }

func orZero(s string) string {
	if s = strings.TrimSpace(s); s == "" {
		return "0"
	}
	return s
}

func wrap(src, term string, err error) error {
	return fmt.Errorf("lecture ZUGFeRD 1.0 %s: %s: %w", src, term, err)
}

func profileFromURN(urn string) string {
	u := strings.ToLower(urn)
	switch {
	case strings.Contains(u, "extended"):
		return "extended"
	case strings.Contains(u, "comfort"):
		return "comfort"
	case strings.Contains(u, "basic"):
		return "basic"
	default:
		return "zugferd1"
	}
}
