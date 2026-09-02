// Package rules est le moteur de validation de Kanjō. Comme pkg/model, il n'importe QUE la
// bibliothèque standard (et pkg/model), sans I/O — 100 % testable et réutilisable (§4.2).
//
// IMPORTANT (§17.7) : un verdict de conformité n'est jamais produit sans avoir été
// effectivement calculé. Les règles ci-dessous calculent réellement leurs contrôles.
package rules

import (
	"sort"
	"sync"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// Severity est la gravité d'une anomalie.
type Severity int

const (
	SeverityInfo    Severity = iota // information
	SeverityWarning                 // avertissement
	SeverityError                   // non conforme
	SeverityFatal                   // document illisible / non traitable
)

// String rend la gravité en identifiant stable (utilisé dans les sorties JSON).
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// Context porte l'état partagé lors de l'exécution d'un jeu de règles sur un document.
type Context struct {
	Currency string
}

// Finding est une anomalie détectée par une règle (§8.1).
type Finding struct {
	RuleID     string   `json:"ruleId"`
	Severity   Severity `json:"severity"`
	Message    string   `json:"message"`
	Term       string   `json:"term,omitempty"`
	Path       string   `json:"path,omitempty"`
	SourcePath string   `json:"sourcePath,omitempty"`
	Expected   string   `json:"expected,omitempty"`
	Actual     string   `json:"actual,omitempty"`
	Fixable    bool     `json:"fixable"`
}

// Rule est une règle de validation (§8.1).
// SetOrderX est le jeu de règles applicable aux bons de commande Order-X (Kind=order). Il est
// mutuellement exclusif des jeux facture : le moteur applique l'un OU l'autre selon le type du document.
const SetOrderX = "orderx"

type Rule struct {
	ID       string
	Set      string // "en16931" | "cius.fr" | "xrechnung" | "peppol" | "kanjo" | "orderx"
	Severity Severity
	Terms    []string
	Message  map[string]string // "fr", "en"
	Check    func(*model.Document, *Context) []Finding
}

var (
	mu       sync.RWMutex
	registry = map[string]Rule{}
)

// Register enregistre une règle. Appelé par les sous-paquets (en16931, cius/fr…) dans init().
// Panique en cas d'identifiant dupliqué : c'est un bogue de programmation.
func Register(r Rule) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[r.ID]; exists {
		panic("rules: identifiant de règle dupliqué : " + r.ID)
	}
	registry[r.ID] = r
}

// All renvoie toutes les règles enregistrées, triées par identifiant.
func All() []Rule {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Rule, 0, len(registry))
	for _, r := range registry {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Sets renvoie la liste triée des jeux de règles enregistrés.
func Sets() []string {
	mu.RLock()
	defer mu.RUnlock()
	seen := map[string]bool{}
	for _, r := range registry {
		seen[r.Set] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// UnknownSets renvoie, parmi les jeux demandés, ceux qui ne correspondent à aucun jeu enregistré
// (dédupliqués, triés). Sert au fail-closed : demander un jeu inexistant ne doit jamais aboutir à
// un verdict « conforme » silencieux (§17.7).
func UnknownSets(requested ...string) []string {
	known := map[string]bool{}
	for _, s := range Sets() {
		known[s] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, s := range requested {
		if !known[s] && !seen[s] {
			out = append(out, s)
			seen[s] = true
		}
	}
	sort.Strings(out)
	return out
}

// RuleUnknownSet est l'identifiant de l'anomalie émise lorsqu'un jeu de règles inexistant est demandé.
const RuleUnknownSet = "KANJO-SET-UNKNOWN"

// Report agrège les anomalies d'une validation.
type Report struct {
	Findings    []Finding `json:"findings"`
	RuleSets    []string  `json:"ruleSets"`    // jeux effectivement exécutés
	RulesRun    int       `json:"rulesRun"`    // nombre de règles exécutées
	PDFAChecked bool      `json:"pdfaChecked"` // §9.2 : jamais simulé
}

// Worst renvoie la gravité maximale rencontrée.
func (r Report) Worst() Severity {
	worst := SeverityInfo
	for _, f := range r.Findings {
		if f.Severity > worst {
			worst = f.Severity
		}
	}
	return worst
}

// HasErrors indique la présence d'au moins une anomalie de gravité ≥ Error.
func (r Report) HasErrors() bool { return r.Worst() >= SeverityError }

// Validate exécute les règles des jeux demandés sur le document. Si sets est vide, tous les
// jeux enregistrés sont exécutés. Le rapport indique explicitement ce qui a été exécuté
// (aucune conformité n'est affirmée au-delà des règles réellement lancées, §17.7).
func Validate(doc *model.Document, sets ...string) Report {
	want := map[string]bool{}
	for _, s := range sets {
		want[s] = true
	}
	ctx := &Context{Currency: doc.CurrencyCode}
	rep := Report{}
	setSeen := map[string]bool{}

	// Fail-closed (§17.7) : un jeu de règles explicitement demandé mais inexistant ne doit jamais
	// aboutir à un verdict « conforme » silencieux. On émet une anomalie FATALE — distincte du cas
	// « aucun jeu demandé » (sets vide → tous les jeux exécutés), qui reste légitime.
	for _, s := range UnknownSets(sets...) {
		rep.Findings = append(rep.Findings, Finding{
			RuleID:   RuleUnknownSet,
			Severity: SeverityFatal,
			Message:  "jeu de règles inconnu : " + s,
			Actual:   s,
		})
	}

	// Sélection consciente du type de document : une commande (Order-X) est validée par le jeu
	// « orderx » uniquement ; les règles facture EN 16931 (TVA, totaux…) ne s'y appliquent pas et
	// produiraient des faux positifs. Réciproquement, une facture n'exécute jamais les règles Order-X.
	isOrder := doc.Kind == model.KindOrder

	for _, r := range All() {
		if len(want) > 0 && !want[r.Set] {
			continue
		}
		if isOrder != (r.Set == SetOrderX) {
			continue
		}
		setSeen[r.Set] = true
		rep.RulesRun++
		if r.Check == nil {
			continue
		}
		for _, f := range r.Check(doc, ctx) {
			if f.RuleID == "" {
				f.RuleID = r.ID
			}
			if f.Severity == SeverityInfo && r.Severity != SeverityInfo {
				f.Severity = r.Severity
			}
			rep.Findings = append(rep.Findings, f)
		}
	}
	for s := range setSeen {
		rep.RuleSets = append(rep.RuleSets, s)
	}
	sort.Strings(rep.RuleSets)
	sort.SliceStable(rep.Findings, func(i, j int) bool {
		if rep.Findings[i].Severity != rep.Findings[j].Severity {
			return rep.Findings[i].Severity > rep.Findings[j].Severity
		}
		return rep.Findings[i].RuleID < rep.Findings[j].RuleID
	})
	return rep
}
