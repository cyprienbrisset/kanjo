// Package read définit l'interface commune des lecteurs de formats et la détection de format
// par inspection du contenu (jamais par l'extension seule, §7.1 MUST).
//
// Chaque format concret (CII, UBL, Factur-X…) vit dans un sous-paquet qui s'enregistre via
// Register() dans son init(). Ce paquet n'importe aucun sous-paquet, ce qui évite les cycles :
// un point d'assemblage (import « à blanc » des sous-paquets) réalise le câblage.
package read

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// Format identifie un format d'entrée reconnu.
type Format string

const (
	FormatUnknown       Format = "unknown"
	FormatCII           Format = "cii"            // CrossIndustryInvoice D16B (XML seul)
	FormatZUGFeRD1      Format = "zugferd1"       // ZUGFeRD 1.0 — CrossIndustryDocument (CII D14B)
	FormatUBLInvoice    Format = "ubl"            // UBL 2.1 Invoice
	FormatUBLCreditNote Format = "ubl-creditnote" // UBL 2.1 CreditNote
	FormatFacturX       Format = "facturx"        // PDF/A-3 + XML CII embarqué
	FormatFatturaPA     Format = "fatturapa"      // FatturaPA italienne (FatturaElettronica v1.2)
	FormatOrderX        Format = "orderx"         // Order-X (bon de commande, CrossIndustryOrder)
	FormatEDIFACT       Format = "edifact"        // UN/EDIFACT INVOIC (message texte, ISO 9735)
	FormatKanjoJSON     Format = "json"           // pivot Kanjō sérialisé
)

// Reader lit des octets et produit un document pivot. Le nom de source sert au diagnostic
// et à la provenance.
type Reader func(data []byte, sourceName string) (*model.Document, error)

// Result agrège le document lu et les métadonnées de détection.
type Result struct {
	Doc     *model.Document
	Format  Format
	Profile string
}

var (
	mu       sync.RWMutex
	registry = map[Format]Reader{}
)

// ErrUnsupportedFormat est renvoyée quand aucun lecteur n'est enregistré pour un format.
var ErrUnsupportedFormat = errors.New("format non pris en charge")

// Register associe un lecteur à un format. Appelé par chaque sous-paquet dans son init().
func Register(f Format, r Reader) {
	mu.Lock()
	defer mu.Unlock()
	registry[f] = r
}

// Get renvoie le lecteur enregistré pour un format, ou nil.
func Get(f Format) Reader {
	mu.RLock()
	defer mu.RUnlock()
	return registry[f]
}

// ReadBytes détecte le format puis délègue au lecteur adéquat.
func ReadBytes(data []byte, sourceName string) (*Result, error) {
	f := Detect(data)
	if f == FormatUnknown {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, sourceName)
	}
	r := Get(f)
	if r == nil {
		return nil, fmt.Errorf("%w: %s (%s)", ErrUnsupportedFormat, sourceName, f)
	}
	doc, err := r(data, sourceName)
	if err != nil {
		return nil, err
	}
	profile := ""
	if doc.Provenance != nil {
		profile = doc.Provenance.SourceProfile
	}
	return &Result{Doc: doc, Format: f, Profile: profile}, nil
}
