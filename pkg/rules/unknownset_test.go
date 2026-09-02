package rules_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"

	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

// TestValidateUnknownSetFailsClosed : demander un jeu de règles inexistant ne doit JAMAIS produire
// un « conforme » silencieux — le moteur émet une anomalie fatale (§17.7).
func TestValidateUnknownSetFailsClosed(t *testing.T) {
	doc := &model.Document{CurrencyCode: "EUR"}

	rep := rules.Validate(doc, "jeu-qui-nexiste-pas")
	if !rep.HasErrors() {
		t.Fatal("un jeu de règles inconnu doit produire une anomalie bloquante (fail-closed)")
	}
	found := false
	for _, f := range rep.Findings {
		if f.RuleID == rules.RuleUnknownSet && f.Severity == rules.SeverityFatal {
			found = true
		}
	}
	if !found {
		t.Fatalf("anomalie fatale %q attendue, findings=%+v", rules.RuleUnknownSet, rep.Findings)
	}
}

// TestValidateKnownSetNotFlagged : un jeu de règles réel n'est jamais signalé comme inconnu.
func TestValidateKnownSetNotFlagged(t *testing.T) {
	doc := &model.Document{CurrencyCode: "EUR"}
	rep := rules.Validate(doc, "en16931")
	for _, f := range rep.Findings {
		if f.RuleID == rules.RuleUnknownSet {
			t.Fatalf("le jeu connu en16931 ne doit pas être signalé inconnu : %+v", f)
		}
	}
}

// TestUnknownSetsHelper : déduplication et tri, seuls les jeux réellement absents sont renvoyés.
func TestUnknownSetsHelper(t *testing.T) {
	got := rules.UnknownSets("en16931", "zzz", "aaa", "zzz")
	if len(got) != 2 || got[0] != "aaa" || got[1] != "zzz" {
		t.Fatalf("attendu [aaa zzz], obtenu %v", got)
	}
}
