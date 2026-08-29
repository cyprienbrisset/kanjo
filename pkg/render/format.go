// Package render produit la face lisible d'une facture (HTML) et les rapports de validation
// (HTML), en pur Go via html/template, dans l'esprit du système de design 大福帳 (§12, §G10).
// Le rendu PDF passe par ce HTML et un moteur externe optionnel (détecté par doctor, §G10).
package render

import (
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// Lang est la langue d'affichage.
type Lang = model.Lang

// FormatMoney rend un montant selon la convention française : « 1 250,00 € » (espace fine
// insécable comme séparateur de milliers, virgule décimale). Pour en/de, format simple.
func FormatMoney(a model.Amount, lang Lang) string {
	dec := a.String() // "1250.00"
	intPart, frac, _ := strings.Cut(dec, ".")
	neg := strings.HasPrefix(intPart, "-")
	intPart = strings.TrimPrefix(intPart, "-")

	symbol := currencySymbol(a.Currency)
	switch lang {
	case model.LangEN:
		grouped := groupThousands(intPart, ",")
		out := symbol + grouped
		if frac != "" {
			out = symbol + grouped + "." + frac
		}
		if neg {
			out = "-" + out
		}
		return out
	default: // fr (et de, proche)
		grouped := groupThousands(intPart, " ") // espace fine insécable
		out := grouped
		if frac != "" {
			out += "," + frac
		}
		out += " " + symbol
		if neg {
			out = "-" + out
		}
		return out
	}
}

func currencySymbol(cur string) string {
	switch cur {
	case "EUR":
		return "€"
	case "USD":
		return "$"
	case "GBP":
		return "£"
	default:
		return cur
	}
}

// groupThousands insère un séparateur tous les trois chiffres.
func groupThousands(digits, sep string) string {
	n := len(digits)
	if n <= 3 {
		return digits
	}
	var b strings.Builder
	first := n % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(digits[:first])
	for i := first; i < n; i += 3 {
		b.WriteString(sep)
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

// FormatPercent rend un taux « 20 % ».
func FormatPercent(d model.Decimal) string { return d.String() + " %" }
