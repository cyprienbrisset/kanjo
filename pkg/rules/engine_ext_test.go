package rules_test

import (
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"

	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all" // enregistre tous les jeux de règles
)

func TestRegistryPopulated(t *testing.T) {
	if len(rules.All()) < 50 {
		t.Errorf("registre = %d règles, attendu ≥ 50", len(rules.All()))
	}
	sets := strings.Join(rules.Sets(), ",")
	for _, want := range []string{"en16931", "cius.fr", "kanjo"} {
		if !strings.Contains(sets, want) {
			t.Errorf("jeu %q absent des jeux enregistrés (%s)", want, sets)
		}
	}
}

func TestMarkdownCatalog(t *testing.T) {
	md := rules.Markdown()
	if !strings.Contains(md, "BR-") || len(md) < 500 {
		t.Errorf("catalogue Markdown inattendu (len=%d)", len(md))
	}
}

func TestValidateSetFiltering(t *testing.T) {
	d := model.NewDocument(model.KindInvoice) // document quasi vide → anomalies garanties
	full := rules.Validate(d)
	only := rules.Validate(d, "en16931")
	if only.RulesRun == 0 {
		t.Fatal("aucune règle exécutée avec le filtre en16931")
	}
	if only.RulesRun >= full.RulesRun {
		t.Errorf("le filtre en16931 (%d) devrait exécuter moins de règles que tous les jeux (%d)", only.RulesRun, full.RulesRun)
	}
	for _, s := range only.RuleSets {
		if s != "en16931" {
			t.Errorf("jeu inattendu exécuté sous filtre : %s", s)
		}
	}
}

func TestReportSeverity(t *testing.T) {
	d := model.NewDocument(model.KindInvoice) // manque numéro, dates, parties… → erreurs
	rep := rules.Validate(d, "en16931")
	if !rep.HasErrors() {
		t.Error("un document vide devrait produire des erreurs")
	}
	if rep.Worst() != rules.SeverityError {
		t.Errorf("worst = %v, veut error", rep.Worst())
	}
	if len(rep.Findings) == 0 {
		t.Error("des anomalies étaient attendues")
	}
}

func TestValidateUnknownSet(t *testing.T) {
	rep := rules.Validate(model.NewDocument(model.KindInvoice), "n-existe-pas")
	if rep.RulesRun != 0 {
		t.Errorf("un jeu inconnu ne devrait exécuter aucune règle, got %d", rep.RulesRun)
	}
	// Fail-closed (§17.7) : un jeu inconnu ne doit PAS aboutir à un verdict conforme silencieux.
	// Le moteur émet désormais une anomalie fatale plutôt que de rester sans erreur.
	if !rep.HasErrors() {
		t.Error("un jeu de règles inconnu doit produire une anomalie bloquante (fail-closed)")
	}
	found := false
	for _, f := range rep.Findings {
		if f.RuleID == rules.RuleUnknownSet {
			found = true
		}
	}
	if !found {
		t.Errorf("anomalie %q attendue pour un jeu inconnu", rules.RuleUnknownSet)
	}
}
