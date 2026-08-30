package fatturapa

import "encoding/xml"

// Structures de sérialisation FatturaElettronica v1.2. Le nom local des éléments correspond à
// celui attendu par le lecteur (pkg/read/fatturapa), garantissant l'aller-retour. Le root porte
// le préfixe d'espace de noms « p: » et sa déclaration, comme dans les fichiers réels.

type root struct {
	XMLName  xml.Name `xml:"p:FatturaElettronica"`
	Xmlns    string   `xml:"xmlns:p,attr"`
	Versione string   `xml:"versione,attr"`
	Header   header   `xml:"FatturaElettronicaHeader"`
	Body     body     `xml:"FatturaElettronicaBody"`
}

type header struct {
	Cedente     anagrafica `xml:"CedentePrestatore"`
	Cessionario anagrafica `xml:"CessionarioCommittente"`
}

type anagrafica struct {
	DatiAnagrafici struct {
		IdFiscaleIVA struct {
			IdPaese  string `xml:"IdPaese,omitempty"`
			IdCodice string `xml:"IdCodice,omitempty"`
		} `xml:"IdFiscaleIVA,omitempty"`
		CodiceFiscale string `xml:"CodiceFiscale,omitempty"`
		Anagrafica    struct {
			Denominazione string `xml:"Denominazione,omitempty"`
			Nome          string `xml:"Nome,omitempty"`
			Cognome       string `xml:"Cognome,omitempty"`
		} `xml:"Anagrafica"`
	} `xml:"DatiAnagrafici"`
	Sede struct {
		Indirizzo string `xml:"Indirizzo,omitempty"`
		CAP       string `xml:"CAP,omitempty"`
		Comune    string `xml:"Comune,omitempty"`
		Provincia string `xml:"Provincia,omitempty"`
		Nazione   string `xml:"Nazione,omitempty"`
	} `xml:"Sede"`
}

type body struct {
	DatiGenerali struct {
		Documento documento `xml:"DatiGeneraliDocumento"`
	} `xml:"DatiGenerali"`
	DatiBeniServizi struct {
		Linee     []dettaglioLinea `xml:"DettaglioLinee"`
		Riepilogo []datiRiepilogo  `xml:"DatiRiepilogo"`
	} `xml:"DatiBeniServizi"`
	DatiPagamento []datiPagamento `xml:"DatiPagamento,omitempty"`
}

type documento struct {
	TipoDocumento string `xml:"TipoDocumento"`
	Divisa        string `xml:"Divisa"`
	Data          string `xml:"Data,omitempty"`
	Numero        string `xml:"Numero"`
	ImportoTotale string `xml:"ImportoTotaleDocumento,omitempty"`
}

type dettaglioLinea struct {
	NumeroLinea    string `xml:"NumeroLinea"`
	Descrizione    string `xml:"Descrizione"`
	Quantita       string `xml:"Quantita,omitempty"`
	UnitaMisura    string `xml:"UnitaMisura,omitempty"`
	PrezzoUnitario string `xml:"PrezzoUnitario"`
	PrezzoTotale   string `xml:"PrezzoTotale"`
	AliquotaIVA    string `xml:"AliquotaIVA"`
	Natura         string `xml:"Natura,omitempty"`
}

type datiRiepilogo struct {
	AliquotaIVA       string `xml:"AliquotaIVA"`
	ImponibileImporto string `xml:"ImponibileImporto"`
	Imposta           string `xml:"Imposta"`
	Natura            string `xml:"Natura,omitempty"`
	EsigibilitaIVA    string `xml:"EsigibilitaIVA,omitempty"`
}

type datiPagamento struct {
	CondizioniPagamento string               `xml:"CondizioniPagamento"`
	Dettaglio           []dettaglioPagamento `xml:"DettaglioPagamento"`
}

type dettaglioPagamento struct {
	ModalitaPagamento string `xml:"ModalitaPagamento"`
	DataScadenza      string `xml:"DataScadenzaPagamento,omitempty"`
	Importo           string `xml:"ImportoPagamento"`
	IBAN              string `xml:"IBAN,omitempty"`
}
