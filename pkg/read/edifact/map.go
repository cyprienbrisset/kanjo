package edifact

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// mapToPivot traduit les segments d'un message INVOIC en document pivot. La couverture vise les
// segments porteurs de sens métier (BGM, DTM, NAD, RFF, CUX, LIN/IMD/QTY/MOA/PRI, TAX, totaux MOA).
// Fidèle au §17.7 : rien n'est inventé ; les segments non modélisés sont simplement ignorés (une
// version ultérieure produira un rapport de perte).
func mapToPivot(segs []segment, profile, sourceName string) (*model.Document, error) {
	// Type de document (BGM/1001, aligné sur UNTDID 1001 = BT-3) → sens facture / avoir.
	typeCode := model.TypeCommercialInvoice
	for _, s := range segs {
		if s.tag == "BGM" {
			if c := strings.TrimSpace(s.comp(0, 0)); c != "" {
				typeCode = model.TypeCode(c)
			}
			break
		}
	}
	kind := model.KindInvoice
	if typeCode.IsCreditNote() {
		kind = model.KindCreditNote
	}

	// Devise (CUX) : nécessaire pour typer les montants ; pré-balayage, jamais devinée.
	currency := ""
	for _, s := range segs {
		if s.tag == "CUX" {
			if c := strings.TrimSpace(s.comp(0, 1)); c != "" {
				currency = c
				break
			}
		}
	}

	doc := model.NewDocument(kind)
	doc.TypeCode = typeCode
	doc.CurrencyCode = currency

	var lastParty *model.Party // pour rattacher les RFF+VA suivants
	var line *model.Line       // ligne courante (groupe LIN)

	flushLine := func() {
		if line != nil {
			doc.Lines = append(doc.Lines, *line)
			line = nil
		}
	}

	for _, s := range segs {
		switch s.tag {
		case "BGM":
			doc.ID = strings.TrimSpace(s.comp(1, 0))

		case "DTM":
			if s.comp(0, 0) == "137" { // date du document (BT-2)
				if d, ok := parseEdifactDate(s.comp(0, 1)); ok {
					doc.IssueDate = d
				}
			}

		case "NAD":
			p := parseNAD(s)
			switch s.comp(0, 0) {
			case "SU", "SE", "FR": // fournisseur / vendeur / expéditeur du message
				doc.Seller = p
				lastParty = &doc.Seller
			case "BY", "BT", "IV": // acheteur / à facturer / destinataire de facture
				doc.Buyer = p
				lastParty = &doc.Buyer
			default:
				lastParty = nil // ST (livraison), etc. — non modélisé ici
			}

		case "RFF":
			if lastParty != nil && s.comp(0, 0) == "VA" { // identifiant TVA (BT-31/BT-48)
				if v := strings.TrimSpace(s.comp(0, 1)); v != "" {
					lastParty.VATID = v
				}
			}

		case "LIN":
			flushLine()
			l := model.Line{ID: fmt.Sprint(len(doc.Lines) + 1)}
			if item := strings.TrimSpace(s.comp(2, 0)); item != "" {
				l.Name = item // provisoire ; IMD l'écrase si présent
			}
			line = &l

		case "IMD":
			if line != nil {
				if desc := imdDescription(s); desc != "" {
					line.Name = desc
				}
			}

		case "QTY":
			if line != nil {
				if q, err := model.ParseDecimal(strings.TrimSpace(s.comp(0, 1))); err == nil {
					line.Quantity = q
				}
				if u := strings.TrimSpace(s.comp(0, 2)); u != "" {
					line.UnitCode = model.UnitCode(u)
				}
			}

		case "PRI":
			if line != nil && (s.comp(0, 0) == "AAA" || s.comp(0, 0) == "NET" || s.comp(0, 0) == "") {
				if a, err := model.ParseAmount(strings.TrimSpace(s.comp(0, 1)), currency); err == nil {
					line.NetPrice = a
				}
			}

		case "MOA":
			applyMOA(s, currency, doc, line)

		case "TAX":
			applyTAX(s, doc, line)

		case "UNS": // séparateur en-tête/sommaire : la ligne courante est close
			flushLine()
		}
	}
	flushLine()

	doc.Provenance = model.NewProvenance(sourceName, "edifact", profile)
	doc.Provenance.SpecIdentifier = profile // ex. « INVOIC:D:97A:UN »
	return doc, nil
}

// parseNAD extrait un tiers d'un segment NAD. Le nom provient du champ « nom du tiers » (C080) ;
// à défaut — cas fréquent des messages réels — du premier composant de l'identifiant (C082).
func parseNAD(s segment) model.Party {
	p := model.Party{}
	if name := joinComps(s.element(3)); name != "" {
		p.Name = name
	} else if id := strings.TrimSpace(s.comp(1, 0)); id != "" {
		p.Name = id
	}
	p.Address = model.Address{
		Line1:       joinComps(s.element(4)),
		City:        strings.TrimSpace(s.comp(5, 0)),
		PostalCode:  strings.TrimSpace(s.comp(7, 0)),
		CountryCode: strings.TrimSpace(s.comp(8, 0)),
	}
	return p
}

// imdDescription assemble la désignation d'article d'un segment IMD (C273 : composants 4 et 5).
func imdDescription(s segment) string {
	el := s.element(2)
	var parts []string
	for i := 3; i < len(el); i++ {
		if v := strings.TrimSpace(el[i]); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}

// applyMOA route un montant selon son qualifiant (C516/5025), au niveau ligne ou document.
func applyMOA(s segment, currency string, doc *model.Document, line *model.Line) {
	qual := strings.TrimSpace(s.comp(0, 0))
	amt, err := model.ParseAmount(strings.TrimSpace(s.comp(0, 1)), currency)
	if err != nil {
		return
	}
	switch qual {
	case "203": // montant net de ligne (BT-131)
		if line != nil {
			line.NetAmount = amt
		}
	case "79", "38": // total des montants de ligne (BT-106)
		doc.Totals.LineExtensionAmount = amt
		if doc.Totals.TaxExclusiveAmount.IsZero() {
			doc.Totals.TaxExclusiveAmount = amt
		}
	case "125": // montant imposable (BT-116 / BT-109)
		doc.Totals.TaxExclusiveAmount = amt
	case "124", "176": // montant de TVA (BT-110)
		doc.Totals.TaxAmount = amt
	case "77", "128": // total TTC (BT-112)
		doc.Totals.TaxInclusiveAmount = amt
	case "9", "86": // net à payer (BT-115)
		doc.Totals.DuePayableAmount = amt
	}
}

// applyTAX enregistre une ventilation de TVA (segment TAX). Le taux est en C243/5278, la
// catégorie en 5305 ; à défaut de catégorie explicite on déduit taux normal / taux zéro.
func applyTAX(s segment, doc *model.Document, line *model.Line) {
	if strings.TrimSpace(s.comp(0, 0)) != "7" { // 7 = taxe/impôt/redevance
		return
	}
	rateStr := strings.TrimSpace(s.comp(4, 3))
	rate, err := model.ParseDecimal(rateStr)
	if err != nil {
		return
	}
	cat := model.TaxCategoryCode(strings.TrimSpace(s.comp(5, 0)))
	if cat == "" {
		if rate.IsZero() {
			cat = model.TaxZeroRated
		} else {
			cat = model.TaxStandard
		}
	}
	if line != nil {
		line.TaxCategory = cat
		r := rate
		line.TaxRate = &r
	}
	// Éviter les doublons de ventilation pour une même (catégorie, taux).
	for _, ts := range doc.TaxBreakdown {
		if ts.Category == cat && ts.Rate.String() == rate.String() {
			return
		}
	}
	doc.TaxBreakdown = append(doc.TaxBreakdown, model.TaxSubtotal{Category: cat, Rate: rate})
}

// joinComps assemble les composants non vides d'un élément (nom/adresse répartis en composants).
func joinComps(comps []string) string {
	var parts []string
	for _, c := range comps {
		if v := strings.TrimSpace(c); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}

// parseEdifactDate décode une date EDIFACT (DTM) au format CCYYMMDD (102) ou CCYYMMDDHHMM (203) :
// on ne conserve que la partie calendaire, le pivot ne portant pas d'heure sur une date de facture.
func parseEdifactDate(v string) (model.Date, bool) {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, v)
	if len(digits) < 8 {
		return model.Date{}, false
	}
	d, err := model.ParseISO(digits[0:4] + "-" + digits[4:6] + "-" + digits[6:8])
	if err != nil {
		return model.Date{}, false
	}
	return d, true
}
