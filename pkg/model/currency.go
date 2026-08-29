package model

// zeroDecimalCurrencies liste les devises usuelles sans unité mineure (0 décimale) : les
// montants et la TVA s'y arrondissent à l'entier. Les autres devises utilisent 2 décimales.
var zeroDecimalCurrencies = map[string]bool{
	"BIF": true, "CLP": true, "DJF": true, "GNF": true, "HUF": true, "ISK": true,
	"JPY": true, "KMF": true, "KRW": true, "PYG": true, "RWF": true, "UGX": true,
	"VND": true, "VUV": true, "XAF": true, "XOF": true, "XPF": true,
}

// threeDecimalCurrencies liste les devises usuelles à 3 décimales.
var threeDecimalCurrencies = map[string]bool{
	"BHD": true, "IQD": true, "JOD": true, "KWD": true, "LYD": true, "OMR": true, "TND": true,
}

// MinorUnits renvoie le nombre de décimales (unités mineures) d'une devise ISO 4217 : 0 pour
// les devises entières (JPY, HUF…), 3 pour certaines devises du Golfe, 2 par défaut.
func MinorUnits(currency string) uint8 {
	switch {
	case zeroDecimalCurrencies[currency]:
		return 0
	case threeDecimalCurrencies[currency]:
		return 3
	default:
		return 2
	}
}
