package test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/rules"

	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

// parityFloor est un cliquet anti-régression : le nombre de règles canoniques EN 16931 couvertes
// ne doit jamais repasser sous ce seuil. À relever au fil des lots (jamais à baisser).
const parityFloor = 132

var (
	canonicalPath = filepath.Join("..", "testdata", "en16931", "canonical-rule-ids.txt")
	reportPath    = filepath.Join("..", "docs", "CONFORMITE-EN16931.md")
)

func loadCanonical(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("liste canonique introuvable (%s) : %v", canonicalPath, err)
	}
	var ids []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ids = append(ids, line)
	}
	sort.Strings(ids)
	return ids
}

// implementedEN renvoie l'ensemble des identifiants de règles du jeu « en16931 » enregistrés.
func implementedEN() map[string]bool {
	set := map[string]bool{}
	for _, r := range rules.All() {
		if r.Set == "en16931" {
			set[r.ID] = true
		}
	}
	return set
}

func parityStats(t *testing.T) (canon []string, covered, missing, beyond []string) {
	canon = loadCanonical(t)
	inCanon := map[string]bool{}
	for _, id := range canon {
		inCanon[id] = true
	}
	impl := implementedEN()
	for _, id := range canon {
		if impl[id] {
			covered = append(covered, id)
		} else {
			missing = append(missing, id)
		}
	}
	for id := range impl {
		if !inCanon[id] {
			beyond = append(beyond, id)
		}
	}
	sort.Strings(covered)
	sort.Strings(missing)
	sort.Strings(beyond)
	return
}

func TestEN16931Parity(t *testing.T) {
	canon, covered, missing, beyond := parityStats(t)
	pct := 100 * len(covered) / len(canon)
	t.Logf("Parité EN 16931 : %d / %d règles canoniques couvertes (%d %%)", len(covered), len(canon), pct)
	if len(missing) > 0 {
		t.Logf("Manquantes (%d) : %s", len(missing), strings.Join(missing, " "))
	}
	if len(beyond) > 0 {
		// Non bloquant : règles utiles implémentées au-delà du Schematron CEN de référence.
		t.Logf("Au-delà du référentiel CEN (%d) : %s", len(beyond), strings.Join(beyond, " "))
	}
	if len(covered) < parityFloor {
		t.Fatalf("RÉGRESSION de parité : %d règles couvertes < seuil %d", len(covered), parityFloor)
	}
}

func TestParityReportInSync(t *testing.T) {
	want := buildParityReport(t)
	if os.Getenv("KANJO_REGEN") != "" {
		if err := os.WriteFile(reportPath, []byte(want), 0o644); err != nil {
			t.Fatalf("écriture rapport : %v", err)
		}
		return
	}
	got, err := os.ReadFile(reportPath)
	if err != nil || string(got) != want {
		t.Fatalf("docs/CONFORMITE-EN16931.md désynchronisé. Régénérez : KANJO_REGEN=1 go test ./test/ -run TestParityReportInSync")
	}
}

func buildParityReport(t *testing.T) string {
	canon, covered, missing, beyond := parityStats(t)
	pct := 100 * len(covered) / len(canon)

	// Regroupement par famille pour la lisibilité.
	byFamily := func(ids []string) string {
		fam := map[string][]string{}
		var order []string
		for _, id := range ids {
			f := id[:strings.LastIndex(id, "-")]
			if _, ok := fam[f]; !ok {
				order = append(order, f)
			}
			fam[f] = append(fam[f], id)
		}
		sort.Strings(order)
		var b strings.Builder
		for _, f := range order {
			fmt.Fprintf(&b, "- **%s** (%d) : %s\n", f, len(fam[f]), strings.Join(fam[f], ", "))
		}
		return b.String()
	}

	var b strings.Builder
	b.WriteString("# Parité EN 16931 — couverture des règles\n\n")
	b.WriteString("> Fichier **généré** par `test/parity_test.go` à partir de la liste canonique\n")
	b.WriteString("> (`testdata/en16931/canonical-rule-ids.txt`, extraite du Schematron officiel CEN).\n")
	b.WriteString("> Ne pas éditer à la main. Régénérer : `KANJO_REGEN=1 go test ./test/ -run TestParityReportInSync`.\n\n")
	fmt.Fprintf(&b, "## Couverture : %d / %d règles canoniques (%d %%)\n\n", len(covered), len(canon), pct)
	fmt.Fprintf(&b, "- Règles couvertes : **%d**\n- Règles manquantes : **%d**\n", len(covered), len(missing))
	fmt.Fprintf(&b, "- Règles implémentées au-delà du Schematron CEN de référence : **%d**\n\n", len(beyond))
	b.WriteString("## Règles couvertes\n\n")
	b.WriteString(byFamily(covered))
	b.WriteString("\n## Règles manquantes (feuille de route)\n\n")
	b.WriteString(byFamily(missing))
	if len(beyond) > 0 {
		b.WriteString("\n## Au-delà du référentiel CEN\n\n")
		b.WriteString("Règles utiles implémentées par Kanjō mais absentes du Schematron CEN de référence :\n\n")
		b.WriteString(byFamily(beyond))
	}
	return b.String()
}
