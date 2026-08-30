package pdfa

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// EmbedResult porte le PDF produit et l'état de conformité (jamais simulé, §9.2/§17.7).
type EmbedResult struct {
	PDF         []byte
	AttachedAs  string
	PDFAChecked bool // true UNIQUEMENT si un validateur (veraPDF) a effectivement confirmé (§17.7)
	Warnings    []string
}

// EmbedXML embarque le XML de facture dans un PDF existant sous le nom `attachName`
// (typiquement "factur-x.xml"), en établissant l'association Factur-X requise par PDF/A-3 :
// EmbeddedFile /Subtype text/xml, /AFRelationship /Data sur la spécification de fichier, et
// référencement par le tableau /AF du catalogue.
//
// Honnêteté (§17.7) : cette fonction établit une association *structurellement conforme*
// (vérifiable par relecture), mais ne PRÉTEND PAS à la conformité PDF/A-3b globale — celle-ci
// dépend du PDF de base et n'est affirmée que si veraPDF la confirme (voir ValidatePDFA).
// `PDFAChecked` reste false ici : aucun verdict n'est apposé sans validation effective.
func EmbedXML(pdf, xml []byte, attachName string) (*EmbedResult, error) {
	if attachName == "" {
		attachName = "factur-x.xml"
	}

	conf := config()
	ctx, err := api.ReadContext(bytes.NewReader(pdf), conf)
	if err != nil {
		if isEncryptedErr(err) {
			return nil, ErrEncrypted
		}
		return nil, fmt.Errorf("pdfa: lecture du PDF: %w", err)
	}

	desc := "Facture électronique (XML) — donnée du document"
	if err := embedFacturX(ctx.XRefTable, xml, attachName, desc, time.Now()); err != nil {
		return nil, fmt.Errorf("pdfa: embarquement Factur-X de %s: %w", attachName, err)
	}

	var buf bytes.Buffer
	if err := api.WriteContext(ctx, &buf); err != nil {
		return nil, fmt.Errorf("pdfa: écriture du PDF: %w", err)
	}

	return &EmbedResult{
		PDF:         buf.Bytes(),
		AttachedAs:  attachName,
		PDFAChecked: false,
		Warnings: []string{
			"W-PDF-001: association Factur-X établie (/AF, /AFRelationship) ; conformité PDF/A-3b globale non vérifiée sans veraPDF (voir kanjo doctor / job CI veraPDF)",
		},
	}, nil
}
