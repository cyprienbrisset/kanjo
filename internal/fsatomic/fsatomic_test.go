package fsatomic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.xml")
	if err := WriteFile(path, []byte("<x/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "<x/>" {
		t.Errorf("contenu = %q", got)
	}
	// Aucun fichier temporaire résiduel.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("attendait 1 fichier, obtenu %d", len(entries))
	}
}

func TestWriteFileOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.xml")
	if err := WriteFile(path, []byte("ancien"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("nouveau"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "nouveau" {
		t.Errorf("contenu = %q, veut nouveau", got)
	}
}
