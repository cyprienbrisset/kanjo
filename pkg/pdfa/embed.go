package pdfa

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// EmbedResult porte le PDF produit et l'état de conformité (jamais simulé, §9.2/§17.7).
type EmbedResult struct {
	PDF         []byte
	AttachedAs  string
	PDFAChecked bool // false : la conformité PDF/A-3b n'a pas été validée (veraPDF absent)
	Warnings    []string
}

// EmbedXML embarque le XML de facture dans un PDF existant sous le nom `attachName`
// (typiquement "factur-x.xml"). Il attache le fichier via pdfcpu.
//
// Honnêteté (§17.7) : cette fonction NE prétend PAS produire un PDF/A-3b conforme. Elle
// embarque la pièce jointe de façon extractible ; la mise en conformité PDF/A-3 complète
// (OutputIntent, XMP fx:, /AFRelationship /Data) et sa validation veraPDF restent à compléter.
// Le résultat porte PDFAChecked=false pour ne jamais mentir sur la conformité.
func EmbedXML(pdf, xml []byte, attachName string) (*EmbedResult, error) {
	if attachName == "" {
		attachName = "factur-x.xml"
	}
	tmpDir, err := os.MkdirTemp("", "kanjo-embed-*")
	if err != nil {
		return nil, fmt.Errorf("pdfa: dossier temporaire: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// pdfcpu utilise le nom de base du fichier comme nom de pièce jointe.
	xmlPath := filepath.Join(tmpDir, attachName)
	if err := os.WriteFile(xmlPath, xml, 0o600); err != nil {
		return nil, fmt.Errorf("pdfa: écriture temporaire du XML: %w", err)
	}

	var buf bytes.Buffer
	conf := config()
	if err := api.AddAttachments(bytes.NewReader(pdf), &buf, []string{xmlPath}, false, conf); err != nil {
		if isEncryptedErr(err) {
			return nil, ErrEncrypted
		}
		return nil, fmt.Errorf("pdfa: embarquement de %s: %w", attachName, err)
	}

	return &EmbedResult{
		PDF:         buf.Bytes(),
		AttachedAs:  attachName,
		PDFAChecked: false,
		Warnings:    []string{"W-PDF-001: conformité PDF/A-3b non vérifiée (embarquement de la pièce jointe uniquement)"},
	}, nil
}
