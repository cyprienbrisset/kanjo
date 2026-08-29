package model

// Note est une note libre de niveau document (BG-1).
type Note struct {
	Content     string `json:"content"`               // BT-22
	SubjectCode string `json:"subjectCode,omitempty"` // BT-21 code sujet
}

// Period est une période de facturation (BG-14 au niveau document, BG-26 au niveau ligne).
type Period struct {
	Start *Date `json:"start,omitempty"` // BT-73/134
	End   *Date `json:"end,omitempty"`   // BT-74/135
}

// Preceding référence une facture antérieure (BG-3), utile pour les avoirs et rectificatifs.
type Preceding struct {
	ID        string `json:"id"`                  // BT-25
	IssueDate *Date  `json:"issueDate,omitempty"` // BT-26
}

// Attachment est une pièce jointe (BG-24) ou une référence de document additionnelle.
type Attachment struct {
	ID          string `json:"id"`                    // BT-122
	Description string `json:"description,omitempty"` // BT-123
	TypeCode    string `json:"typeCode,omitempty"`    // 916 = pièce jointe référencée
	URI         string `json:"uri,omitempty"`         // BT-124 URL externe
	Filename    string `json:"filename,omitempty"`    // BT-125 nom de fichier embarqué
	MIMEType    string `json:"mimeType,omitempty"`    // BT-125-1
	Data        []byte `json:"-"`                     // contenu binaire (non sérialisé en JSON pivot)
}

// HasEmbeddedData indique si la pièce jointe porte un contenu binaire embarqué.
func (a Attachment) HasEmbeddedData() bool { return len(a.Data) > 0 }
