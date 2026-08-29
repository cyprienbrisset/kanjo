// Package test regroupe les tests d'intégration de bout en bout de Kanjō : ils s'appuient sur le
// corpus publiable (100 % synthétique, versionné dans testdata/corpus/published) et vérifient que
// les factures valides sont déclarées conformes et les factures volontairement erronées rejetées.
package test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/rules"

	_ "github.com/cyprienbrisset/kanjo/pkg/formats"   // lecteurs/écrivains
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all" // jeux de règles
)

func walkXML(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".xml") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s: %v", dir, err)
	}
	return files
}

func validate(t *testing.T, path string) rules.Report {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture %s: %v", path, err)
	}
	rd, err := read.ReadBytes(data, path)
	if err != nil {
		t.Fatalf("analyse %s: %v", path, err)
	}
	return rules.Validate(rd.Doc)
}

func TestPublishedCorpusValidesSontConformes(t *testing.T) {
	files := walkXML(t, filepath.Join("..", "testdata", "corpus", "published", "valides"))
	if len(files) == 0 {
		t.Fatal("corpus publiable « valides » introuvable — lancez testdata/corpus/generer-corpus.sh")
	}
	for _, f := range files {
		t.Run(filepath.Base(filepath.Dir(f))+"/"+filepath.Base(f), func(t *testing.T) {
			if rep := validate(t, f); rep.HasErrors() {
				t.Errorf("attendu conforme, anomalies : %+v", rep.Findings)
			}
		})
	}
}

func TestPublishedCorpusInvalidesSontRejetees(t *testing.T) {
	files := walkXML(t, filepath.Join("..", "testdata", "corpus", "published", "invalides"))
	if len(files) == 0 {
		t.Fatal("corpus publiable « invalides » introuvable — lancez testdata/corpus/generer-corpus.sh")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			if rep := validate(t, f); !rep.HasErrors() {
				t.Errorf("attendu NON conforme, mais aucune anomalie détectée")
			}
		})
	}
}
