package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/api"
)

func writeFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunAggregatesAndSurvivesPanic(t *testing.T) {
	files := []string{"a", "b", "boom", "c"}
	proc := func(path string) api.Result {
		if path == "boom" {
			panic("kaboom")
		}
		return api.Result{Input: path, Status: api.StatusOK}
	}
	rep := Run(files, proc, Options{Workers: 2})
	if rep.Summary.Total != 4 {
		t.Errorf("total = %d, veut 4", rep.Summary.Total)
	}
	if rep.Summary.Error != 1 || rep.Summary.OK != 3 {
		t.Errorf("résumé inattendu : %+v", rep.Summary)
	}
	// La panique est convertie en erreur, le lot n'est pas interrompu.
	var boom *api.Result
	for i := range rep.Results {
		if rep.Results[i].Input == "boom" {
			boom = &rep.Results[i]
		}
	}
	if boom == nil || !strings.Contains(boom.Error, "panique") {
		t.Errorf("le fichier en panique n'a pas produit d'erreur exploitable : %+v", boom)
	}
}

func TestDiscoverRecursiveAndFilters(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "a.xml", "b.pdf", "sub/c.xml", "sub/d.txt")

	all, err := Discover([]string{dir}, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Errorf("récursif = %d fichiers, veut 4 : %v", len(all), all)
	}

	onlyXML, err := Discover([]string{dir}, true, []string{"*.xml"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(onlyXML) != 2 {
		t.Errorf("filtre *.xml = %d, veut 2 : %v", len(onlyXML), onlyXML)
	}

	noTxt, err := Discover([]string{dir}, true, nil, []string{"*.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(noTxt) != 3 {
		t.Errorf("exclude *.txt = %d, veut 3 : %v", len(noTxt), noTxt)
	}

	nonRec, err := Discover([]string{dir}, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(nonRec) != 2 {
		t.Errorf("non récursif = %d, veut 2 : %v", len(nonRec), nonRec)
	}
}

func TestResumeSkipsDone(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.log")
	st, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.MarkDone("a"); err != nil {
		t.Fatal(err)
	}
	if err := st.MarkDone("b"); err != nil {
		t.Fatal(err)
	}
	remaining := st.Filter([]string{"a", "b", "c", "d"})
	if len(remaining) != 2 || remaining[0] != "c" || remaining[1] != "d" {
		t.Errorf("Filter = %v, veut [c d]", remaining)
	}
	_ = st.Close()

	// Recharger : l'état persiste.
	st2, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if !st2.Done("a") || st2.Done("c") {
		t.Error("l'état de reprise n'a pas persisté correctement")
	}
	_ = st2.Close()
}
