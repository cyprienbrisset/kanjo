package model

// TaxSubtotal est une ventilation de TVA par catégorie/taux (BG-23).
type TaxSubtotal struct {
	Category            TaxCategoryCode `json:"category"`                      // BT-118
	Rate                Decimal         `json:"rate"`                          // BT-119 taux (%)
	TaxableAmount       Amount          `json:"taxableAmount"`                 // BT-116 base d'imposition
	TaxAmount           Amount          `json:"taxAmount"`                     // BT-117 montant de TVA
	ExemptionReason     string          `json:"exemptionReason,omitempty"`     // BT-120
	ExemptionReasonCode string          `json:"exemptionReasonCode,omitempty"` // BT-121
}

// ComputeTaxAmount renvoie le montant de TVA attendu pour cette ventilation :
// base × taux%, arrondi à 2 décimales (BR-CO-17 / BR-S-08).
func (t TaxSubtotal) ComputeTaxAmount() Amount {
	// TaxableAmount × (Rate / 100)
	rate := t.Rate
	// diviser le taux par 100 => augmenter l'échelle de 2
	rateOver100 := Decimal{Unscaled: rate.Unscaled, Scale: rate.Scale + 2}
	// Arrondir à la précision de la devise (2 par défaut ; 0 pour JPY/HUF… ; 3 pour KWD…).
	return t.TaxableAmount.MulQuantity(rateOver100, MinorUnits(t.TaxableAmount.Currency))
}
