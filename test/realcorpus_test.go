package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/read"

	_ "github.com/cyprienbrisset/kanjo/pkg/formats"
)

// officialDir est le corpus RÉEL open-source, téléchargé par testdata/corpus/fetch.sh.
// Il n'est pas versionné (licences) : les tests ci-dessous se sautent en son absence (CI),
// mais s'exécutent en local après un `fetch.sh` pour éprouver les lecteurs sur > 500 documents
// réels (CEN, XRechnung, Peppol).
var officialDir = filepath.Join("..", "testdata", "corpus", "official")

func officialFiles(t *testing.T) []string {
	t.Helper()
	if _, err := os.Stat(officialDir); err != nil {
		t.Skipf("corpus réel absent (%s) — lancez testdata/corpus/fetch.sh", officialDir)
	}
	return walkXML(t, officialDir)
}

// readNoPanic lit un fichier en convertissant tout panic éventuel en erreur : les lecteurs ne
// doivent JAMAIS paniquer, même sur des entrées réelles ou malformées (§conception, robustesse).
func readNoPanic(data []byte, name string) (res *read.Result, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	res, _ = read.ReadBytes(data, name)
	return res, false
}

// TestRealCorpusNoPanic garantit qu'aucun document réel ne fait paniquer les lecteurs.
func TestRealCorpusNoPanic(t *testing.T) {
	files := officialFiles(t)
	if len(files) == 0 {
		t.Skip("aucun fichier dans le corpus réel")
	}
	var panics []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if _, panicked := readNoPanic(data, f); panicked {
			panics = append(panics, f)
		}
	}
	if len(panics) > 0 {
		t.Fatalf("%d document(s) ont fait paniquer un lecteur :\n%s", len(panics), strings.Join(panics, "\n"))
	}
	t.Logf("robustesse : %d documents réels lus sans panic", len(files))
}

// TestRealExamplesParse garantit que les factures d'exemple CEN COMPLÈTES (UBL et CII) se lisent
// sans erreur. Le corpus phive-rules, hétérogène (autres standards, cas volontairement invalides,
// fragments), n'est PAS soumis à cette exigence : il sert la robustesse (cf. TestRealCorpusNoPanic).
func TestRealExamplesParse(t *testing.T) {
	all := officialFiles(t)
	var examples []string
	for _, f := range all {
		if strings.Contains(f, "ubl-examples") || strings.Contains(f, "cii-examples") {
			examples = append(examples, f)
		}
	}
	if len(examples) == 0 {
		t.Skip("aucun exemple CEN dans le corpus réel")
	}
	var failed int
	for _, f := range examples {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if _, err := read.ReadBytes(data, f); err != nil {
			failed++
			t.Errorf("exemple CEN non lisible %s : %v", filepath.Base(f), err)
		}
	}
	t.Logf("exemples CEN : %d lus, %d en échec", len(examples)-failed, failed)
}

// TestRealCorpusReadRate mesure (à titre informatif) la proportion du corpus réel qui se lit comme
// document pivot. Le reste correspond à des XML d'autres standards ou non-factures présents dans les
// dépôts de test agrégés. Aucune régression n'est tolérée sous un plancher prudent.
func TestRealCorpusReadRate(t *testing.T) {
	files := officialFiles(t)
	if len(files) == 0 {
		t.Skip("corpus réel vide")
	}
	readable := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if _, err := read.ReadBytes(data, f); err == nil {
			readable++
		}
	}
	pct := 100 * readable / len(files)
	t.Logf("lisibilité du corpus réel : %d / %d documents lus comme facture (%d %%)", readable, len(files), pct)
	if readable < 3000 {
		t.Errorf("régression de lisibilité : %d documents lus (< plancher 3000)", readable)
	}
}
