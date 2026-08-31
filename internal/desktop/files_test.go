package desktop

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFilesEncodesBase64(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "facture.xml")
	content := []byte(`<?xml version="1.0"?><rsm:CrossIndustryInvoice/>`)
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := ReadFiles([]string{p})
	if err != nil {
		t.Fatalf("ReadFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("attendu 1 fichier, obtenu %d", len(files))
	}
	if files[0].Name != "facture.xml" {
		t.Errorf("Name = %q, veut facture.xml", files[0].Name)
	}
	got, err := base64.StdEncoding.DecodeString(files[0].Data)
	if err != nil {
		t.Fatalf("Data n'est pas du base64 valide: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("contenu décodé = %q, veut %q", got, content)
	}
}

func TestReadFilesMissingPathErrors(t *testing.T) {
	if _, err := ReadFiles([]string{"/introuvable/xyz.xml"}); err == nil {
		t.Fatal("un chemin introuvable doit produire une erreur")
	}
}

func TestReadFilesSizeExceededErrors(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "gros.xml")
	if err := os.WriteFile(p, []byte("plus de deux octets"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := readFiles([]string{p}, 2)
	if err == nil {
		t.Fatal("un fichier dépassant la limite doit produire une erreur")
	}
	if !strings.Contains(err.Error(), "taille maximale") {
		t.Errorf("l'erreur doit mentionner la taille, obtenu: %v", err)
	}
}
