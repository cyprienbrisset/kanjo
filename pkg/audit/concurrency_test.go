package audit

import (
	"path/filepath"
	"sync"
	"testing"
)

// TestConcurrentJournalsKeepChainCoherent simule plusieurs processus écrivant dans le même journal :
// chaque écrivain ouvre son propre Journal (descripteur distinct, comme un autre processus) puis
// journalise. Grâce au verrou de fichier et à la relecture de la dernière entrée sous verrou, la
// chaîne reste cohérente — numéros de séquence uniques et contigus, aucune fourche de prevHash.
//
// Deux descripteurs distincts sur le même fichier dans un même processus sont mutuellement exclus
// par flock (verrou associé à la description ouverte), ce qui reproduit la concurrence inter-processus.
func TestConcurrentJournalsKeepChainCoherent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	const n = 25

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j, err := Open(path)
			if err != nil {
				t.Errorf("Open: %v", err)
				return
			}
			defer j.Close()
			if err := j.Log(Entry{Action: "concurrent", Verdict: "ok"}); err != nil {
				t.Errorf("Log: %v", err)
			}
		}()
	}
	wg.Wait()

	entries, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != n {
		t.Fatalf("attendu %d entrées, obtenu %d", n, len(entries))
	}

	rep := VerifyChain(entries)
	if !rep.OK {
		t.Fatalf("chaîne rompue après écritures concurrentes: %+v", rep.Issues)
	}

	seen := map[int64]bool{}
	for _, e := range entries {
		if seen[e.Seq] {
			t.Fatalf("numéro de séquence dupliqué: %d", e.Seq)
		}
		seen[e.Seq] = true
	}
	for i := int64(1); i <= n; i++ {
		if !seen[i] {
			t.Fatalf("numéro de séquence manquant: %d", i)
		}
	}
}
