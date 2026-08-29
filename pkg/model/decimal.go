package model

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// Decimal représente un nombre décimal exact sans devise : quantités (BT-129),
// pourcentages de TVA (BT-119), pourcentages de remise, etc. Il évite tout float64
// et donc toute dérive d'arrondi (§5.2).
//
// La valeur représentée est Unscaled × 10^-Scale. Exemple : 12,50 → {Unscaled: 1250, Scale: 2}.
type Decimal struct {
	Unscaled int64 `json:"unscaled"`
	Scale    uint8 `json:"scale"`
}

// ErrDecimalParse est renvoyée lorsqu'une chaîne ne représente pas un décimal valide.
var ErrDecimalParse = errors.New("décimal invalide")

// NewDecimal construit un décimal à partir d'une valeur non mise à l'échelle et d'une échelle.
func NewDecimal(unscaled int64, scale uint8) Decimal {
	return Decimal{Unscaled: unscaled, Scale: scale}
}

// DecimalFromInt construit un décimal entier (échelle 0).
func DecimalFromInt(v int64) Decimal { return Decimal{Unscaled: v, Scale: 0} }

// ParseDecimal analyse une représentation décimale ("12.50", "-3", "0.005", "1,50").
// La virgule est acceptée comme séparateur décimal (tolérance de lecture). Aucun
// séparateur de milliers n'est admis.
func ParseDecimal(s string) (Decimal, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return Decimal{}, fmt.Errorf("%w: chaîne vide", ErrDecimalParse)
	}
	raw = strings.Replace(raw, ",", ".", 1)

	neg := false
	switch raw[0] {
	case '+':
		raw = raw[1:]
	case '-':
		neg = true
		raw = raw[1:]
	}
	if raw == "" {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalParse, s)
	}

	intPart, fracPart, hasFrac := strings.Cut(raw, ".")
	if hasFrac && strings.Contains(fracPart, ".") {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalParse, s)
	}
	digits := intPart + fracPart
	if digits == "" {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalParse, s)
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalParse, s)
		}
	}
	if len(fracPart) > 255 {
		return Decimal{}, fmt.Errorf("%w: trop de décimales", ErrDecimalParse)
	}

	bi, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalParse, s)
	}
	if !bi.IsInt64() {
		return Decimal{}, fmt.Errorf("%w: dépassement de capacité", ErrDecimalParse)
	}
	unscaled := bi.Int64()
	if neg {
		unscaled = -unscaled
	}
	return Decimal{Unscaled: unscaled, Scale: uint8(len(fracPart))}, nil
}

// MustParseDecimal est la variante paniquant utilisée dans les tests et les littéraux internes.
func MustParseDecimal(s string) Decimal {
	d, err := ParseDecimal(s)
	if err != nil {
		panic(err)
	}
	return d
}

// String rend le décimal en notation à point ("12.50"), toujours avec Scale décimales.
func (d Decimal) String() string {
	neg := d.Unscaled < 0
	mag := d.Unscaled
	if neg {
		mag = -mag
	}
	digits := fmt.Sprintf("%d", mag)
	s := d.Scale
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	if s == 0 {
		b.WriteString(digits)
		return b.String()
	}
	if len(digits) <= int(s) {
		digits = strings.Repeat("0", int(s)-len(digits)+1) + digits
	}
	split := len(digits) - int(s)
	b.WriteString(digits[:split])
	b.WriteByte('.')
	b.WriteString(digits[split:])
	return b.String()
}

// IsZero indique si le décimal vaut zéro.
func (d Decimal) IsZero() bool { return d.Unscaled == 0 }

// bigValue renvoie la valeur non mise à l'échelle comme *big.Int.
func (d Decimal) bigValue() *big.Int { return big.NewInt(d.Unscaled) }

// Rescale renvoie une copie du décimal exprimée avec l'échelle cible, en arrondissant
// au plus proche (demi vers l'infini positif/négatif selon le signe, « half away from zero »),
// conformément à l'arrondi normatif d'EN 16931.
func (d Decimal) Rescale(scale uint8) Decimal {
	if scale == d.Scale {
		return d
	}
	if scale > d.Scale {
		factor := pow10(int(scale) - int(d.Scale))
		return Decimal{Unscaled: mulInt64(d.Unscaled, factor), Scale: scale}
	}
	// scale < d.Scale : réduction avec arrondi half-away-from-zero.
	factor := pow10Big(int(d.Scale) - int(scale))
	rounded := roundDivBig(big.NewInt(d.Unscaled), factor)
	return Decimal{Unscaled: rounded, Scale: scale}
}

// pow10 renvoie 10^n en int64 (n petit, sans dépassement attendu pour les échelles usuelles).
func pow10(n int) int64 {
	r := int64(1)
	for i := 0; i < n; i++ {
		r *= 10
	}
	return r
}

func pow10Big(n int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}

// mulInt64 multiplie deux int64 en paniquant en cas de dépassement (usage interne contrôlé).
func mulInt64(a, b int64) int64 {
	r := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	if !r.IsInt64() {
		panic("model: dépassement int64 lors d'une mise à l'échelle")
	}
	return r.Int64()
}

// roundDivBig divise num par den (den > 0) avec arrondi half-away-from-zero et renvoie un int64.
func roundDivBig(num, den *big.Int) int64 {
	q := new(big.Int)
	r := new(big.Int)
	q.QuoRem(num, den, r) // troncature vers zéro, r du signe de num
	if r.Sign() == 0 {
		return int64OrPanic(q)
	}
	// 2*|r| >= den → arrondir en s'éloignant de zéro
	absR := new(big.Int).Abs(r)
	twice := new(big.Int).Lsh(absR, 1)
	if twice.Cmp(den) >= 0 {
		if num.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	return int64OrPanic(q)
}

func int64OrPanic(b *big.Int) int64 {
	if !b.IsInt64() {
		panic("model: dépassement int64")
	}
	return b.Int64()
}

// bigMul renvoie a×b comme nouveau *big.Int.
func bigMul(a, b *big.Int) *big.Int { return new(big.Int).Mul(a, b) }
