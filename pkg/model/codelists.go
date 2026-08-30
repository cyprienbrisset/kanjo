package model

import "strings"

// Listes de codes officielles pour la validation (BR-CL-*). Stockées en chaînes compactes et
// indexées à l'init() : plus léger qu'une map littérale, et facile à mettre à jour.

// iso4217 : devises actives ISO 4217 (alpha-3).
const iso4217 = "AED AFN ALL AMD ANG AOA ARS AUD AWG AZN BAM BBD BDT BGN BHD BIF BMD BND BOB BOV " +
	"BRL BSD BTN BWP BYN BZD CAD CDF CHE CHF CHW CLF CLP CNY COP COU CRC CUC CUP CVE CZK DJF DKK " +
	"DOP DZD EGP ERN ETB EUR FJD FKP GBP GEL GHS GIP GMD GNF GTQ GYD HKD HNL HRK HTG HUF IDR ILS " +
	"INR IQD IRR ISK JMD JOD JPY KES KGS KHR KMF KPW KRW KWD KYD KZT LAK LBP LKR LRD LSL LYD MAD " +
	"MDL MGA MKD MMK MNT MOP MRU MUR MVR MWK MXN MXV MYR MZN NAD NGN NIO NOK NPR NZD OMR PAB PEN " +
	"PGK PHP PKR PLN PYG QAR RON RSD RUB RWF SAR SBD SCR SDG SEK SGD SHP SLE SLL SOS SRD SSP STN " +
	"SVC SYP SZL THB TJS TMT TND TOP TRY TTD TWD TZS UAH UGX USD USN UYI UYU UYW UZS VED VES VND " +
	"VUV WST XAF XCD XDR XOF XPF XSU XUA YER ZAR ZMW ZWL"

// iso3166 : codes pays ISO 3166-1 alpha-2 (+ EL/UK tolérés, usités dans les factures européennes).
const iso3166 = "AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM " +
	"BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ " +
	"DK DM DO DZ EC EE EG EH EL ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ " +
	"GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM " +
	"KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ " +
	"MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN " +
	"PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY " +
	"SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UK UM US UY UZ VA VC VE VG VI VN VU " +
	"WF WS YE YT ZA ZM ZW"

var (
	knownCurrencies = fieldSet(iso4217)
	knownCountries  = fieldSet(iso3166)
)

func fieldSet(s string) map[string]bool {
	m := make(map[string]bool)
	for _, f := range strings.Fields(s) {
		m[f] = true
	}
	return m
}

// IsKnownCurrency indique si le code est une devise ISO 4217 active.
func IsKnownCurrency(code string) bool { return knownCurrencies[strings.ToUpper(code)] }

// IsKnownCountry indique si le code est un code pays ISO 3166-1 alpha-2 (EL/UK tolérés).
func IsKnownCountry(code string) bool { return knownCountries[strings.ToUpper(code)] }
