package render

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"

	"github.com/cyprienbrisset/kanjo/pkg/api"
)

const reportCSS = designCSS + `
.summary{display:flex;gap:32px;margin:16px 0 24px}
.summary .n{font-family:"IBM Plex Mono",monospace;font-size:24px}
.rule{border-left:3px solid var(--rule-firm);padding:8px 16px;margin:12px 0;background:var(--paper-0)}
.rule.err{border-left-color:var(--beni)}
.rule.warn{border-left-color:var(--kohaku)}
.rule .id{font-family:"IBM Plex Mono",monospace;font-weight:400;color:var(--ink-700)}
.rule .count{color:var(--ink-500);font-size:13px}
.rule .msg{color:var(--ink-900);margin:4px 0}
.rule .docs{font-family:"IBM Plex Mono",monospace;font-size:12px;color:var(--ink-500)}
`

type ruleGroup struct {
	ID       string
	Severity string
	Message  string
	Docs     []string
	IsError  bool
	IsWarn   bool
}

type reportView struct {
	Title        string
	StartedAt    string
	RulesVersion string
	Summary      api.Summary
	Groups       []ruleGroup
	CSS          template.CSS
}

var reportTmpl = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="fr"><head><meta charset="utf-8"><title>{{.Title}}</title><style>{{.CSS}}</style></head>
<body><div class="sheet">
 <h1>Rapport de validation</h1>
 <div class="meta">{{.Summary.Total}} documents · {{.StartedAt}} · jeu de règles {{.RulesVersion}}</div>
 <div class="summary">
  <div><div class="n" style="color:var(--koke)">適 {{.Summary.OK}}</div>conformes</div>
  <div><div class="n" style="color:var(--kohaku)">保 {{.Summary.Warning}}</div>réserve</div>
  <div><div class="n" style="color:var(--beni)">否 {{.Summary.Error}}</div>non conformes</div>
 </div>
 {{if .Groups}}<h2 style="font-family:'Shippori Mincho B1',serif;font-weight:400;color:var(--ink-700)">Par règle</h2>
 {{range .Groups}}<div class="rule{{if .IsError}} err{{else if .IsWarn}} warn{{end}}">
  <span class="id">{{.ID}}</span> <span class="count">総 {{len .Docs}} document(s) · {{.Severity}}</span>
  <div class="msg">{{.Message}}</div>
  <div class="docs">{{range .Docs}}{{.}} {{end}}</div>
 </div>{{end}}
 {{else}}<p style="color:var(--koke)">Aucune anomalie. 適</p>{{end}}
 <div class="foot">Rapport généré par Kanjō. Empreinte des règles : {{.RulesVersion}}.</div>
</div></body></html>`))

// RenderValidationReportHTML produit un rapport HTML autonome (imprimable, sans réseau),
// groupé par règle (§G4), destiné à être joint à un dossier de conformité.
func RenderValidationReportHTML(env *api.Envelope, rulesVersion string) ([]byte, error) {
	groups := map[string]*ruleGroup{}
	var order []string
	for _, r := range env.Results {
		doc := baseName(r.Input)
		for _, f := range r.Findings {
			g, ok := groups[f.RuleID]
			if !ok {
				g = &ruleGroup{
					ID: f.RuleID, Severity: f.Severity, Message: f.Message,
					IsError: f.Severity == "error" || f.Severity == "fatal",
					IsWarn:  f.Severity == "warning",
				}
				groups[f.RuleID] = g
				order = append(order, f.RuleID)
			}
			g.Docs = append(g.Docs, doc)
		}
	}
	// Tri : erreurs d'abord, puis par identifiant.
	sort.SliceStable(order, func(i, j int) bool {
		gi, gj := groups[order[i]], groups[order[j]]
		if gi.IsError != gj.IsError {
			return gi.IsError
		}
		return gi.ID < gj.ID
	})
	var gs []ruleGroup
	for _, id := range order {
		gs = append(gs, *groups[id])
	}

	v := reportView{
		Title: "Rapport de validation — Kanjō", StartedAt: env.StartedAt,
		RulesVersion: rulesVersion, Summary: env.Summary, Groups: gs, CSS: template.CSS(reportCSS),
	}
	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, v); err != nil {
		return nil, fmt.Errorf("rendu du rapport HTML: %w", err)
	}
	return buf.Bytes(), nil
}

// baseName renvoie le nom de fichier d'un chemin.
func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}
