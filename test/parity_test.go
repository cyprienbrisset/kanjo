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
const parityFloor = 223

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

// familyLabel décrit chaque famille de règles EN 16931 (libellé métier lisible).
var familyLabel = map[string]string{
	"BR":     "Présence & structure des informations obligatoires",
	"BR-CO":  "Calculs et cohérence des totaux",
	"BR-DEC": "Nombre de décimales des montants",
	"BR-CL":  "Conformité aux listes de codes officielles",
	"BR-S":   "TVA — taux normal (Standard rated)",
	"BR-Z":   "TVA — taux zéro (Zero rated)",
	"BR-E":   "TVA — exonérée (Exempt)",
	"BR-AE":  "TVA — autoliquidation (Reverse charge)",
	"BR-G":   "TVA — export hors UE",
	"BR-IC":  "TVA — livraison intracommunautaire",
	"BR-O":   "TVA — hors champ d'application",
	"BR-B":   "TVA — split payment (régime italien)",
	"BR-AF":  "TVA — régime régional IGIC (Canaries)",
	"BR-AG":  "TVA — régime régional IPSI (Ceuta & Melilla)",
	"BR-IG":  "TVA — IGIC (variante)",
	"BR-IP":  "TVA — IPSI (variante)",
}

// familyOf renvoie le préfixe de famille d'un identifiant (ex. BR-CL-06 → BR-CL, BR-01 → BR).
func familyOf(id string) string {
	i := strings.LastIndex(id, "-")
	if i < 0 {
		return id
	}
	// Un suffixe numérique (BR-01) ou alphanumérique de rang (BR-CL-06) : la famille exclut le dernier segment.
	return id[:i]
}

func buildParityReport(t *testing.T) string {
	canon, covered, missing, beyond := parityStats(t)
	pct := 100 * len(covered) / len(canon)

	// Indexation par famille (ordre canonique + comptage couvert/total).
	type famStat struct {
		total, done int
		coveredIDs  []string
		missingIDs  []string
	}
	stats := map[string]*famStat{}
	var order []string
	get := func(f string) *famStat {
		if _, ok := stats[f]; !ok {
			stats[f] = &famStat{}
			order = append(order, f)
		}
		return stats[f]
	}
	coveredSet := map[string]bool{}
	for _, id := range covered {
		coveredSet[id] = true
	}
	for _, id := range canon {
		f := familyOf(id)
		s := get(f)
		s.total++
		if coveredSet[id] {
			s.done++
			s.coveredIDs = append(s.coveredIDs, id)
		} else {
			s.missingIDs = append(s.missingIDs, id)
		}
	}
	sort.Strings(order)

	label := func(f string) string {
		if l, ok := familyLabel[f]; ok {
			return l
		}
		return "—"
	}

	var b strings.Builder
	b.WriteString("# Rapport de conformité EN 16931\n\n")
	b.WriteString("**Kanjō** — moteur de validation natif Go des factures électroniques.  \n")
	b.WriteString("_Document généré automatiquement — ne pas éditer à la main._\n\n")
	b.WriteString("---\n\n")

	// 1. Synthèse
	b.WriteString("## 1. Synthèse\n\n")
	verdict := "Couverture partielle"
	if len(missing) == 0 {
		verdict = "**Couverture intégrale du référentiel CEN**"
	}
	fmt.Fprintf(&b, "| Indicateur | Valeur |\n|---|---|\n")
	fmt.Fprintf(&b, "| Règles canoniques (Schematron CEN) | **%d** |\n", len(canon))
	fmt.Fprintf(&b, "| Règles couvertes | **%d** |\n", len(covered))
	fmt.Fprintf(&b, "| Règles manquantes | **%d** |\n", len(missing))
	fmt.Fprintf(&b, "| **Taux de couverture** | **%d %%** |\n", pct)
	fmt.Fprintf(&b, "| Règles au-delà du référentiel CEN | %d |\n", len(beyond))
	fmt.Fprintf(&b, "| Verdict | %s |\n\n", verdict)

	// 2. Couverture par famille (tableau)
	b.WriteString("## 2. Couverture par famille\n\n")
	b.WriteString("| Famille | Description | Couverture |\n|---|---|:--:|\n")
	for _, f := range order {
		s := stats[f]
		mark := "✅"
		if s.done < s.total {
			mark = fmt.Sprintf("%d/%d", s.done, s.total)
		} else {
			mark = fmt.Sprintf("%d/%d ✅", s.done, s.total)
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", f, label(f), mark)
	}
	total := "✅"
	if len(missing) > 0 {
		total = fmt.Sprintf("%d/%d", len(covered), len(canon))
	} else {
		total = fmt.Sprintf("**%d/%d** ✅", len(covered), len(canon))
	}
	fmt.Fprintf(&b, "| **Total** | **%d familles** | %s |\n\n", len(order), total)

	// 3. Méthode
	b.WriteString("## 3. Méthode de mesure\n\n")
	b.WriteString("La conformité n'est pas *déclarée*, elle est **mesurée à chaque commit** :\n\n")
	b.WriteString("1. **Source de vérité** — la liste canonique des règles est extraite des **Schematron officiels du CEN** " +
		"(UBL + CII préprocessés, dépôt `ConnectingEurope/eInvoicing-EN16931`), vendorée dans " +
		"[`testdata/en16931/canonical-rule-ids.txt`](../testdata/en16931/canonical-rule-ids.txt).\n")
	b.WriteString("2. **Comparaison automatique** — [`test/parity_test.go`](../test/parity_test.go) confronte les règles " +
		"réellement enregistrées par le moteur à cette liste, et régénère le présent rapport.\n")
	b.WriteString("3. **Cliquet anti-régression** — un plancher testé en CI interdit toute baisse de couverture.\n")
	b.WriteString("4. **Aucun stub** — chaque règle est réellement calculée, dotée d'un test **passant** et d'un test " +
		"**échouant**, et éprouvée sur un corpus réel (exemples officiels CEN + corpus publié) **sans faux positif**.\n\n")

	// Numérotation dynamique des sections suivantes (pas de trou si une section est vide).
	sec := 4

	// Détail des règles couvertes — replié par défaut (évite le mur de texte).
	fmt.Fprintf(&b, "## %d. Détail des règles couvertes\n\n", sec)
	sec++
	b.WriteString("<details>\n")
	fmt.Fprintf(&b, "<summary>Afficher les %d règles couvertes, par famille</summary>\n\n", len(covered))
	for _, f := range order {
		s := stats[f]
		if len(s.coveredIDs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "#### `%s` — %s (%d)\n\n", f, label(f), len(s.coveredIDs))
		fmt.Fprintf(&b, "```\n%s\n```\n\n", strings.Join(s.coveredIDs, ", "))
	}
	b.WriteString("</details>\n\n")

	// Règles manquantes (uniquement s'il en reste).
	if len(missing) > 0 {
		fmt.Fprintf(&b, "## %d. Règles manquantes (feuille de route)\n\n", sec)
		sec++
		b.WriteString("| Famille | Manquantes |\n|---|---|\n")
		for _, f := range order {
			s := stats[f]
			if len(s.missingIDs) == 0 {
				continue
			}
			fmt.Fprintf(&b, "| `%s` | %s |\n", f, strings.Join(s.missingIDs, ", "))
		}
		b.WriteString("\n")
	}

	// Règles au-delà du référentiel.
	if len(beyond) > 0 {
		fmt.Fprintf(&b, "## %d. Règles au-delà du référentiel CEN\n\n", sec)
		b.WriteString("Règles utiles implémentées par Kanjō mais absentes du Schematron CEN de référence :\n\n")
		famBeyond := map[string][]string{}
		var ob []string
		for _, id := range beyond {
			f := familyOf(id)
			if _, ok := famBeyond[f]; !ok {
				ob = append(ob, f)
			}
			famBeyond[f] = append(famBeyond[f], id)
		}
		sort.Strings(ob)
		b.WriteString("| Famille | Règles |\n|---|---|\n")
		for _, f := range ob {
			fmt.Fprintf(&b, "| `%s` | %s |\n", f, strings.Join(famBeyond[f], ", "))
		}
		b.WriteString("\n")
	}

	// Annexe
	b.WriteString("---\n\n")
	b.WriteString("### Reproductibilité\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Régénérer ce rapport\n")
	b.WriteString("KANJO_REGEN=1 go test ./test/ -run TestParityReportInSync\n")
	b.WriteString("# Re-extraire la liste canonique depuis le Schematron CEN\n")
	b.WriteString("scripts/extraire-regles-canoniques.sh\n")
	b.WriteString("```\n")
	return b.String()
}
