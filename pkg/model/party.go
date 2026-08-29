package model

// Party représente une partie à la transaction (vendeur BG-4, acheteur BG-7, etc.).
type Party struct {
	Name            string             `json:"name,omitempty"`                // BT-27/44 nom
	TradingName     string             `json:"tradingName,omitempty"`         // BT-28/45 nom commercial
	ID              string             `json:"id,omitempty"`                  // BT-29/46 identifiant
	IDScheme        string             `json:"idScheme,omitempty"`            // schéma de l'identifiant (ex. GLN)
	LegalID         string             `json:"legalId,omitempty"`             // BT-30/47 identifiant légal (SIRET/SIREN)
	LegalIDScheme   string             `json:"legalIdScheme,omitempty"`       // ex. "0002" (SIRENE)
	VATID           string             `json:"vatId,omitempty"`               // BT-31/48 n° TVA intracommunautaire
	TaxID           string             `json:"taxId,omitempty"`               // BT-32 identifiant fiscal complémentaire
	Address         Address            `json:"address"`                       // BG-5/8
	Contact         *Contact           `json:"contact,omitempty"`             // BG-6/9
	AdditionalLegal string             `json:"additionalLegalInfo,omitempty"` // BT-33 forme juridique / capital
	ElectronicAddr  *ElectronicAddress `json:"electronicAddress,omitempty"`   // BT-34/49 adresse électronique
}

// Address représente une adresse postale (BG-5 vendeur, BG-8 acheteur, etc.).
type Address struct {
	Line1       string `json:"line1,omitempty"`       // BT-35/50
	Line2       string `json:"line2,omitempty"`       // BT-36/51
	Line3       string `json:"line3,omitempty"`       // BT-162/163
	City        string `json:"city,omitempty"`        // BT-37/52
	PostalCode  string `json:"postalCode,omitempty"`  // BT-38/53
	Subdivision string `json:"subdivision,omitempty"` // BT-39/54 subdivision (région/état)
	CountryCode string `json:"countryCode"`           // BT-40/55 code pays ISO 3166-1 alpha-2
}

// Empty indique une adresse totalement vide.
func (a Address) Empty() bool { return a == Address{} }

// Contact représente les coordonnées de contact d'une partie (BG-6/9).
type Contact struct {
	Name  string `json:"name,omitempty"`  // BT-41/56
	Phone string `json:"phone,omitempty"` // BT-42/57
	Email string `json:"email,omitempty"` // BT-43/58
}

// ElectronicAddress est l'adresse électronique d'échange (BT-34/49) avec son schéma (EAS).
type ElectronicAddress struct {
	Value  string `json:"value"`
	Scheme string `json:"scheme,omitempty"` // ex. "0225" (SIRENE), "9957" (peppol)
}

// DeliveryInfo décrit la livraison (BG-13) : lieu et date de livraison.
type DeliveryInfo struct {
	Name           string  `json:"name,omitempty"`       // BT-70 nom du destinataire de la livraison
	LocationID     string  `json:"locationId,omitempty"` // BT-71 identifiant du lieu
	LocationScheme string  `json:"locationScheme,omitempty"`
	Address        Address `json:"address"`                // BG-15 adresse de livraison
	DeliveryDate   *Date   `json:"deliveryDate,omitempty"` // BT-72 date de livraison effective
}
