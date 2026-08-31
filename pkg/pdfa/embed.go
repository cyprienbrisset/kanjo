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
	desc := "Facture électronique (XML) — donnée du document"

	// Voie privilégiée : mise à jour INCRÉMENTALE. Les octets du PDF de base restent intacts
	// (préfixe exact du résultat), donc sa conformité PDF/A-3b est PRÉSERVÉE — contrairement à une
	// réécriture complète. On ne se rabat sur pdfcpu que si la structure xref n'est pas classique.
	if out, err := embedIncremental(pdf, xml, attachName, desc, time.Now()); err == nil {
		return &EmbedResult{
			PDF:         out,
			AttachedAs:  attachName,
			PDFAChecked: false,
			Warnings: []string{
				"W-PDF-001: association Factur-X ajoutée par mise à jour incrémentale (octets du PDF de base préservés) ; conformité PDF/A-3b confirmée par veraPDF uniquement (job CI).",
			},
		}, nil
	} else if err != ErrIncrementalUnsupported {
		return nil, fmt.Errorf("pdfa: embarquement incrémental de %s: %w", attachName, err)
	}

	// Repli : réécriture complète via pdfcpu (ne préserve pas la conformité PDF/A globale).
	conf := config()
	ctx, err := api.ReadContext(bytes.NewReader(pdf), conf)
	if err != nil {
		if isEncryptedErr(err) {
			return nil, ErrEncrypted
		}
		return nil, fmt.Errorf("pdfa: lecture du PDF: %w", err)
	}
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
			"W-PDF-002: repli réécriture complète (xref non classique) ; l'association Factur-X est établie mais la conformité PDF/A-3b du PDF de base n'est pas garantie.",
		},
	}, nil
}
