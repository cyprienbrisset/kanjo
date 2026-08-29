package model

// AllowanceCharge représente une remise (allowance) ou une charge (charge), au niveau
// document (BG-20/21) ou au niveau ligne (BG-27/28). Le booléen IsCharge distingue les deux.
type AllowanceCharge struct {
	IsCharge   bool     `json:"isCharge"`             // true = charge (BG-21), false = remise (BG-20)
	Amount     Amount   `json:"amount"`               // BT-92/99 montant
	BaseAmount *Amount  `json:"baseAmount,omitempty"` // BT-93/100 base de calcul
	Percent    *Decimal `json:"percent,omitempty"`    // BT-94/101 pourcentage

	ReasonCode string `json:"reasonCode,omitempty"` // BT-98/105 code motif
	Reason     string `json:"reason,omitempty"`     // BT-97/104 motif

	// TVA applicable à la remise/charge de niveau document (BG-20/21)
	TaxCategory TaxCategoryCode `json:"taxCategory,omitempty"` // BT-95/102
	TaxRate     *Decimal        `json:"taxRate,omitempty"`     // BT-96/103
}

// Signed renvoie le montant signé : positif pour une charge, négatif pour une remise.
func (ac AllowanceCharge) Signed() Amount {
	if ac.IsCharge {
		return ac.Amount
	}
	return ac.Amount.Neg()
}
