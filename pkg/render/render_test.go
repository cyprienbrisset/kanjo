package render_test

import (
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/api"
	"github.com/cyprienbrisset/kanjo/pkg/generate"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/render"
)

// stripSeps retire les espaces séparateurs (ordinaire, insécable, fine insécable) pour
// comparer les montants indépendamment du type d'espace employé.
func stripSeps(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 0x20, 0xA0, 0x202F: // espace, insécable, fine insécable
			return -1
		default:
			return r
		}
	}, s)
}

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		val, cur string
		lang     model.Lang
		want     string
	}{
		// La convention française utilise l'espace fine insécable (U+202F) comme séparateur
		// de milliers et avant le symbole monétaire (§12.9) ; la comparaison ignore le type d'espace.
		{"1250.00", "EUR", model.LangFR, "1250,00€"},
		{"999.90", "EUR", model.LangFR, "999,90€"},
		{"1250000.50", "EUR", model.LangFR, "1250000,50€"},
		{"-10.00", "EUR", model.LangFR, "-10,00€"},
		{"1250.00", "USD", model.LangEN, "$1,250.00"},
	}
	for _, c := range cases {
		got := stripSeps(render.FormatMoney(model.MustParseAmount(c.val, c.cur), c.lang))
		if got != stripSeps(c.want) {
			t.Errorf("FormatMoney(%s %s, %s) = %q, veut %q", c.val, c.cur, c.lang, got, c.want)
		}
	}
}

func TestFormatMoneyUsesNarrowNoBreakSpace(t *testing.T) {
	// Vérifie explicitement l'espace fine insécable (U+202F) en français.
	got := render.FormatMoney(model.MustParseAmount("1250.00", "EUR"), model.LangFR)
	if !strings.ContainsRune(got, ' ') {
		t.Errorf("le format français doit utiliser U+202F, obtenu %q", got)
	}
}

func TestRenderInvoiceHTML(t *testing.T) {
	doc, _ := generate.Generate(0, generate.Options{Scenario: generate.ScenarioSimple, Seed: 1})
	html, err := render.RenderInvoiceHTML(doc, model.LangFR, "ok")
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	for _, want := range []string{"<!doctype html>", doc.ID, doc.Seller.Name, "Total TTC", "conforme", "Kanjō"} {
		if !strings.Contains(s, want) {
			t.Errorf("le HTML ne contient pas %q", want)
		}
	}
	if strings.Contains(s, "http://") || strings.Contains(s, "https://") {
		t.Error("le HTML rendu référence une ressource réseau (doit être autonome)")
	}
}

func TestRenderValidationReport(t *testing.T) {
	env := &api.Envelope{
		StartedAt: "2026-08-29T10:00:00Z",
		Summary:   api.Summary{Total: 2, OK: 1, Error: 1},
		Results: []api.Result{
			{Input: "/x/F1.xml", Status: api.StatusOK},
			{Input: "/x/F2.xml", Status: api.StatusError, Findings: []api.Finding{
				{RuleID: "BR-CO-15", Severity: "error", Message: "TTC incohérent"},
			}},
		},
	}
	html, err := render.RenderValidationReportHTML(env, "2026.3")
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	for _, want := range []string{"Rapport de validation", "BR-CO-15", "TTC incohérent", "F2.xml", "2026.3"} {
		if !strings.Contains(s, want) {
			t.Errorf("le rapport ne contient pas %q", want)
		}
	}
}
