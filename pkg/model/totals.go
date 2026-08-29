package model

// Totals regroupe les montants totaux de la facture (BG-22).
type Totals struct {
	LineExtensionAmount           Amount  `json:"lineExtensionAmount"`           // BT-106 somme des montants nets de ligne
	AllowanceTotal                *Amount `json:"allowanceTotal,omitempty"`      // BT-107 total des remises document
	ChargeTotal                   *Amount `json:"chargeTotal,omitempty"`         // BT-108 total des charges document
	TaxExclusiveAmount            Amount  `json:"taxExclusiveAmount"`            // BT-109 total HT
	TaxAmount                     Amount  `json:"taxAmount"`                     // BT-110 total TVA
	TaxAmountInAccountingCurrency *Amount `json:"taxAmountAccounting,omitempty"` // BT-111
	TaxInclusiveAmount            Amount  `json:"taxInclusiveAmount"`            // BT-112 total TTC
	RoundingAmount                *Amount `json:"roundingAmount,omitempty"`      // BT-114 arrondi
	PrepaidAmount                 *Amount `json:"prepaidAmount,omitempty"`       // BT-113 acompte versé
	DuePayableAmount              Amount  `json:"duePayableAmount"`              // BT-115 net à payer
}

// amountOrZero renvoie *a ou un zéro dans la devise donnée si le pointeur est nil.
func amountOrZero(a *Amount, currency string) Amount {
	if a == nil {
		return ZeroAmount(currency)
	}
	return *a
}

// ComputeDuePayable calcule le net à payer attendu (BR-CO-16) :
// TTC (BT-112) − acompte (BT-113) + arrondi (BT-114).
func (t Totals) ComputeDuePayable(currency string) (Amount, error) {
	due := t.TaxInclusiveAmount
	var err error
	due, err = due.Sub(amountOrZero(t.PrepaidAmount, currency))
	if err != nil {
		return Amount{}, err
	}
	due, err = due.Add(amountOrZero(t.RoundingAmount, currency))
	if err != nil {
		return Amount{}, err
	}
	return due.Rescale(2), nil
}

// ComputeTaxInclusive calcule le TTC attendu (BR-CO-15) : HT (BT-109) + TVA (BT-110).
func (t Totals) ComputeTaxInclusive() (Amount, error) {
	sum, err := t.TaxExclusiveAmount.Add(t.TaxAmount)
	if err != nil {
		return Amount{}, err
	}
	return sum.Rescale(2), nil
}

// ComputeTaxExclusive calcule le HT attendu (BR-CO-13) :
// somme des lignes (BT-106) − remises (BT-107) + charges (BT-108).
func (t Totals) ComputeTaxExclusive(currency string) (Amount, error) {
	ht := t.LineExtensionAmount
	var err error
	ht, err = ht.Sub(amountOrZero(t.AllowanceTotal, currency))
	if err != nil {
		return Amount{}, err
	}
	ht, err = ht.Add(amountOrZero(t.ChargeTotal, currency))
	if err != nil {
		return Amount{}, err
	}
	return ht.Rescale(2), nil
}
