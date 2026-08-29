package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all" // charge tous les jeux pour un catalogue complet
)

// docPath est l'emplacement du catalogue généré.
func docPath() string { return filepath.Join("..", "..", "docs", "rules.md") }

// TestRulesDocInSync vérifie que docs/rules.md correspond au registre. Poser KANJO_REGEN=1
// régénère le fichier (§8.2 : catalogue généré, la CI échoue s'il est désynchronisé).
func TestRulesDocInSync(t *testing.T) {
	generated := rules.Markdown()

	if os.Getenv("KANJO_REGEN") != "" {
		if err := os.WriteFile(docPath(), []byte(generated), 0o644); err != nil {
			t.Fatalf("écriture de %s: %v", docPath(), err)
		}
		t.Logf("docs/rules.md régénéré (%d règles)", len(rules.All()))
		return
	}

	current, err := os.ReadFile(docPath())
	if err != nil {
		t.Fatalf("docs/rules.md introuvable — régénérez avec KANJO_REGEN=1 go test ./pkg/rules/... : %v", err)
	}
	if string(current) != generated {
		t.Errorf("docs/rules.md désynchronisé du registre. Régénérez : KANJO_REGEN=1 go test ./pkg/rules/...")
	}
}

// TestNoUndocumentedRule garantit qu'aucune règle n'a un message vide (documentation minimale).
func TestNoUndocumentedRule(t *testing.T) {
	for _, r := range rules.All() {
		if r.Message["fr"] == "" && r.Message["en"] == "" {
			t.Errorf("règle %s sans message (jeu %s)", r.ID, r.Set)
		}
	}
}
