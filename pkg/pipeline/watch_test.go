package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWatcherStabilityDetection(t *testing.T) {
	dir := t.TempDir()
	w := NewWatcher(dir, false)

	path := filepath.Join(dir, "a.xml")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Premier scan : le fichier vient d'apparaître, pas encore stable.
	r1, _ := w.Ready()
	if len(r1) != 0 {
		t.Errorf("premier scan devrait être vide, obtenu %v", r1)
	}
	// Deuxième scan : taille inchangée → prêt.
	r2, _ := w.Ready()
	if len(r2) != 1 || r2[0] != path {
		t.Errorf("deuxième scan devrait renvoyer [a], obtenu %v", r2)
	}
	// Troisième scan : déjà traité → plus renvoyé.
	r3, _ := w.Ready()
	if len(r3) != 0 {
		t.Errorf("troisième scan devrait être vide, obtenu %v", r3)
	}
}

func TestWatcherGrowingFileNotReady(t *testing.T) {
	dir := t.TempDir()
	w := NewWatcher(dir, false)
	path := filepath.Join(dir, "b.xml")

	os.WriteFile(path, []byte("a"), 0o644)
	w.Ready() // 1er passage : première vue

	os.WriteFile(path, []byte("aaaa"), 0o644) // le fichier grossit
	r := mustReady(t, w)
	if len(r) != 0 {
		t.Errorf("un fichier qui grossit ne doit pas être prêt, obtenu %v", r)
	}
	// Maintenant stable.
	r = mustReady(t, w)
	if len(r) != 1 {
		t.Errorf("après stabilisation, le fichier doit être prêt, obtenu %v", r)
	}
}

func TestWatcherIgnoresStateDirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "done"), 0o755)
	os.WriteFile(filepath.Join(dir, "done", "old.xml"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "new.xml"), []byte("x"), 0o644)

	w := NewWatcher(dir, true)
	w.Ready()
	r := mustReady(t, w)
	for _, p := range r {
		if filepath.Base(filepath.Dir(p)) == "done" {
			t.Errorf("le watcher ne doit pas surveiller le dossier done/ : %s", p)
		}
	}
}

func mustReady(t *testing.T, w *Watcher) []string {
	t.Helper()
	r, err := w.Ready()
	if err != nil {
		t.Fatal(err)
	}
	return r
}
