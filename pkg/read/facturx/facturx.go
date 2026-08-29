// Package facturx lit une facture hybride Factur-X (PDF/A-3 + XML CII/UBL embarqué) :
// il extrait le XML embarqué puis délègue au lecteur XML approprié via le registre (§7.1).
package facturx

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/pdfa"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

func init() { read.Register(read.FormatFacturX, Read) }

// Read extrait le XML de facture d'un PDF Factur-X et le convertit en pivot.
func Read(data []byte, sourceName string) (*model.Document, error) {
	xml, filename, warnCode, err := pdfa.ExtractInvoiceXML(data)
	if err != nil {
		return nil, fmt.Errorf("lecture Factur-X %s: %w", sourceName, err)
	}
	rd, err := read.ReadBytes(xml, sourceName)
	if err != nil {
		return nil, fmt.Errorf("lecture du XML embarqué (%s) de %s: %w", filename, sourceName, err)
	}
	doc := rd.Doc
	if doc.Provenance == nil {
		doc.Provenance = model.NewProvenance(sourceName, "facturx", rd.Profile)
	}
	// Conserver le profil interne mais marquer le format porteur comme facturx.
	doc.Provenance.SourceFormat = "facturx"
	doc.Provenance.SourceFile = sourceName
	if warnCode != "" {
		doc.Provenance.Warn(model.ReadWarning{
			Code:    warnCode,
			Message: fmt.Sprintf("nom de pièce jointe non conforme (%s), contenu valide extrait", filename),
		})
	}
	return doc, nil
}
