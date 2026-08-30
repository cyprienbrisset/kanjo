package model

import "testing"

func TestIsKnownCurrency(t *testing.T) {
	for _, c := range []string{"EUR", "USD", "JPY", "gbp"} {
		if !IsKnownCurrency(c) {
			t.Errorf("%q devrait être une devise connue", c)
		}
	}
	for _, c := range []string{"EURO", "XXX", "", "US"} {
		if IsKnownCurrency(c) {
			t.Errorf("%q ne devrait pas être une devise connue", c)
		}
	}
}

func TestIsKnownCountry(t *testing.T) {
	for _, c := range []string{"FR", "DE", "it", "EL", "UK"} {
		if !IsKnownCountry(c) {
			t.Errorf("%q devrait être un pays connu", c)
		}
	}
	for _, c := range []string{"XX", "FRA", "", "ZZ"} {
		if IsKnownCountry(c) {
			t.Errorf("%q ne devrait pas être un pays connu", c)
		}
	}
}
