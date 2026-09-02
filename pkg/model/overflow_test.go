package model

import (
	"errors"
	"strings"
	"testing"
)

// TestParseDecimalRejectsExcessiveMagnitude : une valeur trop grande pour être remise à l'échelle
// sans risque de dépassement est refusée À LA LECTURE (entrée hostile → erreur, jamais panique).
func TestParseDecimalRejectsExcessiveMagnitude(t *testing.T) {
	// 16 chiffres significatifs → au-delà de la borne (10^15).
	big := strings.Repeat("9", 16)
	if _, err := ParseDecimal(big); !errors.Is(err, ErrDecimalParse) {
		t.Fatalf("attendu ErrDecimalParse pour %q, obtenu %v", big, err)
	}
	// Même chose via un dénominateur de très grande partie entière avec décimales.
	if _, err := ParseDecimal("12345678901234567.89"); !errors.Is(err, ErrDecimalParse) {
		t.Fatalf("attendu ErrDecimalParse pour une magnitude excessive, obtenu %v", err)
	}
}

// TestParseDecimalAcceptsLargeButBounded : 15 chiffres significatifs restent acceptés et la
// remise à l'échelle usuelle ne panique pas.
func TestParseDecimalAcceptsLargeButBounded(t *testing.T) {
	d, err := ParseDecimal("999999999999999") // 15 chiffres
	if err != nil {
		t.Fatalf("valeur bornée refusée à tort: %v", err)
	}
	// Remise à l'échelle +2 (×100) : ne doit pas paniquer et rester exacte.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Rescale a paniqué sur une valeur bornée: %v", r)
		}
	}()
	got := d.Rescale(2)
	if got.Scale != 2 {
		t.Fatalf("échelle attendue 2, obtenue %d", got.Scale)
	}
}

// TestAmountAddOverflowReturnsError : une addition qui déborde int64 renvoie ErrOverflow au lieu
// de paniquer (les valeurs sont construites directement, hors chemin de lecture borné).
func TestAmountAddOverflowReturnsError(t *testing.T) {
	a := NewAmount(1<<62, 0, "EUR")
	b := NewAmount(1<<62, 0, "EUR") // 2^62 + 2^62 = 2^63 > int64 max
	if _, err := a.Add(b); !errors.Is(err, ErrOverflow) {
		t.Fatalf("attendu ErrOverflow, obtenu %v", err)
	}
}

// TestAmountSubOverflowReturnsError : une soustraction qui déborde renvoie ErrOverflow.
// -2^62 - (2^62 + 1) = -2^63 - 1 < int64 min.
func TestAmountSubOverflowReturnsError(t *testing.T) {
	a := NewAmount(-(1 << 62), 0, "EUR")
	b := NewAmount((1<<62)+1, 0, "EUR")
	if _, err := a.Sub(b); !errors.Is(err, ErrOverflow) {
		t.Fatalf("attendu ErrOverflow, obtenu %v", err)
	}
}
