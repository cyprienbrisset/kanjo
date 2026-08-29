// Package pdfa manipule les PDF/A-3 des factures hybrides (Factur-X, Order-X, ZUGFeRD).
// Pour L1, il fournit l'extraction du XML embarqué et des pièces jointes (§9.3).
// L'embarquement et la mise en conformité PDF/A-3 (§9.1/9.2) viendront compléter ce paquet.
package pdfa

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpumodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Erreurs d'extraction.
var (
	// ErrNoInvoiceXML : aucun XML de facture reconnu n'est embarqué (W-PDF hors périmètre facture).
	ErrNoInvoiceXML = errors.New("pdfa: aucun XML de facture embarqué")
	// ErrEncrypted : PDF chiffré, extraction impossible (W-PDF-004).
	ErrEncrypted = errors.New("pdfa: PDF chiffré, extraction impossible")
)

// EmbeddedFile est une pièce jointe extraite d'un PDF.
type EmbeddedFile struct {
	Name string
	Desc string
	Data []byte
}

// invoiceXMLNames liste les noms de pièce jointe reconnus comme XML de facture, par priorité.
var invoiceXMLNames = []string{
	"factur-x.xml",
	"zugferd-invoice.xml",
	"xrechnung.xml",
	"order-x.xml",
}

// config renvoie une configuration pdfcpu hermétique et tolérante.
func config() *pdfcpumodel.Configuration {
	conf := pdfcpumodel.NewDefaultConfiguration()
	conf.ValidationMode = pdfcpumodel.ValidationRelaxed
	return conf
}

// ExtractAttachments extrait toutes les pièces jointes embarquées du PDF.
func ExtractAttachments(data []byte) ([]EmbeddedFile, error) {
	atts, err := api.ExtractAttachmentsRaw(bytes.NewReader(data), "", nil, config())
	if err != nil {
		if isEncryptedErr(err) {
			return nil, ErrEncrypted
		}
		return nil, fmt.Errorf("pdfa: extraction des pièces jointes: %w", err)
	}
	out := make([]EmbeddedFile, 0, len(atts))
	for _, a := range atts {
		b, err := io.ReadAll(a)
		if err != nil {
			return nil, fmt.Errorf("pdfa: lecture de la pièce jointe %s: %w", a.FileName, err)
		}
		out = append(out, EmbeddedFile{Name: a.FileName, Desc: a.Desc, Data: b})
	}
	return out, nil
}

// ExtractInvoiceXML extrait le XML de facture embarqué (Factur-X/ZUGFeRD/Order-X). Il renvoie
// le contenu, le nom de fichier trouvé, et un avertissement W-PDF-003 si le nom n'est pas
// conforme mais qu'un unique XML plausible est présent.
func ExtractInvoiceXML(data []byte) (xml []byte, filename string, warnCode string, err error) {
	atts, err := ExtractAttachments(data)
	if err != nil {
		return nil, "", "", err
	}
	// 1) nom exactement reconnu (priorité).
	byName := map[string]EmbeddedFile{}
	for _, a := range atts {
		byName[strings.ToLower(a.Name)] = a
	}
	for _, want := range invoiceXMLNames {
		if a, ok := byName[want]; ok {
			return a.Data, a.Name, "", nil
		}
	}
	// 2) repli : un unique fichier .xml → extraction avec avertissement de nom non conforme.
	var xmls []EmbeddedFile
	for _, a := range atts {
		if strings.HasSuffix(strings.ToLower(a.Name), ".xml") {
			xmls = append(xmls, a)
		}
	}
	if len(xmls) == 1 {
		return xmls[0].Data, xmls[0].Name, "W-PDF-003", nil
	}
	return nil, "", "", ErrNoInvoiceXML
}

func isEncryptedErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "encrypt") || strings.Contains(s, "password")
}
