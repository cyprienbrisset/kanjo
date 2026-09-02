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

// ErrOverflow signale un dépassement de capacité int64 lors d'une opération monétaire.
var ErrOverflow = errors.New("dépassement de capacité")

// maxUnscaledDigits borne le nombre de chiffres significatifs d'un décimal lu depuis un document.
// int64 culmine à ≈ 9,22×10^18 (19 chiffres) ; en plafonnant la valeur non mise à l'échelle à
// 10^15 on garde une marge suffisante pour toute remise à l'échelle interne usuelle (×10^k, k≤3)
// sans dépassement. Au-delà, la valeur est refusée À LA LECTURE (entrée hostile → erreur explicite,
// jamais panique en aval, §17.7 robustesse).
const maxUnscaledDigits = 15

// maxUnscaled = 10^maxUnscaledDigits.
var maxUnscaled = new(big.Int).Exp(big.NewInt(10), big.NewInt(maxUnscaledDigits), nil)

// int64Checked convertit un *big.Int en int64 sans paniquer : (valeur, true) si elle tient, sinon
// (0, false). Utilisé par l'arithmétique monétaire publique pour renvoyer une erreur au lieu de paniquer.
func int64Checked(b *big.Int) (int64, bool) {
	if !b.IsInt64() {
		return 0, false
	}
	return b.Int64(), true
}

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
	// Refus des magnitudes excessives : borne à 10^maxUnscaledDigits pour garantir qu'aucune
	// remise à l'échelle ultérieure ne dépasse int64 (robustesse face à une entrée hostile).
	if !bi.IsInt64() || bi.Cmp(maxUnscaled) >= 0 {
		return Decimal{}, fmt.Errorf("%w: magnitude excessive (%d chiffres significatifs maximum)", ErrDecimalParse, maxUnscaledDigits)
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
