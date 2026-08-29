package model

import (
	"testing"
	"time"
)

func TestDateParseAndFormat(t *testing.T) {
	d, err := ParseISO("2026-09-01")
	if err != nil {
		t.Fatal(err)
	}
	if d.Year != 2026 || d.Month != time.September || d.Day != 1 {
		t.Errorf("date = %+v", d)
	}
	if d.Compact() != "20260901" {
		t.Errorf("Compact = %s", d.Compact())
	}
	c, err := ParseCompact("20260901")
	if err != nil {
		t.Fatal(err)
	}
	if c != d {
		t.Errorf("ParseCompact != ParseISO : %+v vs %+v", c, d)
	}
	if c.ISO() != "2026-09-01" {
		t.Errorf("ISO = %s", c.ISO())
	}
}

func TestParseDateAutoDetect(t *testing.T) {
	iso, _ := ParseDate("2026-09-01")
	comp, _ := ParseDate("20260901")
	if iso != comp {
		t.Errorf("détection auto incohérente : %+v vs %+v", iso, comp)
	}
}

func TestParseISOTimezoneSuffix(t *testing.T) {
	want, _ := ParseISO("2016-06-27")
	// xs:date autorise un décalage horaire final ; il est ignoré (une date de facture n'a pas de fuseau).
	for _, s := range []string{"2016-06-27+01:00", "2016-06-27-05:00", "2016-06-27Z", "2016-06-27+0100", " 2016-06-27+01:00 "} {
		got, err := ParseISO(s)
		if err != nil {
			t.Errorf("ParseISO(%q) : erreur inattendue %v", s, err)
			continue
		}
		if got != want {
			t.Errorf("ParseISO(%q) = %+v, veut %+v", s, got, want)
		}
	}
	// Un suffixe non-fuseau reste rejeté.
	for _, s := range []string{"2016-06-27X", "2016-06-27+1", "2016-06-27+1:00"} {
		if _, err := ParseISO(s); err == nil {
			t.Errorf("ParseISO(%q) devrait échouer", s)
		}
	}
}

func TestDateInvalid(t *testing.T) {
	for _, s := range []string{"2026-02-30", "2026-13-01", "0000-01-01", "abcd-01-01", "2026-1-1", "20260230"} {
		if _, err := ParseDate(s); err == nil {
			t.Errorf("ParseDate(%q) devrait échouer", s)
		}
	}
}

func TestDateOrdering(t *testing.T) {
	a, _ := ParseISO("2026-08-29")
	b, _ := ParseISO("2026-09-01")
	if !a.Before(b) || !b.After(a) || a.After(b) {
		t.Error("ordre des dates incorrect")
	}
}
