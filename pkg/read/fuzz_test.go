package read_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/read"

	_ "github.com/cyprienbrisset/kanjo/pkg/formats" // enregistre lecteurs/écrivains
)

// FuzzReadBytes garantit qu'aucune entrée — aléatoire, tronquée ou malveillante — ne fait paniquer
// la détection de format ni les lecteurs. Toute entrée invalide doit produire une erreur.
func FuzzReadBytes(f *testing.F) {
	f.Add([]byte(`<?xml version="1.0"?><rsm:CrossIndustryInvoice/>`))
	f.Add([]byte(`<?xml version="1.0"?><Invoice xmlns="urn:oasis:...:Invoice-2"/>`))
	f.Add([]byte(`<p:FatturaElettronica/>`))
	f.Add([]byte(`{"schemaVersion":"github.com/cyprienbrisset/kanjo/1"}`))
	f.Add([]byte("%PDF-1.7\n%\xE2\xE3\xCF\xD3"))
	f.Add([]byte(``))
	f.Add([]byte(`<Invoice>`))
	// Graines issues du corpus de sécurité et de quelques factures publiées.
	seedFromDir(f, filepath.Join("..", "..", "testdata", "fuzz", "xxe"))
	seedFromDir(f, filepath.Join("..", "..", "testdata", "corpus", "published", "valides", "cii", "simple"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_ = read.Detect(data)                   // ne doit pas paniquer
		_, _ = read.ReadBytes(data, "fuzz.dat") // ne doit pas paniquer
	})
}

func seedFromDir(f *testing.F, dir string) {
	f.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if b, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
			f.Add(b)
		}
	}
}
