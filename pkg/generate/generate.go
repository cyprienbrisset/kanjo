// Package generate produit des factures synthétiques cohérentes, pour constituer un corpus
// de test sans donnée réelle (§11.2 kanjo generate). Les documents sont arithmétiquement
// conformes (toutes les BR-CO passent) sauf si Invalid est demandé.
package generate

import (
	"fmt"
	"math/rand"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// Scenario décrit un cas de facturation à générer.
type Scenario string

const (
	ScenarioSimple             Scenario = "simple"
	ScenarioMultiTVA           Scenario = "multi-tva"
	ScenarioAvoir              Scenario = "avoir"
	ScenarioAutoliquidation    Scenario = "autoliquidation"
	ScenarioIntracommunautaire Scenario = "intracommunautaire"
	ScenarioAcompte            Scenario = "acompte"
)

// Options paramètre la génération.
type Options struct {
	Scenario Scenario
	Seed     int64
	Invalid  bool // produire un document volontairement non conforme
}

var sellers = []string{"SAS Aurore", "SARL Bellefeuille", "SAS Comptoir du Rhône", "EURL Delorme", "SA Meunier"}
var buyers = []string{"Société Cliente", "SAS Petit", "SARL Nord Services", "SA Lavande", "EURL Ozero"}
var cities = []string{"Paris", "Lyon", "Marseille", "Bordeaux", "Lille"}
var products = []string{"Prestation de conseil", "Licence annuelle", "Développement logiciel", "Hébergement", "Formation", "Maintenance", "Audit de sécurité"}

// Generate produit une facture pour l'index donné (0-based). L'aléa est déterministe pour un
// couple (Seed, index) donné, ce qui rend les corpus reproductibles.
func Generate(index int, opts Options) (*model.Document, error) {
	if opts.Scenario == "" {
		opts.Scenario = ScenarioSimple
	}
	rng := rand.New(rand.NewSource(opts.Seed + int64(index)*1_000_003))

	kind := model.KindInvoice
	typeCode := model.TypeCommercialInvoice
	if opts.Scenario == ScenarioAvoir {
		kind = model.KindCreditNote
		typeCode = model.TypeCreditNote
	}
	if opts.Scenario == ScenarioAcompte {
		typeCode = model.TypePrepaymentInvoice
	}

	doc := model.NewDocument(kind)
	doc.ID = fmt.Sprintf("F2026-%04d", index+1)
	doc.IssueDate, _ = model.NewDate(2026, 9, 1+rng.Intn(28))
	doc.TypeCode = typeCode
	doc.CurrencyCode = "EUR"
	siren := fmt.Sprintf("%09d", 100000000+rng.Intn(899999999))
	doc.Seller = model.Party{
		Name: pick(rng, sellers), LegalID: siren, LegalIDScheme: "0002",
		VATID:   "FR" + fmt.Sprintf("%02d", rng.Intn(100)) + siren,
		Address: model.Address{Line1: "1 rue du Test", PostalCode: "69000", City: pick(rng, cities), CountryCode: "FR"},
	}
	doc.Buyer = model.Party{
		Name:    pick(rng, buyers),
		Address: model.Address{Line1: "2 avenue Client", PostalCode: "75000", City: pick(rng, cities), CountryCode: "FR"},
	}
	// Autoliquidation et livraison intracommunautaire exigent l'identifiant TVA de l'acheteur
	// (BR-AE-02 / BR-IC-02) ; l'intracommunautaire place l'acheteur dans un autre État membre.
	switch opts.Scenario {
	case ScenarioAutoliquidation:
		doc.Buyer.VATID = "FR" + fmt.Sprintf("%02d", rng.Intn(100)) + fmt.Sprintf("%09d", 200000000+rng.Intn(799999999))
	case ScenarioIntracommunautaire:
		doc.Buyer.Address.CountryCode = "DE"
		doc.Buyer.VATID = "DE" + fmt.Sprintf("%09d", 100000000+rng.Intn(899999999))
		// Une livraison intracommunautaire exige une date et un pays de livraison (BR-IC-11/12).
		delivery, _ := model.NewDate(2026, 9, 1+rng.Intn(27))
		doc.DeliverTo = &model.DeliveryInfo{
			Name:         doc.Buyer.Name,
			Address:      model.Address{City: "Berlin", PostalCode: "10115", CountryCode: "DE"},
			DeliveryDate: &delivery,
		}
	}

	cat, rate := scenarioTax(opts.Scenario)
	nLines := 1 + rng.Intn(3)
	for i := 0; i < nLines; i++ {
		qty := int64(1 + rng.Intn(5))
		price := model.NewAmount(int64(1000+rng.Intn(50000)), 2, "EUR") // 10,00 à 510,00
		net := price.MulQuantity(model.DecimalFromInt(qty), 2)
		r := rate
		line := model.Line{
			ID: fmt.Sprintf("%d", i+1), Name: pick(rng, products),
			Quantity: model.DecimalFromInt(qty), UnitCode: model.UnitPiece,
			NetPrice: price, TaxCategory: cat, NetAmount: net,
		}
		if cat == model.TaxStandard {
			line.TaxRate = &r
		} else {
			zero := model.MustParseDecimal("0")
			line.TaxRate = &zero
		}
		doc.Lines = append(doc.Lines, line)
	}

	if opts.Scenario == ScenarioMultiTVA && len(doc.Lines) > 1 {
		// Deuxième catégorie : taux réduit 10 %.
		reduced := model.MustParseDecimal("10")
		doc.Lines[len(doc.Lines)-1].TaxRate = &reduced
	}

	doc.PaymentTerms = "Paiement à 30 jours à réception de facture."
	applyExemptionNote(doc, opts.Scenario)
	computeTotals(doc)

	if opts.Scenario == ScenarioAcompte {
		prepaid := doc.Totals.TaxInclusiveAmount.MulQuantity(model.MustParseDecimal("0.30"), 2)
		doc.Totals.PrepaidAmount = &prepaid
		due, _ := doc.Totals.ComputeDuePayable("EUR")
		doc.Totals.DuePayableAmount = due
	}

	if opts.Invalid {
		// Casser volontairement le total TTC (BR-CO-15) pour tester les validateurs.
		doc.Totals.TaxInclusiveAmount, _ = doc.Totals.TaxInclusiveAmount.Add(model.MustParseAmount("10.00", "EUR"))
	}
	doc.Provenance = model.NewProvenance("", "generated", "en16931")
	doc.Provenance.SpecIdentifier = "urn:cen.eu:en16931:2017"
	return doc, nil
}

func scenarioTax(s Scenario) (model.TaxCategoryCode, model.Decimal) {
	switch s {
	case ScenarioAutoliquidation:
		return model.TaxReverseCharge, model.MustParseDecimal("0")
	case ScenarioIntracommunautaire:
		return model.TaxIntraCommunity, model.MustParseDecimal("0")
	default:
		return model.TaxStandard, model.MustParseDecimal("20")
	}
}

func applyExemptionNote(doc *model.Document, s Scenario) {
	switch s {
	case ScenarioAutoliquidation:
		doc.Notes = append(doc.Notes, model.Note{Content: "Autoliquidation — TVA due par le preneur (art. 283-2 du CGI)."})
	case ScenarioIntracommunautaire:
		doc.Notes = append(doc.Notes, model.Note{Content: "Exonération TVA, art. 262 ter I du CGI (livraison intracommunautaire)."})
	}
}

// computeTotals calcule une ventilation de TVA groupée et des totaux cohérents.
func computeTotals(doc *model.Document) {
	type key struct {
		cat  model.TaxCategoryCode
		rate string
	}
	order := []key{}
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
			bases[k] = model.ZeroAmount("EUR")
			rates[k] = r
		}
		bases[k], _ = bases[k].Add(l.NetAmount)
	}

	lineTotal := model.ZeroAmount("EUR")
	taxTotal := model.ZeroAmount("EUR")
	doc.TaxBreakdown = nil
	for _, k := range order {
		ts := model.TaxSubtotal{Category: k.cat, Rate: rates[k], TaxableAmount: bases[k].Rescale(2)}
		ts.TaxAmount = ts.ComputeTaxAmount()
		if reason := exemptionReason(k.cat); reason != "" {
			ts.ExemptionReason = reason
		}
		doc.TaxBreakdown = append(doc.TaxBreakdown, ts)
		lineTotal, _ = lineTotal.Add(ts.TaxableAmount)
		taxTotal, _ = taxTotal.Add(ts.TaxAmount)
	}
	lineTotal = lineTotal.Rescale(2)
	taxTotal = taxTotal.Rescale(2)
	ttc, _ := lineTotal.Add(taxTotal)
	ttc = ttc.Rescale(2)

	doc.Totals = model.Totals{
		LineExtensionAmount: lineTotal,
		TaxExclusiveAmount:  lineTotal,
		TaxAmount:           taxTotal,
		TaxInclusiveAmount:  ttc,
		DuePayableAmount:    ttc,
	}
}

// exemptionReason renvoie un motif d'exonération pour les catégories de TVA qui l'exigent.
func exemptionReason(cat model.TaxCategoryCode) string {
	switch cat {
	case model.TaxReverseCharge:
		return "Autoliquidation — art. 283-2 du CGI."
	case model.TaxIntraCommunity:
		return "Exonération TVA — art. 262 ter I du CGI."
	case model.TaxExport:
		return "Exportation hors UE — art. 262 I du CGI."
	case model.TaxExempt:
		return "Opération exonérée de TVA."
	case model.TaxOutsideScope:
		return "Opération hors champ de la TVA."
	default:
		return ""
	}
}

func pick(rng *rand.Rand, pool []string) string { return pool[rng.Intn(len(pool))] }
