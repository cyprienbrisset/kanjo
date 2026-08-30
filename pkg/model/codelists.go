package model

import (
	_ "embed"
	"strings"
)

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

// uncl4461 : moyens de paiement (UNTDID/UNCL 4461) usuels.
const uncl4461 = "1 2 3 4 5 6 7 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 30 31 " +
	"32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 " +
	"63 64 65 66 67 68 70 74 75 76 77 78 91 92 93 94 95 96 97 ZZZ"

var knownPaymentMeans = fieldSet(uncl4461)

// IsKnownPaymentMeans indique si le code appartient à la liste UNCL 4461.
func IsKnownPaymentMeans(code string) bool { return knownPaymentMeans[code] }

// uncl5189 : motifs de remise (UNCL 5189) usuels.
const uncl5189 = "41 42 60 62 63 64 65 66 67 68 70 71 88 95 100 102 103 104 105"

// uncl7161 : motifs de charge (UNCL 7161) — sous-ensemble usuel.
const uncl7161 = "AA AAA AAC AAD AAE AAF AAH AAI AAS AAT AAV AAY AAZ ABA ABB ABC ABD ABF ABK ABL " +
	"ABN ABR ABS ABT ABU ACF ACG ACH ACI ACJ ACK ACL ACM ACS ADC ADE ADJ ADK ADL ADM ADN ADO ADP " +
	"ADQ ADR ADT ADW ADY ADZ AEA AEB AEC AED AEF AEH AEI AEJ AEK AEL AEM AEN AEO AEP AES AET AEU " +
	"AEV AEW AEX AEY AEZ AJ AU CA CAB CAD CAE CAF CAI CAJ CAK CAL CAM CAN CAO CAP CAQ CAR CAS CAT " +
	"CAU CAV CAW CAX CAY CAZ CD CG CS CT DAB DAD DL EG EP ER FAA FAB FAC FC FH FI GAA HAA HD HH " +
	"IAA IAB ID IF IR IS KO L1 LA LAA LAB LF MAE MI ML NAA OA PA PAA PC PL RAB RAC RAD RAF RE RF " +
	"RH RV SA SAA SAD SAE SAI SG SH SM SU TAB TAC TT TV V1 V2 WH XAA YY ZZZ"

// cefEAS : schémas d'adresse électronique (CEF EAS) usuels.
const cefEAS = "0002 0007 0009 0037 0060 0088 0096 0106 0130 0135 0142 0151 0183 0184 0188 0190 " +
	"0191 0192 0193 0195 0196 0198 0199 0200 0201 0202 0204 0208 0209 0210 0211 0212 0213 0215 " +
	"0216 9901 9906 9907 9910 9913 9914 9915 9918 9919 9920 9922 9923 9924 9925 9926 9927 9928 " +
	"9929 9930 9931 9932 9933 9934 9935 9936 9937 9938 9939 9940 9941 9942 9943 9944 9945 9946 " +
	"9947 9948 9949 9950 9951 9952 9953 9955 9957 9959"

var (
	knownAllowanceReasons = fieldSet(uncl5189)
	knownChargeReasons    = fieldSet(uncl7161)
	knownEAS              = fieldSet(cefEAS)
)

// IsKnownAllowanceReason indique si le code appartient à UNCL 5189.
func IsKnownAllowanceReason(code string) bool { return knownAllowanceReasons[code] }

// IsKnownChargeReason indique si le code appartient à UNCL 7161.
func IsKnownChargeReason(code string) bool { return knownChargeReasons[code] }

// IsKnownEAS indique si le schéma d'adresse électronique appartient à la liste CEF EAS.
func IsKnownEAS(code string) bool { return knownEAS[code] }

//go:embed codelists_data/icd.txt
var icdData string

var knownICD = fieldSet(icdData)

// IsKnownICD indique si le schéma d'identifiant appartient au registre ISO 6523 ICD (inclut EAS).
func IsKnownICD(code string) bool { return knownICD[code] }

//go:embed codelists_data/units.txt
var unitsData string

//go:embed codelists_data/notesubject.txt
var noteSubjectData string

//go:embed codelists_data/objectscheme.txt
var objectSchemeData string

var (
	knownUnits         = fieldSet(unitsData)
	knownNoteSubjects  = fieldSet(noteSubjectData)
	knownObjectSchemes = fieldSet(objectSchemeData)
)

func IsKnownUnit(code string) bool         { return knownUnits[code] }
func IsKnownNoteSubject(code string) bool  { return knownNoteSubjects[code] }
func IsKnownObjectScheme(code string) bool { return knownObjectSchemes[code] }
