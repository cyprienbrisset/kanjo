package model

// Ce fichier définit les types de codes nommés du pivot (§5.2, règle 4). Chaque type
// dispose de Valid() et Label(lang). Les listes de valeurs sont ici restreintes à ce que
// le socle (Factur-X/UBL/CII, profil EN 16931) exige ; les listes de codes officielles
// complètes seront générées dans pkg/model/codes/ par `go generate` (L2).

// TypeCode — BT-3, code type de document (UNTDID 1001).
type TypeCode string

const (
	TypeCommercialInvoice  TypeCode = "380" // facture commerciale
	TypeCreditNote         TypeCode = "381" // avoir
	TypeCorrectedInvoice   TypeCode = "384" // facture rectificative
	TypePrepaymentInvoice  TypeCode = "386" // facture d'acompte
	TypeSelfBilledInvoice  TypeCode = "389" // autofacturation
	TypeInvoiceInformation TypeCode = "751" // informations de facture (comptabilisation, extensions FR)
)

var typeCodeLabels = map[TypeCode]map[Lang]string{
	TypeCommercialInvoice:  {LangFR: "Facture commerciale", LangEN: "Commercial invoice"},
	TypeCreditNote:         {LangFR: "Avoir", LangEN: "Credit note"},
	TypeCorrectedInvoice:   {LangFR: "Facture rectificative", LangEN: "Corrected invoice"},
	TypePrepaymentInvoice:  {LangFR: "Facture d'acompte", LangEN: "Prepayment invoice"},
	TypeSelfBilledInvoice:  {LangFR: "Autofacturation", LangEN: "Self-billed invoice"},
	TypeInvoiceInformation: {LangFR: "Informations de facture", LangEN: "Invoice information"},
}

// Valid indique si le code type est reconnu par le socle.
func (c TypeCode) Valid() bool { _, ok := typeCodeLabels[c]; return ok }

// Label renvoie le libellé dans la langue demandée (repli sur FR puis sur le code brut).
func (c TypeCode) Label(l Lang) string { return labelOf(typeCodeLabels[c], l, string(c)) }

// IsCreditNote indique si le code désigne un avoir.
func (c TypeCode) IsCreditNote() bool { return c == TypeCreditNote }

// TaxCategoryCode — BT-95/102/118/151, catégorie de TVA (UNTDID 5305).
type TaxCategoryCode string

const (
	TaxStandard         TaxCategoryCode = "S"  // taux normal
	TaxZeroRated        TaxCategoryCode = "Z"  // taux zéro
	TaxExempt           TaxCategoryCode = "E"  // exonéré
	TaxReverseCharge    TaxCategoryCode = "AE" // autoliquidation
	TaxIntraCommunity   TaxCategoryCode = "K"  // livraison intracommunautaire
	TaxExport           TaxCategoryCode = "G"  // export hors UE
	TaxOutsideScope     TaxCategoryCode = "O"  // hors champ de la TVA
	TaxCanaryIGIC       TaxCategoryCode = "L"  // IGIC (Canaries)
	TaxCeutaMelillaIPSI TaxCategoryCode = "M"  // IPSI (Ceuta/Melilla)
)

var taxCategoryLabels = map[TaxCategoryCode]map[Lang]string{
	TaxStandard:       {LangFR: "Taux normal", LangEN: "Standard rate"},
	TaxZeroRated:      {LangFR: "Taux zéro", LangEN: "Zero rated"},
	TaxExempt:         {LangFR: "Exonéré de TVA", LangEN: "Exempt from tax"},
	TaxReverseCharge:  {LangFR: "Autoliquidation", LangEN: "Reverse charge"},
	TaxIntraCommunity: {LangFR: "Livraison intracommunautaire", LangEN: "Intra-community supply"},
	TaxExport:         {LangFR: "Export hors UE", LangEN: "Export outside EU"},
	TaxOutsideScope:   {LangFR: "Hors champ TVA", LangEN: "Outside scope of tax"},
}

// Valid indique si la catégorie de TVA est reconnue.
func (c TaxCategoryCode) Valid() bool {
	switch c {
	case TaxStandard, TaxZeroRated, TaxExempt, TaxReverseCharge, TaxIntraCommunity,
		TaxExport, TaxOutsideScope, TaxCanaryIGIC, TaxCeutaMelillaIPSI:
		return true
	default:
		return false
	}
}

// Label renvoie le libellé de la catégorie de TVA.
func (c TaxCategoryCode) Label(l Lang) string { return labelOf(taxCategoryLabels[c], l, string(c)) }

// RequiresRate indique si la catégorie impose un taux strictement positif (S) ou nul (autres).
func (c TaxCategoryCode) RequiresRate() bool { return c == TaxStandard }

// PaymentMeansCode — BT-81, moyen de paiement (UNTDID 4461).
type PaymentMeansCode string

const (
	PayCredit    PaymentMeansCode = "30" // virement
	PayDirectDeb PaymentMeansCode = "49" // prélèvement
	PayCard      PaymentMeansCode = "48" // carte
	PayCheque    PaymentMeansCode = "20" // chèque
	PayCash      PaymentMeansCode = "10" // espèces
	PaySEPACT    PaymentMeansCode = "58" // virement SEPA
	PaySEPADD    PaymentMeansCode = "59" // prélèvement SEPA
	PayUnknown   PaymentMeansCode = "1"  // instrument non défini
)

var paymentMeansLabels = map[PaymentMeansCode]map[Lang]string{
	PayCredit:    {LangFR: "Virement", LangEN: "Credit transfer"},
	PayDirectDeb: {LangFR: "Prélèvement", LangEN: "Direct debit"},
	PayCard:      {LangFR: "Carte bancaire", LangEN: "Card payment"},
	PayCheque:    {LangFR: "Chèque", LangEN: "Cheque"},
	PayCash:      {LangFR: "Espèces", LangEN: "Cash"},
	PaySEPACT:    {LangFR: "Virement SEPA", LangEN: "SEPA credit transfer"},
	PaySEPADD:    {LangFR: "Prélèvement SEPA", LangEN: "SEPA direct debit"},
}

// Valid renvoie true si le code moyen de paiement est non vide (liste UNTDID large, non close ici).
func (c PaymentMeansCode) Valid() bool { return c != "" }

// Label renvoie le libellé du moyen de paiement.
func (c PaymentMeansCode) Label(l Lang) string {
	return labelOf(paymentMeansLabels[c], l, string(c))
}

// UnitCode — BT-130, unité de mesure (UN/ECE Recommendation 20).
type UnitCode string

const (
	UnitPiece    UnitCode = "C62" // unité (one)
	UnitKilogram UnitCode = "KGM"
	UnitLitre    UnitCode = "LTR"
	UnitMetre    UnitCode = "MTR"
	UnitHour     UnitCode = "HUR"
	UnitDay      UnitCode = "DAY"
	UnitLumpSum  UnitCode = "LS" // forfait
	UnitSquareM  UnitCode = "MTK"
)

// Valid renvoie true si le code unité est non vide (liste rec20 large, non close ici).
func (c UnitCode) Valid() bool { return c != "" }

// labelOf est l'utilitaire commun de résolution de libellé avec repli.
func labelOf(m map[Lang]string, l Lang, fallback string) string {
	if m == nil {
		return fallback
	}
	if s, ok := m[l]; ok {
		return s
	}
	if s, ok := m[LangFR]; ok {
		return s
	}
	return fallback
}
