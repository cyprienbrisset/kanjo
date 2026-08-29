package model

import (
	"math/rand"
	"testing"
)

func TestAmountAddSub(t *testing.T) {
	a := MustParseAmount("1250.00", "EUR")
	b := MustParseAmount("249.98", "EUR")
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if sum.String() != "1499.98" {
		t.Errorf("Add = %s, veut 1499.98", sum)
	}
	diff, err := sum.Sub(b)
	if err != nil {
		t.Fatal(err)
	}
	if !diff.Equal(a) {
		t.Errorf("Sub = %s, veut %s", diff, a)
	}
}

func TestAmountCurrencyMismatch(t *testing.T) {
	a := MustParseAmount("10.00", "EUR")
	b := MustParseAmount("10.00", "USD")
	if _, err := a.Add(b); err == nil {
		t.Error("Add de devises différentes devrait échouer")
	}
}

func TestAmountMulQuantity(t *testing.T) {
	// prix unitaire 19,99 € × 3 = 59,97 €
	price := MustParseAmount("19.99", "EUR")
	got := price.MulQuantity(DecimalFromInt(3), 2)
	if got.String() != "59.97" {
		t.Errorf("19.99 × 3 = %s, veut 59.97", got)
	}
	// 0,333 × 3 quantités décimales avec arrondi à 2
	unit := MustParseAmount("0.333", "EUR")
	got = unit.MulQuantity(DecimalFromInt(3), 2)
	if got.String() != "1.00" { // 0.999 arrondi à 1.00
		t.Errorf("0.333 × 3 arrondi = %s, veut 1.00", got)
	}
}

func TestAmountAssociativityNoDrift(t *testing.T) {
	// Propriété : additionner 10 000 montants aléatoires par la gauche ou par la droite
	// donne exactement le même total (pas de dérive d'arrondi, cf. BR-CO-13).
	r := rand.New(rand.NewSource(42))
	const n = 10000
	amounts := make([]Amount, n)
	for i := range amounts {
		amounts[i] = NewAmount(r.Int63n(1_000_00)-500_00, 2, "EUR") // -500.00 à +500.00
	}
	left := ZeroAmount("EUR")
	for _, a := range amounts {
		left, _ = left.Add(a)
	}
	right := ZeroAmount("EUR")
	for i := n - 1; i >= 0; i-- {
		right, _ = right.Add(amounts[i])
	}
	if !left.Equal(right) {
		t.Errorf("dérive détectée : gauche=%s droite=%s", left, right)
	}
	// contrôle croisé avec une somme entière naïve
	var raw int64
	for _, a := range amounts {
		raw += a.Value
	}
	if left.Value != raw {
		t.Errorf("somme=%d, attendu %d", left.Value, raw)
	}
}

func TestAmountCmp(t *testing.T) {
	a := MustParseAmount("100.00", "EUR")
	b := MustParseAmount("100.000", "EUR") // échelle différente, même valeur
	if c, _ := a.Cmp(b); c != 0 {
		t.Errorf("Cmp = %d, veut 0", c)
	}
	if c, _ := a.Cmp(MustParseAmount("100.01", "EUR")); c != -1 {
		t.Errorf("Cmp = %d, veut -1", c)
	}
}
