// Package anonymize remplace les données personnelles d'une facture par des valeurs
// synthétiques cohérentes, de façon déterministe (§11.2 kanjo anonymize). Point de conformité
// RGPD majeur : permet de constituer un corpus de régression à partir de cas clients réels
// sans exporter aucune donnée personnelle.
package anonymize

import (
	"fmt"
	"hash/fnv"
	"math/rand"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// Options paramètre l'anonymisation.
type Options struct {
	Seed        string          // graine pour un remplacement déterministe
	KeepAmounts bool            // conserver les montants d'origine (par défaut ils sont remplacés)
	keep        map[string]bool // ensemble des BT à préserver
}

// SetKeep définit les termes à préserver (ex. "amounts").
func (o *Options) SetKeep(terms []string) {
	o.keep = map[string]bool{}
	for _, t := range terms {
		o.keep[t] = true
	}
	if o.keep["amounts"] {
		o.KeepAmounts = true
	}
}

var sellerPool = []string{"SAS Alpha", "SARL Beta", "EURL Gamma", "SA Delta", "SAS Epsilon", "SARL Zeta"}
var buyerPool = []string{"Société Un", "Société Deux", "Société Trois", "Société Quatre", "Société Cinq"}
var streetPool = []string{"rue des Lilas", "avenue Centrale", "impasse du Test", "boulevard Nord", "chemin des Prés"}
var cityPool = []string{"Villeneuve", "Bourg", "Sainte-Foy", "Montclair", "Rivedoux"}
var firstNames = []string{"Camille", "Dominique", "Claude", "Alix", "Sacha", "Maxime"}
var lastNames = []string{"Durand", "Petit", "Moreau", "Girard", "Lambert", "Rousseau"}

// Anonymize remplace en place les données personnelles du document.
func Anonymize(doc *model.Document, opts Options) {
	rng := rand.New(rand.NewSource(seedInt(opts.Seed)))

	doc.ID = fmt.Sprintf("ANON-%06d", rng.Intn(1000000))
	doc.BuyerReference = ""
	doc.PurchaseOrderRef = ""

	anonParty(&doc.Seller, sellerPool, rng)
	anonParty(&doc.Buyer, buyerPool, rng)
	if doc.Payee != nil {
		anonParty(doc.Payee, sellerPool, rng)
	}
	if doc.DeliverTo != nil {
		doc.DeliverTo.Name = pick(rng, buyerPool)
		doc.DeliverTo.Address = anonAddress(doc.DeliverTo.Address, rng)
	}
	if doc.PaymentInstructions != nil {
		for i := range doc.PaymentInstructions.CreditTransfers {
			doc.PaymentInstructions.CreditTransfers[i].IBAN = genIBAN(rng)
			doc.PaymentInstructions.CreditTransfers[i].AccountName = ""
			doc.PaymentInstructions.CreditTransfers[i].BIC = ""
		}
		doc.PaymentInstructions.RemittanceInfo = ""
	}

	// Neutraliser les extensions FR susceptibles de porter des identifiants réels.
	if doc.Extensions.FR != nil {
		doc.Extensions.FR.SellerSIREN = doc.Seller.LegalID
		doc.Extensions.FR.BuyerSIREN = doc.Buyer.LegalID
	}

	if !opts.KeepAmounts {
		anonAmounts(doc, rng)
	}
}

func anonParty(p *model.Party, pool []string, rng *rand.Rand) {
	if p.Name != "" {
		p.Name = pick(rng, pool)
	}
	if p.TradingName != "" {
		p.TradingName = p.Name
	}
	if p.LegalID != "" {
		p.LegalID = fmt.Sprintf("%09d", 100000000+rng.Intn(899999999))
	}
	if p.VATID != "" {
		p.VATID = "FR" + fmt.Sprintf("%02d", rng.Intn(100)) + orLegal(p.LegalID, rng)
	}
	p.TaxID = ""
	p.Address = anonAddress(p.Address, rng)
	if p.Contact != nil {
		name := pick(rng, firstNames) + " " + pick(rng, lastNames)
		p.Contact.Name = name
		if p.Contact.Email != "" {
			p.Contact.Email = "contact@example.test"
		}
		if p.Contact.Phone != "" {
			p.Contact.Phone = "+33100000000"
		}
	}
	if p.ElectronicAddr != nil {
		p.ElectronicAddr.Value = "0000000000000"
	}
}

func anonAddress(a model.Address, rng *rand.Rand) model.Address {
	if a.Empty() {
		return a
	}
	return model.Address{
		Line1:       fmt.Sprintf("%d %s", 1+rng.Intn(200), pick(rng, streetPool)),
		City:        pick(rng, cityPool),
		PostalCode:  fmt.Sprintf("%05d", 10000+rng.Intn(89999)),
		CountryCode: a.CountryCode, // le pays est structurel, préservé
	}
}

// anonAmounts remplace les prix par des valeurs synthétiques et recalcule des totaux cohérents.
func anonAmounts(doc *model.Document, rng *rand.Rand) {
	for i := range doc.Lines {
		l := &doc.Lines[i]
		qty := int64(1)
		if l.Quantity.Unscaled > 0 && l.Quantity.Scale == 0 {
			qty = l.Quantity.Unscaled
		} else {
			l.Quantity = model.DecimalFromInt(1)
		}
		l.NetPrice = model.NewAmount(int64(1000+rng.Intn(50000)), 2, doc.CurrencyCode)
		l.GrossPrice = nil
		l.PriceDiscount = nil
		l.NetAmount = l.NetPrice.MulQuantity(model.DecimalFromInt(qty), 2)
	}
	recomputeTotals(doc)
}

// recomputeTotals reconstruit la ventilation de TVA et les totaux à partir des lignes.
func recomputeTotals(doc *model.Document) {
	type key struct {
		cat  model.TaxCategoryCode
		rate string
	}
	var order []key
	bases := map[key]model.Amount{}
	rates := map[key]model.Decimal{}
	for _, l := range doc.Lines {
		r := model.MustParseDecimal("0")
		if l.TaxRate != nil {
			r = *l.TaxRate
		}
		k := key{l.TaxCategory, r.String()}
		if _, ok := bases[k]; !ok {
			order = append(order, k)
			bases[k] = model.ZeroAmount(doc.CurrencyCode)
			rates[k] = r
		}
		bases[k], _ = bases[k].Add(l.NetAmount)
	}
	lineTotal := model.ZeroAmount(doc.CurrencyCode)
	taxTotal := model.ZeroAmount(doc.CurrencyCode)
	doc.TaxBreakdown = nil
	for _, k := range order {
		ts := model.TaxSubtotal{Category: k.cat, Rate: rates[k], TaxableAmount: bases[k].Rescale(2)}
		ts.TaxAmount = ts.ComputeTaxAmount()
		doc.TaxBreakdown = append(doc.TaxBreakdown, ts)
		lineTotal, _ = lineTotal.Add(ts.TaxableAmount)
		taxTotal, _ = taxTotal.Add(ts.TaxAmount)
	}
	lineTotal = lineTotal.Rescale(2)
	taxTotal = taxTotal.Rescale(2)
	ttc, _ := lineTotal.Add(taxTotal)
	doc.Totals = model.Totals{
		LineExtensionAmount: lineTotal,
		TaxExclusiveAmount:  lineTotal,
		TaxAmount:           taxTotal,
		TaxInclusiveAmount:  ttc.Rescale(2),
		DuePayableAmount:    ttc.Rescale(2),
	}
}

func orLegal(legal string, rng *rand.Rand) string {
	if len(legal) == 9 {
		return legal
	}
	return fmt.Sprintf("%09d", 100000000+rng.Intn(899999999))
}

func seedInt(seed string) int64 {
	if seed == "" {
		return 1
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

func pick(rng *rand.Rand, pool []string) string { return pool[rng.Intn(len(pool))] }

// genIBAN produit un IBAN français synthétique satisfaisant le contrôle modulo 97.
func genIBAN(rng *rand.Rand) string {
	bban := make([]byte, 23)
	for i := range bban {
		bban[i] = byte('0' + rng.Intn(10))
	}
	// Calcul des deux chiffres de contrôle : réarranger "FR00"+bban → nombres, mod 97.
	rearr := string(bban) + "1527" + "00" // F=15,R=27 ; "00" = check provisoire
	rem := 0
	for i := 0; i < len(rearr); i++ {
		rem = (rem*10 + int(rearr[i]-'0')) % 97
	}
	check := 98 - rem
	return fmt.Sprintf("FR%02d%s", check, string(bban))
}
