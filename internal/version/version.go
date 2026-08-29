// Package version centralise les trois numéros de version distincts de Kanjō (§19.3) :
// version de l'outil, version du jeu de règles et version du schéma de sortie.
package version

// Ces valeurs sont injectées au build via -ldflags "-X".
var (
	// Tool est la version de l'outil (SemVer), ex. "1.2.0". "dev" hors release.
	Tool = "dev"
	// Commit est le hash du commit de build.
	Commit = "none"
	// BuildDate est la date de build (RFC3339).
	BuildDate = "unknown"
)

const (
	// Rules est la version du jeu de règles de validation, alignée sur la réglementation.
	Rules = "2026.3"
	// Schema est la version du schéma des sorties JSON (ADR-007).
	Schema = "github.com/cyprienbrisset/kanjo/1"
)

// Info agrège les versions pour la sortie `kanjo version --format json`.
type Info struct {
	Tool      string `json:"tool"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	Rules     string `json:"rules"`
	Schema    string `json:"schema"`
}

// Get renvoie l'agrégat des versions courantes.
func Get() Info {
	return Info{Tool: Tool, Commit: Commit, BuildDate: BuildDate, Rules: Rules, Schema: Schema}
}
