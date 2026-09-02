package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// TestPendingFilesReadsAndClears vérifie le modèle « pull » des fichiers de lancement :
// PendingFiles lit tous les chemins en attente (quel qu'en soit le nombre) et vide la file,
// de sorte qu'un second appel ne renvoie rien. Non-régression de la course d'émission qui
// perdait silencieusement les lots volumineux au chargement (affichage vide).
func TestPendingFilesReadsAndClears(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for i := 0; i < 12; i++ {
		p := filepath.Join(dir, "f"+string(rune('a'+i))+".xml")
		if err := os.WriteFile(p, []byte("<Invoice/>"), 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}

	app := NewApp(paths)
	files, err := app.PendingFiles()
	if err != nil {
		t.Fatalf("PendingFiles: %v", err)
	}
	if len(files) != len(paths) {
		t.Fatalf("attendu %d fichiers, reçu %d", len(paths), len(files))
	}
	if want := base64.StdEncoding.EncodeToString([]byte("<Invoice/>")); files[0].Data != want {
		t.Errorf("contenu base64 inattendu: %q", files[0].Data)
	}

	// Deuxième appel : la file est vidée, plus aucun fichier.
	again, err := app.PendingFiles()
	if err != nil {
		t.Fatalf("PendingFiles (2e appel): %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("la file de lancement doit être vidée, reçu %d", len(again))
	}
}

// TestPendingFilesEmpty : sans argument de lancement, PendingFiles ne renvoie rien ni erreur.
func TestPendingFilesEmpty(t *testing.T) {
	app := NewApp(nil)
	files, err := app.PendingFiles()
	if err != nil {
		t.Fatalf("PendingFiles: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("attendu 0 fichier, reçu %d", len(files))
	}
}
