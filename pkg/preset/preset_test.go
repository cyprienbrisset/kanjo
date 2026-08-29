package preset

import (
	"os"
	"strings"
	"testing"
)

func TestSaveLoadListDelete(t *testing.T) {
	s := Open(t.TempDir())
	p := Preset{Name: "fournisseurs", To: "ubl", Profile: "en16931", MaxLoss: "minor"}
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("fournisseurs")
	if err != nil {
		t.Fatal(err)
	}
	if got.To != "ubl" || got.Profile != "en16931" {
		t.Errorf("preset chargé incorrect : %+v", got)
	}

	if err := s.Save(Preset{Name: "archivage-cii", To: "cii"}); err != nil {
		t.Fatal(err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "archivage-cii" || list[1].Name != "fournisseurs" {
		t.Errorf("liste incorrecte : %+v", list)
	}

	if err := s.Delete("fournisseurs"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("fournisseurs"); err == nil {
		t.Error("preset supprimé encore chargeable")
	}
}

func TestInvalidName(t *testing.T) {
	s := Open(t.TempDir())
	for _, bad := range []string{"", "../evil", "a/b", "nom avec espace", strings.Repeat("x", 65)} {
		if err := s.Save(Preset{Name: bad, To: "ubl"}); err == nil {
			t.Errorf("nom %q aurait dû être rejeté", bad)
		}
	}
}

func TestExportedPresetHasNoAbsolutePath(t *testing.T) {
	// §G6 MUST : un preset ne contient jamais de chemin absolu ni de nom d'utilisateur.
	dir := t.TempDir()
	s := Open(dir)
	if err := s.Save(Preset{Name: "p", To: "ubl", Naming: "{issueDate:2006-01}/{id}"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(s.path("p"))
	if strings.Contains(string(data), "/Users/") || strings.Contains(string(data), "/home/") {
		t.Errorf("le preset exporté contient un chemin absolu : %s", data)
	}
}
