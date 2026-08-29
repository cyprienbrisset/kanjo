package model

import "testing"

func TestParseDecimalRoundTrip(t *testing.T) {
	cases := []struct {
		in       string
		unscaled int64
		scale    uint8
		str      string
	}{
		{"12.50", 1250, 2, "12.50"},
		{"0", 0, 0, "0"},
		{"-3", -3, 0, "-3"},
		{"0.005", 5, 3, "0.005"},
		{"1,50", 150, 2, "1.50"},
		{"1000000.00", 100000000, 2, "1000000.00"},
		{"-0.01", -1, 2, "-0.01"},
	}
	for _, c := range cases {
		d, err := ParseDecimal(c.in)
		if err != nil {
			t.Fatalf("ParseDecimal(%q): %v", c.in, err)
		}
		if d.Unscaled != c.unscaled || d.Scale != c.scale {
			t.Errorf("ParseDecimal(%q) = {%d,%d}, veut {%d,%d}", c.in, d.Unscaled, d.Scale, c.unscaled, c.scale)
		}
		if got := d.String(); got != c.str {
			t.Errorf("String(%q) = %q, veut %q", c.in, got, c.str)
		}
	}
}

func TestParseDecimalInvalid(t *testing.T) {
	for _, in := range []string{"", "  ", "abc", "1.2.3", "1e3", "1 000", "--1", "."} {
		if _, err := ParseDecimal(in); err == nil {
			t.Errorf("ParseDecimal(%q) devrait échouer", in)
		}
	}
}

func TestDecimalRescaleRoundsHalfAwayFromZero(t *testing.T) {
	cases := []struct {
		in    string
		scale uint8
		want  string
	}{
		{"1.005", 2, "1.01"},   // demi vers le haut
		{"1.004", 2, "1.00"},   // en dessous
		{"-1.005", 2, "-1.01"}, // demi en s'éloignant de zéro
		{"2.5", 0, "3"},
		{"-2.5", 0, "-3"},
		{"1.2345", 2, "1.23"},
		{"1.00", 4, "1.0000"}, // extension d'échelle
	}
	for _, c := range cases {
		got := MustParseDecimal(c.in).Rescale(c.scale).String()
		if got != c.want {
			t.Errorf("%s.Rescale(%d) = %q, veut %q", c.in, c.scale, got, c.want)
		}
	}
}
