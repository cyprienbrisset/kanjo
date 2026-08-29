package model

// DocumentKind distingue les natures de document couvertes par le pivot.
type DocumentKind string

const (
	KindInvoice    DocumentKind = "invoice"    // facture
	KindCreditNote DocumentKind = "creditNote" // avoir
	KindOrder      DocumentKind = "order"      // bon de commande (Order-X, L3)
)

// Valid indique si la nature est connue.
func (k DocumentKind) Valid() bool {
	switch k {
	case KindInvoice, KindCreditNote, KindOrder:
		return true
	default:
		return false
	}
}

// OperationCat est la catégorie d'opération au sens des mentions CTC françaises.
type OperationCat string

const (
	OpGoods    OperationCat = "biens"
	OpServices OperationCat = "services"
	OpMixed    OperationCat = "mixte"
)

// Valid indique si la catégorie est connue.
func (o OperationCat) Valid() bool {
	switch o {
	case OpGoods, OpServices, OpMixed, "":
		return true
	default:
		return false
	}
}

// Lang est un code de langue d'affichage des libellés (fr par défaut, référence).
type Lang string

const (
	LangFR Lang = "fr"
	LangEN Lang = "en"
	LangDE Lang = "de"
)
