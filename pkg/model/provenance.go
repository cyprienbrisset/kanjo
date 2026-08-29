package model

// Provenance trace l'origine de chaque champ du pivot (ADR-009). Elle n'est jamais
// sérialisée dans le JSON pivot ; elle sert au diagnostic, aux rapports de perte et à
// la correspondance BT ↔ XPath dans l'inspecteur.
type Provenance struct {
	SourceFile    string
	SourceFormat  string            // "facturx", "ubl", "cii", ...
	SourceProfile string            // "en16931", "extended", ...
	FieldOrigins  map[string]string // "BT-1" -> "/rsm:CrossIndustryInvoice/.../ram:ID"
	Warnings      []ReadWarning
}

// NewProvenance initialise une provenance vide prête à recevoir des origines.
func NewProvenance(file, format, profile string) *Provenance {
	return &Provenance{
		SourceFile:    file,
		SourceFormat:  format,
		SourceProfile: profile,
		FieldOrigins:  make(map[string]string),
	}
}

// Record associe un Business Term à son XPath source.
func (p *Provenance) Record(term, xpath string) {
	if p == nil {
		return
	}
	if p.FieldOrigins == nil {
		p.FieldOrigins = make(map[string]string)
	}
	p.FieldOrigins[term] = xpath
}

// Warn ajoute un avertissement de lecture à la provenance.
func (p *Provenance) Warn(w ReadWarning) {
	if p == nil {
		return
	}
	p.Warnings = append(p.Warnings, w)
}

// ReadWarning décrit un avertissement rencontré à la lecture (encodage, champ non mappé…).
type ReadWarning struct {
	Code    string `json:"code"` // ex. "W-ENC-001", "W-EXT-002"
	Message string `json:"message"`
	XPath   string `json:"xpath,omitempty"`
}
