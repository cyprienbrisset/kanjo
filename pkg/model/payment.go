package model

// PaymentInstructions regroupe les informations de paiement (BG-16).
type PaymentInstructions struct {
	MeansCode      PaymentMeansCode `json:"meansCode,omitempty"`      // BT-81 moyen de paiement
	MeansText      string           `json:"meansText,omitempty"`      // BT-82 libellé du moyen
	RemittanceInfo string           `json:"remittanceInfo,omitempty"` // BT-83 référence de paiement

	CreditTransfers []CreditTransfer `json:"creditTransfers,omitempty"` // BG-17 virements
	Card            *PaymentCard     `json:"card,omitempty"`            // BG-18 carte
	DirectDebit     *DirectDebit     `json:"directDebit,omitempty"`     // BG-19 prélèvement
}

// CreditTransfer décrit un compte de destination d'un virement (BG-17).
type CreditTransfer struct {
	IBAN        string `json:"iban,omitempty"`        // BT-84 identifiant du compte
	AccountName string `json:"accountName,omitempty"` // BT-85 nom du compte
	BIC         string `json:"bic,omitempty"`         // BT-86 identifiant du prestataire
}

// PaymentCard décrit un paiement par carte (BG-18).
type PaymentCard struct {
	PAN    string `json:"pan,omitempty"`    // BT-87 (numéro masqué)
	Holder string `json:"holder,omitempty"` // BT-88
}

// DirectDebit décrit un prélèvement (BG-19).
type DirectDebit struct {
	MandateReference string `json:"mandateReference,omitempty"` // BT-89
	CreditorID       string `json:"creditorId,omitempty"`       // BT-90 ICS
	DebitedAccount   string `json:"debitedAccount,omitempty"`   // BT-91 IBAN débité
}
