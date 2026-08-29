package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Date représente une date calendaire de facture, sans heure ni fuseau horaire (§5.2, règle 2).
// Une date de facture n'a pas de fuseau : utiliser time.Time introduirait des décalages.
type Date struct {
	Year  int        `json:"year"`
	Month time.Month `json:"month"`
	Day   int        `json:"day"`
}

// ErrDateParse est renvoyée pour une date syntaxiquement ou calendairement invalide.
var ErrDateParse = errors.New("date invalide")

// NewDate construit une date et vérifie sa validité calendaire.
func NewDate(year int, month time.Month, day int) (Date, error) {
	d := Date{Year: year, Month: month, Day: day}
	if !d.Valid() {
		return Date{}, fmt.Errorf("%w: %04d-%02d-%02d", ErrDateParse, year, int(month), day)
	}
	return d, nil
}

// Valid vérifie que la date existe réellement dans le calendrier grégorien.
func (d Date) Valid() bool {
	if d.Month < time.January || d.Month > time.December {
		return false
	}
	if d.Day < 1 || d.Day > 31 || d.Year < 1 || d.Year > 9999 {
		return false
	}
	t := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
	return t.Year() == d.Year && t.Month() == d.Month && t.Day() == d.Day
}

// IsZero indique une date non renseignée (valeur nulle du type).
func (d Date) IsZero() bool { return d == Date{} }

// ISO rend la date au format "AAAA-MM-JJ" (ISO 8601, utilisé par UBL).
func (d Date) ISO() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, int(d.Month), d.Day)
}

// Compact rend la date au format "AAAAMMJJ" (format 102 UN/CEFACT, utilisé par CII).
func (d Date) Compact() string {
	return fmt.Sprintf("%04d%02d%02d", d.Year, int(d.Month), d.Day)
}

// String est un alias lisible d'ISO.
func (d Date) String() string { return d.ISO() }

// ParseISO analyse une date "AAAA-MM-JJ", en tolérant un décalage horaire final optionnel
// (Z, +HH:MM, -HH:MM, +HHMM…) autorisé par xs:date : la partie fuseau est ignorée puisqu'une
// date de facture n'a pas de fuseau (§5.2).
func ParseISO(s string) (Date, error) {
	s = strings.TrimSpace(s)
	core := s
	if len(s) > 10 && s[4] == '-' && s[7] == '-' && isTimezoneSuffix(s[10:]) {
		core = s[:10]
	}
	if len(core) != 10 || core[4] != '-' || core[7] != '-' {
		return Date{}, fmt.Errorf("%w: format ISO attendu AAAA-MM-JJ, reçu %q", ErrDateParse, s)
	}
	return parseYMD(core[0:4], core[5:7], core[8:10], s)
}

// isTimezoneSuffix indique si s est un décalage horaire valide ("", "Z", "+HH:MM", "-HHMM"…).
func isTimezoneSuffix(s string) bool {
	if s == "" || s == "Z" {
		return true
	}
	if s[0] != '+' && s[0] != '-' {
		return false
	}
	digits := strings.Replace(s[1:], ":", "", 1)
	if len(digits) != 2 && len(digits) != 4 {
		return false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ParseCompact analyse une date "AAAAMMJJ" (format CII 102).
func ParseCompact(s string) (Date, error) {
	if len(s) != 8 {
		return Date{}, fmt.Errorf("%w: format compact attendu AAAAMMJJ, reçu %q", ErrDateParse, s)
	}
	return parseYMD(s[0:4], s[4:6], s[6:8], s)
}

// ParseDate détecte automatiquement le format (ISO ou compact) et analyse la date.
// C'est le point d'entrée utilisé par les readers ; il tolère les deux syntaxes du socle.
func ParseDate(s string) (Date, error) {
	s = strings.TrimSpace(s)
	if len(s) == 8 { // format compact CII (102) AAAAMMJJ
		return ParseCompact(s)
	}
	return ParseISO(s) // ISO AAAA-MM-JJ, décalage horaire final toléré
}

func parseYMD(ys, ms, ds, orig string) (Date, error) {
	y, err := strconv.Atoi(ys)
	if err != nil {
		return Date{}, fmt.Errorf("%w: %q", ErrDateParse, orig)
	}
	m, err := strconv.Atoi(ms)
	if err != nil {
		return Date{}, fmt.Errorf("%w: %q", ErrDateParse, orig)
	}
	dd, err := strconv.Atoi(ds)
	if err != nil {
		return Date{}, fmt.Errorf("%w: %q", ErrDateParse, orig)
	}
	return NewDate(y, time.Month(m), dd)
}

// Before indique si d est strictement antérieure à o.
func (d Date) Before(o Date) bool {
	if d.Year != o.Year {
		return d.Year < o.Year
	}
	if d.Month != o.Month {
		return d.Month < o.Month
	}
	return d.Day < o.Day
}

// After indique si d est strictement postérieure à o.
func (d Date) After(o Date) bool { return o.Before(d) }
