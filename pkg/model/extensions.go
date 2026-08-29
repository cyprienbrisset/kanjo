package model

// Extensions regroupe les informations hors norme EN 16931, typées et non « fourre-tout »
// (§5.2, règle 5).
type Extensions struct {
	// Extensions françaises (mentions obligatoires de la réforme CTC)
	FR *FrenchCTC `json:"fr,omitempty"`
	// Champs du profil EXTENDED non couverts par EN 16931
	Extended *ExtendedFields `json:"extended,omitempty"`
	// Extensions sectorielles / spécifiques émetteur, conservées telles quelles
	Unmapped []UnmappedField `json:"unmapped,omitempty"`
}

// Empty indique l'absence totale d'extensions.
func (e Extensions) Empty() bool {
	return e.FR == nil && e.Extended == nil && len(e.Unmapped) == 0
}

// FrenchCTC porte les mentions obligatoires de la facturation électronique française.
type FrenchCTC struct {
	SellerSIREN       string       `json:"sellerSiren,omitempty"`
	BuyerSIREN        string       `json:"buyerSiren,omitempty"`
	OperationCategory OperationCat `json:"operationCategory,omitempty"` // biens | services | mixte
	VATOnDebitsOption *bool        `json:"vatOnDebitsOption,omitempty"` // option TVA sur les débits
	DeliveryAddress   *Address     `json:"deliveryAddress,omitempty"`
	PublicOrderRef    string       `json:"publicOrderReference,omitempty"`
}

// ExtendedFields porte les champs du profil EXTENDED de Factur-X non couverts par EN 16931.
type ExtendedFields struct {
	// Réservé à l'implémentation progressive du profil EXTENDED (L2).
	// Les champs non modélisés restent dans Unmapped pour garantir l'aller-retour.
	Fields map[string]string `json:"fields,omitempty"`
}

// UnmappedField conserve un champ source non modélisé, pour garantir l'aller-retour
// sans perte sur un format identique.
type UnmappedField struct {
	Syntax    string `json:"syntax"` // "cii" | "ubl"
	XPath     string `json:"xpath"`
	Value     string `json:"value"`
	Namespace string `json:"namespace,omitempty"`
}
