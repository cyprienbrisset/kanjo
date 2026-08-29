// Package fatturapa lit le format italien FatturaPA (FatturaElettronica v1.2) et le convertit
// vers le modèle pivot. Le parsing passe par internal/xmlsafe (§17.1). Les tags sont appariés
// par nom local (le format emploie un préfixe d'espace de noms variable, souvent « p: »).
//
// Couverture : en-tête (cédant/cessionnaire), lignes de détail, ventilation de TVA et totaux.
// Les extensions purement italiennes non modélisées (bollo, ritenuta, cassa…) ne sont pas
// remontées dans cette version et feront l'objet d'un rapport de perte ultérieur.
package fatturapa

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/internal/xmlsafe"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

func init() { read.Register(read.FormatFatturaPA, Read) }

// Read désérialise un flux FatturaPA (FatturaElettronica) en document pivot.
func Read(data []byte, sourceName string) (*model.Document, error) {
	var x fattura
	if err := xmlsafe.Decode(data, &x); err != nil {
		return nil, fmt.Errorf("lecture FatturaPA %s: %w", sourceName, err)
	}
	return mapToPivot(&x, sourceName)
}

// --- Structures XML FatturaElettronica v1.2 (tags par nom local) ---

type fattura struct {
	Header struct {
		Cedente     anagrafica `xml:"CedentePrestatore"`
		Cessionario anagrafica `xml:"CessionarioCommittente"`
	} `xml:"FatturaElettronicaHeader"`
	Body struct {
		DatiGenerali struct {
			Documento struct {
				TipoDocumento string `xml:"TipoDocumento"`
				Divisa        string `xml:"Divisa"`
				Data          string `xml:"Data"`
				Numero        string `xml:"Numero"`
				ImportoTotale string `xml:"ImportoTotaleDocumento"`
			} `xml:"DatiGeneraliDocumento"`
		} `xml:"DatiGenerali"`
		DatiBeniServizi struct {
			Linee     []dettaglioLinea `xml:"DettaglioLinee"`
			Riepilogo []datiRiepilogo  `xml:"DatiRiepilogo"`
		} `xml:"DatiBeniServizi"`
	} `xml:"FatturaElettronicaBody"`
}

type anagrafica struct {
	DatiAnagrafici struct {
		IdFiscaleIVA struct {
			IdPaese  string `xml:"IdPaese"`
			IdCodice string `xml:"IdCodice"`
		} `xml:"IdFiscaleIVA"`
		CodiceFiscale string `xml:"CodiceFiscale"`
		Anagrafica    struct {
			Denominazione string `xml:"Denominazione"`
			Nome          string `xml:"Nome"`
			Cognome       string `xml:"Cognome"`
		} `xml:"Anagrafica"`
	} `xml:"DatiAnagrafici"`
	Sede struct {
		Indirizzo string `xml:"Indirizzo"`
		CAP       string `xml:"CAP"`
		Comune    string `xml:"Comune"`
		Provincia string `xml:"Provincia"`
		Nazione   string `xml:"Nazione"`
	} `xml:"Sede"`
}

type dettaglioLinea struct {
	NumeroLinea    string `xml:"NumeroLinea"`
	Descrizione    string `xml:"Descrizione"`
	Quantita       string `xml:"Quantita"`
	UnitaMisura    string `xml:"UnitaMisura"`
	PrezzoUnitario string `xml:"PrezzoUnitario"`
	PrezzoTotale   string `xml:"PrezzoTotale"`
	AliquotaIVA    string `xml:"AliquotaIVA"`
	Natura         string `xml:"Natura"`
}

type datiRiepilogo struct {
	AliquotaIVA       string `xml:"AliquotaIVA"`
	ImponibileImporto string `xml:"ImponibileImporto"`
	Imposta           string `xml:"Imposta"`
	Natura            string `xml:"Natura"`
	EsigibilitaIVA    string `xml:"EsigibilitaIVA"`
}

func trimmed(s string) string { return strings.TrimSpace(s) }
