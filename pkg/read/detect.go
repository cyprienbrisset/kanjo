package read

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// Detect identifie le format d'un flux par inspection de son contenu : octets magiques, racine XML
// et espace de noms (§7.1 MUST — jamais sur l'extension seule).
//
// Detect réalise un ROUTAGE : il désigne le lecteur le plus probable. La confirmation NORMATIVE
// (schéma, espace de noms exact, profil) incombe au lecteur (`pkg/read/<format>`), qui rejette un
// document mal formé. Là où la racine seule est ambiguë — `<Invoice>`/`<CreditNote>`, partagées
// hors UBL — l'espace de noms est vérifié : un namespace présent mais non-UBL renvoie
// FormatUnknown plutôt qu'un routage UBL trompeur. Un namespace absent reste toléré (certains
// émetteurs l'omettent ; le lecteur tranche).
func Detect(data []byte) Format {
	// Retirer un éventuel BOM UTF-8 puis les blancs de tête.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	trimmed := bytes.TrimLeft(data, " \t\r\n")

	// PDF → Factur-X / ZUGFeRD (le profil et la présence du XML embarqué sont vérifiés au parsing).
	if bytes.HasPrefix(trimmed, []byte("%PDF")) {
		return FormatFacturX
	}

	// UN/EDIFACT : le flux débute par le segment de service UNA ou l'en-tête d'interchange UNB.
	if bytes.HasPrefix(trimmed, []byte("UNA")) || bytes.HasPrefix(trimmed, []byte("UNB")) {
		return FormatEDIFACT
	}

	// JSON pivot Kanjō.
	if len(trimmed) > 0 && trimmed[0] == '{' {
		if bytes.Contains(trimmed, []byte(`"schemaVersion"`)) && bytes.Contains(trimmed, []byte(`kanjo/1`)) {
			return FormatKanjoJSON
		}
	}

	// XML : inspecter la première balise ouvrante.
	root, ns := xmlRoot(trimmed)
	switch root {
	case "CrossIndustryInvoice":
		return FormatCII
	case "SCRDMCCBDACIOMESSAGE", "CrossIndustryOrder":
		return FormatOrderX
	case "CrossIndustryDocument":
		return FormatZUGFeRD1 // ZUGFeRD 1.0 (CII D14B)
	case "FatturaElettronica":
		return FormatFatturaPA // FatturaPA italienne (v1.2)
	case "Invoice":
		if isUBLNamespace(ns) {
			return FormatUBLInvoice
		}
		return FormatUnknown
	case "CreditNote":
		if isUBLNamespace(ns) {
			return FormatUBLCreditNote
		}
		return FormatUnknown
	}
	return FormatUnknown
}

// isUBLNamespace indique si l'espace de noms correspond à UBL (Invoice-2 / CreditNote-2). Un
// namespace absent est toléré (le lecteur UBL confirme) ; un namespace présent mais étranger est
// rejeté, évitant qu'un `<Invoice>` non-UBL soit routé à tort vers le lecteur UBL.
func isUBLNamespace(ns string) bool {
	if ns == "" {
		return true
	}
	return strings.Contains(ns, "oasis:names:specification:ubl") ||
		strings.Contains(ns, "Invoice-2") ||
		strings.Contains(ns, "CreditNote-2")
}

// xmlRoot renvoie le nom local et l'URI d'espace de noms de l'élément racine, sans parser tout le
// document. Il s'arrête au premier StartElement.
//
// Il utilise RawToken (et NON Token) délibérément : sur une entrée non fiable, Token résoudrait la
// DTD et expanserait les entités (risque XXE / bombe) AVANT que xmlsafe n'intervienne. RawToken ne
// résolvant pas les préfixes, l'URI de l'espace de noms racine est reconstituée depuis les
// déclarations xmlns portées par la balise (voir rootNamespaceURI).
func xmlRoot(data []byte) (local, space string) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false // détection tolérante ; la sécurité est appliquée au vrai parsing (xmlsafe)
	for {
		tok, err := dec.RawToken()
		if err == io.EOF || err != nil {
			return "", ""
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, rootNamespaceURI(se)
		}
	}
}

// rootNamespaceURI résout l'URI d'espace de noms de l'élément racine à partir des déclarations
// xmlns de la balise. Avec RawToken, se.Name.Space contient le PRÉFIXE (ou "" si racine sans
// préfixe / namespace par défaut) ; on cherche la déclaration correspondante dans les attributs.
func rootNamespaceURI(se xml.StartElement) string {
	prefix := se.Name.Space
	for _, a := range se.Attr {
		if prefix == "" {
			if a.Name.Space == "" && a.Name.Local == "xmlns" { // xmlns="…" (namespace par défaut)
				return a.Value
			}
		} else if a.Name.Space == "xmlns" && a.Name.Local == prefix { // xmlns:prefix="…"
			return a.Value
		}
	}
	return ""
}
