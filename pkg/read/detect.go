package read

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// Detect identifie le format d'un flux par inspection de son contenu : octets magiques,
// racine XML et espaces de noms (§7.1 MUST — jamais sur l'extension seule).
func Detect(data []byte) Format {
	// Retirer un éventuel BOM UTF-8 puis les blancs de tête.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	trimmed := bytes.TrimLeft(data, " \t\r\n")

	// PDF → Factur-X / ZUGFeRD (le profil et la présence du XML embarqué sont vérifiés au parsing).
	if bytes.HasPrefix(trimmed, []byte("%PDF")) {
		return FormatFacturX
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
	case "CrossIndustryDocument":
		return FormatZUGFeRD1 // ZUGFeRD 1.0 (CII D14B)
	case "FatturaElettronica":
		return FormatFatturaPA // FatturaPA italienne (v1.2)
	case "Invoice":
		if strings.Contains(ns, "Invoice-2") || strings.Contains(ns, "oasis") || ns == "" {
			return FormatUBLInvoice
		}
		return FormatUBLInvoice
	case "CreditNote":
		return FormatUBLCreditNote
	}
	return FormatUnknown
}

// xmlRoot renvoie le nom local et l'espace de noms de l'élément racine, sans parser tout le
// document. Il s'arrête au premier StartElement.
func xmlRoot(data []byte) (local, space string) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false // détection tolérante ; la sécurité est appliquée au vrai parsing (xmlsafe)
	for {
		tok, err := dec.RawToken()
		if err == io.EOF || err != nil {
			return "", ""
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, se.Name.Space
		}
	}
}
