package model

// SchemaVersion est la version du schéma du pivot sérialisé en JSON (ADR-007).
const SchemaVersion = "github.com/cyprienbrisset/kanjo/1"

// Document est la racine du modèle pivot. Il couvre facture, avoir, facture rectificative
// et (en L3) bon de commande. Les identifiants BT-xx/BG-xx renvoient à EN 16931.
type Document struct {
	SchemaVersion string       `json:"schemaVersion"` // "github.com/cyprienbrisset/kanjo/1"
	Kind          DocumentKind `json:"kind"`          // invoice | creditNote | order

	// --- En-tête ---
	ID                 string   `json:"id"`                                 // BT-1  Numéro de facture
	IssueDate          Date     `json:"issueDate"`                          // BT-2
	TypeCode           TypeCode `json:"typeCode"`                           // BT-3
	CurrencyCode       string   `json:"currencyCode"`                       // BT-5  ISO 4217
	TaxCurrencyCode    string   `json:"taxCurrencyCode,omitempty"`          // BT-6
	TaxPointDate       *Date    `json:"taxPointDate,omitempty"`             // BT-7
	TaxPointDateCode   string   `json:"taxPointDateCode,omitempty"`         // BT-8 code de date d'exigibilité (UNTDID 2005 restreint)
	Notes              []Note   `json:"notes,omitempty"`                    // BG-1
	BuyerReference     string   `json:"buyerReference,omitempty"`           // BT-10
	ProjectRef         string   `json:"projectReference,omitempty"`         // BT-11
	ContractRef        string   `json:"contractReference,omitempty"`        // BT-12
	PurchaseOrderRef   string   `json:"purchaseOrderReference,omitempty"`   // BT-13
	SalesOrderRef      string   `json:"salesOrderReference,omitempty"`      // BT-14
	ReceivingAdviceRef string   `json:"receivingAdviceReference,omitempty"` // BT-15
	DespatchAdviceRef  string   `json:"despatchAdviceReference,omitempty"`  // BT-16

	// --- Parties ---
	Seller    Party         `json:"seller"`                      // BG-4
	Buyer     Party         `json:"buyer"`                       // BG-7
	Payee     *Party        `json:"payee,omitempty"`             // BG-10
	TaxRep    *Party        `json:"taxRepresentative,omitempty"` // BG-11
	DeliverTo *DeliveryInfo `json:"deliverTo,omitempty"`         // BG-13

	// --- Période et références ---
	Period     *Period     `json:"period,omitempty"`            // BG-14
	Precedings []Preceding `json:"precedingInvoices,omitempty"` // BG-3

	// --- Paiement ---
	PaymentInstructions *PaymentInstructions `json:"paymentInstructions,omitempty"` // BG-16
	PaymentTerms        string               `json:"paymentTerms,omitempty"`        // BT-20
	DueDate             *Date                `json:"dueDate,omitempty"`             // BT-9

	// --- Charges et remises niveau document ---
	AllowanceCharges []AllowanceCharge `json:"allowanceCharges,omitempty"` // BG-20/21

	// --- TVA ---
	TaxBreakdown []TaxSubtotal `json:"taxBreakdown"` // BG-23

	// --- Lignes ---
	Lines []Line `json:"lines"` // BG-25

	// --- Totaux ---
	Totals Totals `json:"totals"` // BG-22

	// --- Pièces jointes ---
	Attachments []Attachment `json:"attachments,omitempty"` // BG-24

	// --- Hors norme ---
	Extensions Extensions  `json:"extensions,omitempty"`
	Provenance *Provenance `json:"-"` // jamais sérialisé, usage diagnostic
}

// NewDocument initialise un document du type donné avec la version de schéma courante.
func NewDocument(kind DocumentKind) *Document {
	return &Document{SchemaVersion: SchemaVersion, Kind: kind}
}

// IsCreditNote indique si le document est un avoir (par nature ou par code type).
func (d *Document) IsCreditNote() bool {
	return d.Kind == KindCreditNote || d.TypeCode.IsCreditNote()
}

// Currency renvoie la devise du document (BT-5), utilisée par défaut pour les montants.
func (d *Document) Currency() string { return d.CurrencyCode }

// LineByID renvoie la ligne portant l'identifiant donné, ou nil.
func (d *Document) LineByID(id string) *Line {
	for i := range d.Lines {
		if d.Lines[i].ID == id {
			return &d.Lines[i]
		}
	}
	return nil
}

// SumLineNetAmounts renvoie la somme des montants nets de ligne (base de BR-CO-13 / BT-106).
func (d *Document) SumLineNetAmounts() (Amount, error) {
	acc := ZeroAmount(d.CurrencyCode)
	for _, l := range d.Lines {
		var err error
		acc, err = acc.Add(l.NetAmount)
		if err != nil {
			return Amount{}, err
		}
	}
	return acc.Rescale(2), nil
}

// SumTaxAmounts renvoie la somme des montants de TVA de la ventilation (base de BT-110).
func (d *Document) SumTaxAmounts() (Amount, error) {
	acc := ZeroAmount(d.CurrencyCode)
	for _, t := range d.TaxBreakdown {
		var err error
		acc, err = acc.Add(t.TaxAmount)
		if err != nil {
			return Amount{}, err
		}
	}
	return acc.Rescale(2), nil
}
