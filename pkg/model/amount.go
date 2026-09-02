package model

import (
	"errors"
	"fmt"
	"math/big"
)

// Amount représente un montant monétaire exact : une valeur entière en unités mineures
// (centimes pour l'euro), une échelle (nombre de décimales) et une devise ISO 4217.
// C'est le seul type autorisé pour les montants du pivot (§5.2, règle 1 — jamais de float64).
type Amount struct {
	Value    int64  `json:"value"`    // en unités mineures (ex. centimes)
	Scale    uint8  `json:"scale"`    // nombre de décimales, généralement 2
	Currency string `json:"currency"` // ISO 4217, ex. "EUR"
}

// Erreurs d'arithmétique monétaire.
var (
	ErrCurrencyMismatch = errors.New("devises incompatibles")
	ErrAmountParse      = errors.New("montant invalide")
)

// NewAmount construit un montant.
func NewAmount(value int64, scale uint8, currency string) Amount {
	return Amount{Value: value, Scale: scale, Currency: currency}
}

// ZeroAmount renvoie un montant nul dans la devise donnée, à l'échelle 2.
func ZeroAmount(currency string) Amount { return Amount{Value: 0, Scale: 2, Currency: currency} }

// ParseAmount analyse une représentation décimale ("1250.00") dans une devise donnée.
func ParseAmount(s, currency string) (Amount, error) {
	d, err := ParseDecimal(s)
	if err != nil {
		return Amount{}, fmt.Errorf("%w: %v", ErrAmountParse, err)
	}
	return Amount{Value: d.Unscaled, Scale: d.Scale, Currency: currency}, nil
}

// MustParseAmount est la variante paniquant, réservée aux tests et littéraux internes.
func MustParseAmount(s, currency string) Amount {
	a, err := ParseAmount(s, currency)
	if err != nil {
		panic(err)
	}
	return a
}

// Decimal renvoie la partie numérique du montant, sans devise.
func (a Amount) Decimal() Decimal { return Decimal{Unscaled: a.Value, Scale: a.Scale} }

// String rend le montant en notation décimale sans symbole de devise ("1250.00"),
// forme attendue par les syntaxes XML (CII/UBL).
func (a Amount) String() string { return a.Decimal().String() }

// IsZero indique si le montant vaut zéro.
func (a Amount) IsZero() bool { return a.Value == 0 }

// Rescale renvoie le montant exprimé à l'échelle cible (arrondi normatif EN 16931).
func (a Amount) Rescale(scale uint8) Amount {
	d := a.Decimal().Rescale(scale)
	return Amount{Value: d.Unscaled, Scale: d.Scale, Currency: a.Currency}
}

// sameCurrency vérifie la compatibilité de devise (une devise vide est considérée neutre).
func (a Amount) sameCurrency(b Amount) error {
	if a.Currency != "" && b.Currency != "" && a.Currency != b.Currency {
		return fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, a.Currency, b.Currency)
	}
	return nil
}

func chooseCurrency(a, b Amount) string {
	if a.Currency != "" {
		return a.Currency
	}
	return b.Currency
}

// Add additionne deux montants de même devise. Les échelles sont alignées sur la plus fine.
func (a Amount) Add(b Amount) (Amount, error) {
	if err := a.sameCurrency(b); err != nil {
		return Amount{}, err
	}
	scale := maxU8(a.Scale, b.Scale)
	x := a.Rescale(scale)
	y := b.Rescale(scale)
	sum := new(big.Int).Add(big.NewInt(x.Value), big.NewInt(y.Value))
	v, ok := int64Checked(sum)
	if !ok {
		return Amount{}, fmt.Errorf("%w: %s + %s", ErrOverflow, a, b)
	}
	return Amount{Value: v, Scale: scale, Currency: chooseCurrency(a, b)}, nil
}

// Sub soustrait b de a (même devise).
func (a Amount) Sub(b Amount) (Amount, error) {
	if err := a.sameCurrency(b); err != nil {
		return Amount{}, err
	}
	scale := maxU8(a.Scale, b.Scale)
	x := a.Rescale(scale)
	y := b.Rescale(scale)
	diff := new(big.Int).Sub(big.NewInt(x.Value), big.NewInt(y.Value))
	v, ok := int64Checked(diff)
	if !ok {
		return Amount{}, fmt.Errorf("%w: %s - %s", ErrOverflow, a, b)
	}
	return Amount{Value: v, Scale: scale, Currency: chooseCurrency(a, b)}, nil
}

// Neg renvoie l'opposé du montant.
func (a Amount) Neg() Amount { return Amount{Value: -a.Value, Scale: a.Scale, Currency: a.Currency} }

// MulQuantity multiplie le montant par une quantité décimale (ex. prix unitaire × quantité)
// et renvoie le résultat arrondi à l'échelle cible (typiquement 2) selon l'arrondi EN 16931.
func (a Amount) MulQuantity(qty Decimal, targetScale uint8) Amount {
	// produit exact : (a.Value × 10^-a.Scale) × (qty.Unscaled × 10^-qty.Scale)
	prod := new(big.Int).Mul(big.NewInt(a.Value), big.NewInt(qty.Unscaled))
	prodScale := int(a.Scale) + int(qty.Scale)

	d := Decimal{Scale: uint8(prodScale)}
	if prodScale > 255 {
		// cas pathologique : réduire d'abord par big.Int (rare en pratique)
		d = reduceBig(prod, prodScale, int(targetScale))
		return Amount{Value: d.Unscaled, Scale: targetScale, Currency: a.Currency}
	}
	if !prod.IsInt64() {
		d = reduceBig(prod, prodScale, int(targetScale))
		return Amount{Value: d.Unscaled, Scale: targetScale, Currency: a.Currency}
	}
	d.Unscaled = prod.Int64()
	out := d.Rescale(targetScale)
	return Amount{Value: out.Unscaled, Scale: targetScale, Currency: a.Currency}
}

// reduceBig ramène (unscaled × 10^-fromScale) à toScale avec arrondi half-away-from-zero,
// en restant en big.Int tout du long pour éviter tout dépassement intermédiaire.
func reduceBig(unscaled *big.Int, fromScale, toScale int) Decimal {
	if toScale >= fromScale {
		factor := pow10Big(toScale - fromScale)
		v := new(big.Int).Mul(unscaled, factor)
		return Decimal{Unscaled: int64OrPanic(v), Scale: uint8(toScale)}
	}
	den := pow10Big(fromScale - toScale)
	return Decimal{Unscaled: roundDivBig(unscaled, den), Scale: uint8(toScale)}
}

// Cmp compare deux montants de même devise après alignement des échelles : -1, 0 ou 1.
func (a Amount) Cmp(b Amount) (int, error) {
	if err := a.sameCurrency(b); err != nil {
		return 0, err
	}
	scale := maxU8(a.Scale, b.Scale)
	x := a.Rescale(scale)
	y := b.Rescale(scale)
	switch {
	case x.Value < y.Value:
		return -1, nil
	case x.Value > y.Value:
		return 1, nil
	default:
		return 0, nil
	}
}

// Equal indique l'égalité de valeur (échelles alignées) et de devise.
func (a Amount) Equal(b Amount) bool {
	c, err := a.Cmp(b)
	return err == nil && c == 0
}

// SumAmounts additionne une série de montants ; renvoie un zéro à l'échelle 2 si la liste est vide.
func SumAmounts(currency string, amounts ...Amount) (Amount, error) {
	acc := ZeroAmount(currency)
	for _, x := range amounts {
		var err error
		acc, err = acc.Add(x)
		if err != nil {
			return Amount{}, err
		}
	}
	return acc, nil
}

func maxU8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}
