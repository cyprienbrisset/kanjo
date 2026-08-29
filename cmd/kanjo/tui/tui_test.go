package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cyprienbrisset/kanjo/pkg/api"
)

func sampleModel() model {
	return model{
		path:   "/factures",
		width:  100,
		height: 30,
		results: []api.Result{
			{Input: "/factures/F1.xml", Status: api.StatusOK, Format: "cii"},
			{Input: "/factures/F2.xml", Status: api.StatusError, Format: "ubl",
				Findings: []api.Finding{{RuleID: "BR-CO-15", Severity: "error", Message: "TTC incohérent"}}},
			{Input: "/factures/F3.xml", Status: api.StatusWarning, Format: "cii"},
		},
		summary: api.Summary{Total: 3, OK: 1, Warning: 1, Error: 1},
	}
}

func TestViewRendersChromeAndSeals(t *testing.T) {
	v := sampleModel().View()
	for _, want := range []string{"KANJŌ", "/factures", "✓", "✕", "▲", "F1.xml"} {
		if !strings.Contains(v, want) {
			t.Errorf("la vue ne contient pas %q", want)
		}
	}
}

func TestViewShowsFindingsForSelected(t *testing.T) {
	m := sampleModel()
	m.cursor = 1 // F2, en erreur
	v := m.View()
	if !strings.Contains(v, "BR-CO-15") || !strings.Contains(v, "TTC incohérent") {
		t.Errorf("le détail ne montre pas l'anomalie du document sélectionné :\n%s", v)
	}
}

func TestNavigationKeys(t *testing.T) {
	m := sampleModel()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if nm.(model).cursor != 1 {
		t.Errorf("« j » doit descendre le curseur, obtenu %d", nm.(model).cursor)
	}
	nm2, _ := nm.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if nm2.(model).cursor != 0 {
		t.Errorf("« k » doit remonter le curseur, obtenu %d", nm2.(model).cursor)
	}
}

func TestSmallTerminalMessage(t *testing.T) {
	m := sampleModel()
	m.width, m.height = 40, 10
	if !strings.Contains(m.View(), "trop petit") {
		t.Error("un terminal trop petit doit afficher un message explicite")
	}
}

func TestValidatedMsgUpdatesState(t *testing.T) {
	m := model{loading: true}
	nm, _ := m.Update(validatedMsg{
		results: []api.Result{{Input: "a", Status: api.StatusOK}},
		summary: api.Summary{Total: 1, OK: 1},
	})
	if nm.(model).loading || len(nm.(model).results) != 1 {
		t.Error("validatedMsg doit terminer le chargement et charger les résultats")
	}
}
