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

// TestRealExamplesParse garantit que les factures COMPLÈTES des corpus d'exemples (par opposition
// aux fragments unitaires) se lisent sans erreur.
func TestRealExamplesParse(t *testing.T) {
	all := officialFiles(t)
	var examples []string
	for _, f := range all {
		if strings.Contains(f, "examples") || strings.Contains(f, "peppol-bis") {
			examples = append(examples, f)
		}
	}
	if len(examples) == 0 {
		t.Skip("aucun fichier d'exemple dans le corpus réel")
	}
	var failed int
	for _, f := range examples {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if _, err := read.ReadBytes(data, f); err != nil {
			failed++
			t.Errorf("exemple non lisible %s : %v", filepath.Base(f), err)
		}
	}
	t.Logf("exemples : %d lus, %d en échec", len(examples)-failed, failed)
}
