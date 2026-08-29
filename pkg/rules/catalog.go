package rules

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/internal/version"
)

// Markdown génère le catalogue des règles enregistrées, groupé par jeu. C'est la source de
// docs/rules.md (§8.2) : une règle non documentée est une règle absente du registre, ce qui
// fait échouer la génération et donc la CI.
func Markdown() string {
	var b strings.Builder
	b.WriteString("# Catalogue des règles de validation — Kanjō\n\n")
	b.WriteString("> Fichier **généré** depuis le registre des règles (`pkg/rules`).\n")
	b.WriteString("> Ne pas éditer à la main. Régénérer avec `KANJO_REGEN=1 go test ./pkg/rules/...`.\n\n")
	b.WriteString(fmt.Sprintf("Version du jeu de règles : **%s**\n\n", version.Rules))

	bySet := map[string][]Rule{}
	var order []string
	for _, r := range All() {
		if _, ok := bySet[r.Set]; !ok {
			order = append(order, r.Set)
		}
		bySet[r.Set] = append(bySet[r.Set], r)
	}

	for _, set := range order {
		rs := bySet[set]
		b.WriteString(fmt.Sprintf("## %s (%d règles)\n\n", set, len(rs)))
		b.WriteString("| ID | Gravité | Termes | Message |\n")
		b.WriteString("|----|---------|--------|---------|\n")
		for _, r := range rs {
			msg := r.Message["fr"]
			if msg == "" {
				msg = r.Message["en"]
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				r.ID, r.Severity.String(), strings.Join(r.Terms, ", "), escapePipe(msg)))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func escapePipe(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
