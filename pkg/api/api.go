// Package api définit les DTO partagés entre les façades (CLI, TUI, GUI, HTTP) et
// l'enveloppe JSON stable des sorties machine (§11.3, ADR-007). Toute sortie JSON de
// Kanjō respecte cette enveloppe ; schemaVersion change au moindre retrait/renommage.
package api

import "github.com/cyprienbrisset/kanjo/internal/version"

// Envelope est l'enveloppe commune de toute sortie JSON.
type Envelope struct {
	SchemaVersion string   `json:"schemaVersion"`
	Command       string   `json:"command"`
	StartedAt     string   `json:"startedAt"`
	DurationMs    int64    `json:"durationMs"`
	Summary       Summary  `json:"summary"`
	Results       []Result `json:"results"`
}

// NewEnvelope initialise une enveloppe pour une commande donnée.
func NewEnvelope(command, startedAt string) *Envelope {
	return &Envelope{
		SchemaVersion: version.Schema,
		Command:       command,
		StartedAt:     startedAt,
	}
}

// Summary agrège les compteurs de verdict d'un traitement.
type Summary struct {
	Total   int `json:"total"`
	OK      int `json:"ok"`
	Warning int `json:"warning"`
	Error   int `json:"error"`
}

// Status est le verdict d'un résultat unitaire.
type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
)

// Result décrit le traitement d'un fichier.
type Result struct {
	Input    string    `json:"input"`
	Output   string    `json:"output,omitempty"`
	Status   Status    `json:"status"`
	Format   string    `json:"format,omitempty"`
	Profile  string    `json:"profile,omitempty"`
	Findings []Finding `json:"findings,omitempty"`
	Losses   []Loss    `json:"losses,omitempty"`
	Hashes   *Hashes   `json:"hashes,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// Finding est une anomalie de validation, dans l'enveloppe JSON (miroir de rules.Finding).
type Finding struct {
	RuleID     string `json:"ruleId"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Term       string `json:"term,omitempty"`
	Path       string `json:"path,omitempty"`
	SourcePath string `json:"sourcePath,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	Fixable    bool   `json:"fixable"`
}

// Loss décrit une perte d'information lors d'une conversion dégradante (§7.3, §8).
type Loss struct {
	Code     string `json:"code"`     // ex. "W-EXT-001"
	Severity string `json:"severity"` // "warning" | "error"
	Message  string `json:"message"`
	Term     string `json:"term,omitempty"`
	Path     string `json:"path,omitempty"`
}

// Hashes porte les empreintes d'entrée et de sortie (traçabilité §17.5).
type Hashes struct {
	InputSha256  string `json:"inputSha256,omitempty"`
	OutputSha256 string `json:"outputSha256,omitempty"`
}

// Add incrémente le résumé selon le statut d'un résultat.
func (s *Summary) Add(status Status) {
	s.Total++
	switch status {
	case StatusOK:
		s.OK++
	case StatusWarning:
		s.Warning++
	case StatusError:
		s.Error++
	}
}
