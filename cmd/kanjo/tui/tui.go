// Package tui implémente l'interface texte de Kanjō (§13), bâtie sur Bubble Tea + Lipgloss.
// Elle consomme le même cœur que la CLI (pipeline, read, rules) : une seule implémentation,
// plusieurs façades.
package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cyprienbrisset/kanjo/pkg/api"
	_ "github.com/cyprienbrisset/kanjo/pkg/formats"
	"github.com/cyprienbrisset/kanjo/pkg/pipeline"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

// Palette 昼 Hiru approximée pour le terminal (§12.2).
var (
	colInk    = lipgloss.Color("#14243D")
	colInk500 = lipgloss.Color("#5A6B7D")
	colInk300 = lipgloss.Color("#8E9AA6")
	colKoke   = lipgloss.Color("#5E7A4A") // conforme 適
	colKohaku = lipgloss.Color("#B8862F") // réserve 保
	colBeni   = lipgloss.Color("#9E2B32") // non conforme 否
	colAsagi  = lipgloss.Color("#3F8A8B") // information / sélection
	colRule   = lipgloss.Color("#B9B2A2")
)

var (
	titleStyle  = lipgloss.NewStyle().Foreground(colInk).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(colInk300)
	labelStyle  = lipgloss.NewStyle().Foreground(colInk500)
	statusStyle = lipgloss.NewStyle().Foreground(colInk500)
	selStyle    = lipgloss.NewStyle().Foreground(colAsagi).Bold(true)
	borderStyle = lipgloss.NewStyle().Foreground(colRule)
)

// validatedMsg est émis lorsque la validation d'un lot est terminée.
type validatedMsg struct {
	results []api.Result
	summary api.Summary
}

type model struct {
	path    string
	width   int
	height  int
	loading bool
	results []api.Result
	cursor  int
	summary api.Summary
}

// Run lance la TUI sur un chemin (fichier ou dossier). Renvoie un code de sortie.
func Run(path string) int {
	if path == "" {
		path = "."
	}
	m := model{path: path, loading: true}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "erreur TUI : %v\n", err)
		return 1
	}
	return 0
}

func (m model) Init() tea.Cmd { return validateCmd(m.path) }

// validateCmd découvre et valide les fichiers du chemin, de façon asynchrone.
func validateCmd(path string) tea.Cmd {
	return func() tea.Msg {
		files, err := pipeline.Discover([]string{path}, true, nil, nil)
		if err != nil || len(files) == 0 {
			return validatedMsg{}
		}
		proc := func(p string) api.Result {
			res := api.Result{Input: p}
			data, err := os.ReadFile(p)
			if err != nil {
				res.Status, res.Error = api.StatusError, err.Error()
				return res
			}
			rd, err := read.ReadBytes(data, p)
			if err != nil {
				res.Status, res.Error = api.StatusError, err.Error()
				return res
			}
			res.Format = string(rd.Format)
			rep := rules.Validate(rd.Doc)
			for _, f := range rep.Findings {
				res.Findings = append(res.Findings, api.Finding{
					RuleID: f.RuleID, Severity: f.Severity.String(), Message: f.Message,
					Term: f.Term, Expected: f.Expected, Actual: f.Actual,
				})
			}
			switch {
			case rep.HasErrors():
				res.Status = api.StatusError
			case len(rep.Findings) > 0:
				res.Status = api.StatusWarning
			default:
				res.Status = api.StatusOK
			}
			return res
		}
		rep := pipeline.Run(files, proc, pipeline.Options{})
		return validatedMsg{results: rep.Results, summary: rep.Summary}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case validatedMsg:
		m.loading = false
		m.results = msg.results
		m.summary = msg.summary
		if m.cursor >= len(m.results) {
			m.cursor = 0
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "r":
			m.loading = true
			return m, validateCmd(m.path)
		}
	}
	return m, nil
}

func sealGlyph(status api.Status) string {
	switch status {
	case api.StatusOK:
		return lipgloss.NewStyle().Foreground(colKoke).Render("✓")
	case api.StatusWarning:
		return lipgloss.NewStyle().Foreground(colKohaku).Render("▲")
	case api.StatusError:
		return lipgloss.NewStyle().Foreground(colBeni).Render("✕")
	default:
		return dimStyle.Render("·")
	}
}

func (m model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.width < 80 || m.height < 24 {
		return "Terminal trop petit. Kanjō nécessite au moins 80 × 24."
	}
	var b strings.Builder

	// Barre de titre.
	b.WriteString(titleStyle.Render("KANJŌ"))
	b.WriteString(dimStyle.Render("  ·  " + m.path))
	b.WriteString("\n")
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	if m.loading {
		b.WriteString("\n  " + lipgloss.NewStyle().Foreground(colAsagi).Render("◴") + "  validation en cours…\n")
		b.WriteString(m.pad())
		b.WriteString(m.statusBar())
		return b.String()
	}
	if len(m.results) == 0 {
		b.WriteString("\n  Aucune facture trouvée. Indiquez un dossier : kanjo tui <dossier>\n")
		b.WriteString(m.pad())
		b.WriteString(m.statusBar())
		return b.String()
	}

	// Deux colonnes : liste (gauche) et détails (droite).
	listW := 34
	detailW := m.width - listW - 3
	bodyH := m.height - 5

	list := m.renderList(listW, bodyH)
	detail := m.renderDetail(detailW, bodyH)
	body := lipgloss.JoinHorizontal(lipgloss.Top, list, borderStyle.Render(" │ "), detail)
	b.WriteString(body)
	b.WriteString("\n")
	b.WriteString(m.statusBar())
	return b.String()
}

func (m model) renderList(w, h int) string {
	var rows []string
	start := 0
	if m.cursor >= h {
		start = m.cursor - h + 1
	}
	for i := start; i < len(m.results) && i < start+h; i++ {
		r := m.results[i]
		name := baseName(r.Input)
		if len(name) > w-6 {
			name = name[:w-7] + "…"
		}
		line := fmt.Sprintf("%s %s", sealGlyph(r.Status), name)
		if i == m.cursor {
			line = selStyle.Render("▸") + line
		} else {
			line = " " + line
		}
		rows = append(rows, line)
	}
	return lipgloss.NewStyle().Width(w).Render(strings.Join(rows, "\n"))
}

func (m model) renderDetail(w, h int) string {
	if m.cursor >= len(m.results) {
		return ""
	}
	r := m.results[m.cursor]
	var b strings.Builder
	b.WriteString(sealGlyph(r.Status) + " " + titleStyle.Render(baseName(r.Input)) + "\n")
	if r.Format != "" {
		b.WriteString(labelStyle.Render("format  ") + r.Format + "\n")
	}
	if r.Error != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(colBeni).Render(r.Error) + "\n")
	}
	if len(r.Findings) == 0 && r.Error == "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(colKoke).Render("Aucune anomalie.") + "\n")
	} else {
		b.WriteString("\n" + labelStyle.Render("Anomalies") + "\n")
		max := h - 6
		for i, f := range r.Findings {
			if i >= max {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d de plus", len(r.Findings)-max)) + "\n")
				break
			}
			mark := sealForSeverity(f.Severity)
			msg := f.Message
			if len(msg) > w-16 {
				msg = msg[:w-17] + "…"
			}
			b.WriteString(fmt.Sprintf("  %s %-10s %s\n", mark, f.RuleID, msg))
		}
	}
	return lipgloss.NewStyle().Width(w).Render(b.String())
}

func sealForSeverity(sev string) string {
	switch sev {
	case "error", "fatal":
		return lipgloss.NewStyle().Foreground(colBeni).Render("✕")
	case "warning":
		return lipgloss.NewStyle().Foreground(colKohaku).Render("▲")
	default:
		return dimStyle.Render("·")
	}
}

func (m model) statusBar() string {
	s := m.summary
	counters := fmt.Sprintf("%s %d   %s %d   %s %d",
		lipgloss.NewStyle().Foreground(colKoke).Render("✓"), s.OK,
		lipgloss.NewStyle().Foreground(colKohaku).Render("▲"), s.Warning,
		lipgloss.NewStyle().Foreground(colBeni).Render("✕"), s.Error)
	help := dimStyle.Render("[j/k] naviguer  [r] revalider  [q] quitter")
	gap := m.width - lipgloss.Width(counters) - lipgloss.Width(help)
	if gap < 1 {
		gap = 1
	}
	return statusStyle.Render(counters) + strings.Repeat(" ", gap) + help
}

func (m model) pad() string {
	lines := m.height - 6
	if lines < 0 {
		lines = 0
	}
	return strings.Repeat("\n", lines)
}

func baseName(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}
