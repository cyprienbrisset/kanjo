package model

import "math/big"

// Line représente une ligne de facture (BG-25).
type Line struct {
	ID           string `json:"id"`                 // BT-126 identifiant de ligne
	Note         string `json:"note,omitempty"`     // BT-127 note de ligne
	ObjectID     string `json:"objectId,omitempty"` // BT-128 identifiant d'objet
	ObjectScheme string `json:"objectScheme,omitempty"`

	// Article (BG-31)
	Name             string `json:"name"`                       // BT-153 désignation de l'article
	Description      string `json:"description,omitempty"`      // BT-154 description
	SellerAssignedID string `json:"sellerAssignedId,omitempty"` // BT-155
	BuyerAssignedID  string `json:"buyerAssignedId,omitempty"`  // BT-156
	StandardID       string `json:"standardId,omitempty"`       // BT-157 (ex. EAN/GTIN)
	StandardScheme   string `json:"standardScheme,omitempty"`
	OriginCountry    string `json:"originCountry,omitempty"` // BT-159

	// Classification de marchandise (BT-158) — optionnelle ; si présente, son schéma (listID)
	// est requis par BR-65.
	ClassificationID     string `json:"classificationId,omitempty"`     // BT-158
	ClassificationScheme string `json:"classificationScheme,omitempty"` // schéma (listID) de BT-158

	// Quantité et prix
	Quantity Decimal  `json:"quantity"` // BT-129 quantité facturée
	UnitCode UnitCode `json:"unitCode"` // BT-130 unité

	// QuantityPresent indique que la quantité (BT-129) était réellement portée par la source
	// (distinction « absent » vs « zéro », règle 5 du CDC), nécessaire à BR-22. Métadonnée de
	// lecture, non sérialisée.
	QuantityPresent bool     `json:"-"`
	NetPrice        Amount   `json:"netPrice"`                    // BT-146 prix unitaire net
	PriceBaseQty    *Decimal `json:"priceBaseQuantity,omitempty"` // BT-149 quantité de base du prix
	GrossPrice      *Amount  `json:"grossPrice,omitempty"`        // BT-148 prix unitaire brut
	PriceDiscount   *Amount  `json:"priceDiscount,omitempty"`     // BT-147 remise sur prix unitaire

	// TVA de la ligne (BG-30)
	TaxCategory TaxCategoryCode `json:"taxCategory"`       // BT-151
	TaxRate     *Decimal        `json:"taxRate,omitempty"` // BT-152 taux (%)

	// Remises et charges de ligne (BG-27/28)
	AllowanceCharges []AllowanceCharge `json:"allowanceCharges,omitempty"`

	// Montant net de ligne (BT-131)
	NetAmount Amount `json:"netAmount"`

	// Références et période propres à la ligne
	Period      *Period `json:"period,omitempty"`      // BG-26
	OrderLineID string  `json:"orderLineId,omitempty"` // BT-132 référence ligne de commande

	// Attributs d'article (BG-32) et données comptables (BT-133)
	Attributes    []ItemAttribute `json:"attributes,omitempty"`
	AccountingRef string          `json:"accountingReference,omitempty"` // BT-133
}

// ItemAttribute est un attribut d'article (BG-32) : couple nom/valeur.
type ItemAttribute struct {
	Name  string `json:"name"`  // BT-160
	Value string `json:"value"` // BT-161
}

// ComputeNetAmount calcule le montant net de ligne attendu :
// (prix net ÷ quantité de base × quantité facturée) ± remises/charges de ligne.
// Le résultat est arrondi à 2 décimales (arrondi EN 16931). Il ne modifie pas la ligne ;
// c'est le moteur de règles qui compare ce résultat au NetAmount porté.
func (l Line) ComputeNetAmount(currency string) (Amount, error) {
	base := DecimalFromInt(1)
	if l.PriceBaseQty != nil && !l.PriceBaseQty.IsZero() {
		base = *l.PriceBaseQty
	}
	// prix ramené à l'unité : NetPrice / base
	unit := l.NetPrice
	if base.Unscaled != 1 || base.Scale != 0 {
		// division exacte via mise à l'échelle : (NetPrice × 10^k) / base, puis arrondi fin
		unit = divideAmountByDecimal(l.NetPrice, base, 6)
	}
	line := unit.MulQuantity(l.Quantity, 2)
	for _, ac := range l.AllowanceCharges {
		var err error
		if ac.IsCharge {
			line, err = line.Add(ac.Amount)
		} else {
			line, err = line.Sub(ac.Amount)
		}
		if err != nil {
			return Amount{}, err
		}
	}
	line.Currency = currency
	return line.Rescale(2), nil
}

// divideAmountByDecimal divise un montant par un décimal en conservant `scale` décimales,
// avec arrondi half-away-from-zero. Tout le calcul reste en big.Int pour éviter tout dépassement.
func divideAmountByDecimal(a Amount, d Decimal, scale uint8) Amount {
	// (a.Value / 10^a.Scale) / (d.Unscaled / 10^d.Scale)
	//   = a.Value × 10^d.Scale / (d.Unscaled × 10^a.Scale)
	// on vise `scale` décimales : numérateur × 10^scale.
	num := bigMul(big.NewInt(a.Value), pow10Big(int(d.Scale)+int(scale)))
	den := bigMul(big.NewInt(d.Unscaled), pow10Big(int(a.Scale)))
	neg := den.Sign() < 0
	if neg {
		num.Neg(num)
		den.Neg(den)
	}
	q := roundDivBig(num, den)
	return Amount{Value: q, Scale: scale, Currency: a.Currency}
}
