package fatturapa

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

func mapToPivot(x *fattura, sourceName string) (*model.Document, error) {
	currency := trimmed(x.Body.DatiGenerali.Documento.Divisa)
	if currency == "" {
		currency = "EUR"
	}

	td := trimmed(x.Body.DatiGenerali.Documento.TipoDocumento)
	kind := model.KindInvoice
	typeCode := model.TypeCommercialInvoice
	if td == "TD04" { // nota di credito
		kind = model.KindCreditNote
		typeCode = model.TypeCreditNote
	}

	doc := model.NewDocument(kind)
	doc.ID = trimmed(x.Body.DatiGenerali.Documento.Numero)
	doc.TypeCode = typeCode
	doc.CurrencyCode = currency

	if v := trimmed(x.Body.DatiGenerali.Documento.Data); v != "" {
		d, err := model.ParseISO(v)
		if err != nil {
			return nil, fmt.Errorf("lecture FatturaPA %s: date (Data): %w", sourceName, err)
		}
		doc.IssueDate = d
	}

	doc.Seller = mapParty(x.Header.Cedente)
	doc.Buyer = mapParty(x.Header.Cessionario)

	for i, l := range x.Body.DatiBeniServizi.Linee {
		line, err := mapLine(l, currency)
		if err != nil {
			return nil, fmt.Errorf("lecture FatturaPA %s: ligne #%d: %w", sourceName, i+1, err)
		}
		doc.Lines = append(doc.Lines, line)
	}

	for i, r := range x.Body.DatiBeniServizi.Riepilogo {
		ts, err := mapRiepilogo(r, currency)
		if err != nil {
			return nil, fmt.Errorf("lecture FatturaPA %s: récapitulatif TVA #%d: %w", sourceName, i+1, err)
		}
		doc.TaxBreakdown = append(doc.TaxBreakdown, ts)
	}

	if err := mapPayment(doc, x); err != nil {
		return nil, fmt.Errorf("lecture FatturaPA %s: paiement: %w", sourceName, err)
	}

	if err := mapTotals(doc, x, currency); err != nil {
		return nil, fmt.Errorf("lecture FatturaPA %s: totaux: %w", sourceName, err)
	}

	doc.Provenance = model.NewProvenance(sourceName, "fatturapa", "")
	// FatturaPA porte son identifiant de spécification via l'attribut racine `versione`
	// (FPA12/FPR12), ou à défaut FormatoTrasmissione : c'est l'équivalent italien de BT-24,
	// tracé tel quel dans la provenance.
	if spec := strings.TrimSpace(x.Versione); spec != "" {
		doc.Provenance.SpecIdentifier = spec
	} else if spec := strings.TrimSpace(x.Header.FormatoTrasmissione); spec != "" {
		doc.Provenance.SpecIdentifier = spec
	}
	return doc, nil
}

func mapParty(a anagrafica) model.Party {
	name := trimmed(a.DatiAnagrafici.Anagrafica.Denominazione)
	if name == "" {
		name = strings.TrimSpace(trimmed(a.DatiAnagrafici.Anagrafica.Nome) + " " + trimmed(a.DatiAnagrafici.Anagrafica.Cognome))
	}
	p := model.Party{
		Name:    name,
		TaxID:   trimmed(a.DatiAnagrafici.CodiceFiscale),
		Address: mapAddress(a),
	}
	if code := trimmed(a.DatiAnagrafici.IdFiscaleIVA.IdCodice); code != "" {
		p.VATID = trimmed(a.DatiAnagrafici.IdFiscaleIVA.IdPaese) + code
	}
	return p
}

func mapAddress(a anagrafica) model.Address {
	return model.Address{
		Line1:       trimmed(a.Sede.Indirizzo),
		City:        trimmed(a.Sede.Comune),
		PostalCode:  trimmed(a.Sede.CAP),
		CountryCode: trimmed(a.Sede.Nazione),
	}
}

func mapLine(l dettaglioLinea, currency string) (model.Line, error) {
	out := model.Line{
		ID:          trimmed(l.NumeroLinea),
		Name:        trimmed(l.Descrizione),
		UnitCode:    unitOrDefault(l.UnitaMisura),
		TaxCategory: category(l.AliquotaIVA, l.Natura),
	}

	qty := trimmed(l.Quantita)
	if qty == "" {
		qty = "1" // les prestations de service omettent souvent la quantité
	}
	q, err := model.ParseDecimal(qty)
	if err != nil {
		return model.Line{}, fmt.Errorf("quantité: %w", err)
	}
	out.Quantity = q

	if out.NetPrice, err = model.ParseAmount(orZero(l.PrezzoUnitario), currency); err != nil {
		return model.Line{}, fmt.Errorf("prix unitaire: %w", err)
	}
	if out.NetAmount, err = model.ParseAmount(orZero(l.PrezzoTotale), currency); err != nil {
		return model.Line{}, fmt.Errorf("montant de ligne: %w", err)
	}
	if r := trimmed(l.AliquotaIVA); r != "" {
		rate, err := model.ParseDecimal(r)
		if err != nil {
			return model.Line{}, fmt.Errorf("taux TVA: %w", err)
		}
		out.TaxRate = &rate
	}
	return out, nil
}

func mapRiepilogo(r datiRiepilogo, currency string) (model.TaxSubtotal, error) {
	rate, err := model.ParseDecimal(orZero(r.AliquotaIVA))
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("taux: %w", err)
	}
	base, err := model.ParseAmount(orZero(r.ImponibileImporto), currency)
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("base imposable: %w", err)
	}
	tax, err := model.ParseAmount(orZero(r.Imposta), currency)
	if err != nil {
		return model.TaxSubtotal{}, fmt.Errorf("montant TVA: %w", err)
	}
	return model.TaxSubtotal{
		Category:      category(r.AliquotaIVA, r.Natura),
		Rate:          rate,
		TaxableAmount: base,
		TaxAmount:     tax,
	}, nil
}

func mapTotals(doc *model.Document, x *fattura, currency string) error {
	lineTotal := zero(currency)
	for _, l := range doc.Lines {
		var err error
		if lineTotal, err = lineTotal.Add(l.NetAmount); err != nil {
			return err
		}
	}
	taxable := zero(currency)
	tax := zero(currency)
	for _, ts := range doc.TaxBreakdown {
		var err error
		if taxable, err = taxable.Add(ts.TaxableAmount); err != nil {
			return err
		}
		if tax, err = tax.Add(ts.TaxAmount); err != nil {
			return err
		}
	}
	doc.Totals.LineExtensionAmount = lineTotal.Rescale(2)
	doc.Totals.TaxExclusiveAmount = taxable.Rescale(2)
	doc.Totals.TaxAmount = tax.Rescale(2)

	inclusive := trimmed(x.Body.DatiGenerali.Documento.ImportoTotale)
	if inclusive != "" {
		a, err := model.ParseAmount(inclusive, currency)
		if err != nil {
			return fmt.Errorf("total document: %w", err)
		}
		doc.Totals.TaxInclusiveAmount = a
	} else {
		sum, err := taxable.Add(tax)
		if err != nil {
			return err
		}
		doc.Totals.TaxInclusiveAmount = sum.Rescale(2)
	}
	doc.Totals.DuePayableAmount = doc.Totals.TaxInclusiveAmount
	return nil
}

// mapPayment remonte l'échéance (DataScadenzaPagamento → BT-9) et l'IBAN de règlement.
func mapPayment(doc *model.Document, x *fattura) error {
	for _, dp := range x.Body.DatiPagamento {
		for _, det := range dp.Dettaglio {
			if v := trimmed(det.DataScadenza); v != "" && doc.DueDate == nil {
				d, err := model.ParseISO(v)
				if err != nil {
					return fmt.Errorf("échéance (DataScadenzaPagamento): %w", err)
				}
				doc.DueDate = &d
			}
			if iban := trimmed(det.IBAN); iban != "" {
				if doc.PaymentInstructions == nil {
					doc.PaymentInstructions = &model.PaymentInstructions{MeansCode: model.PayCredit}
				}
				doc.PaymentInstructions.CreditTransfers = append(
					doc.PaymentInstructions.CreditTransfers, model.CreditTransfer{IBAN: iban})
			}
		}
	}
	return nil
}

// category déduit la catégorie de TVA EN 16931 à partir du taux et de la Natura FatturaPA.
func category(aliquota, natura string) model.TaxCategoryCode {
	if r := trimmed(aliquota); r != "" && r != "0" && r != "0.00" && r != "0.0" {
		return model.TaxStandard // S : taux positif
	}
	switch n := strings.ToUpper(trimmed(natura)); {
	case strings.HasPrefix(n, "N6"):
		return model.TaxReverseCharge // autoliquidation
	case strings.HasPrefix(n, "N4"):
		return model.TaxExempt // esenti
	case strings.HasPrefix(n, "N3"):
		return model.TaxIntraCommunity // non imponibili (approximation)
	default:
		return model.TaxOutsideScope // N1/N2/N5/N7 (approximation)
	}
}

func unitOrDefault(u string) model.UnitCode {
	if v := trimmed(u); v != "" {
		return model.UnitCode(v)
	}
	return model.UnitPiece
}

func zero(currency string) model.Amount {
	a, _ := model.ParseAmount("0", currency)
	return a
}

func orZero(s string) string {
	if v := trimmed(s); v != "" {
		return v
	}
	return "0"
}
