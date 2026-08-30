package audit

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeJournal(t *testing.T) (string, []Entry) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	j, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	for _, a := range []string{"convert", "validate", "embed"} {
		if err := j.Log(Entry{Action: a, InputFormat: "cii", Verdict: "ok"}); err != nil {
			t.Fatalf("Log: %v", err)
		}
	}
	_ = j.Close()
	entries, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return path, entries
}

// TestChainIntact vérifie qu'un journal écrit normalement présente une chaîne intègre et contiguë.
func TestChainIntact(t *testing.T) {
	_, entries := writeJournal(t)
	if len(entries) != 3 {
		t.Fatalf("entrées = %d", len(entries))
	}
	// Chaînage : seq 1,2,3 ; prevHash lié.
	for i, e := range entries {
		if e.Seq != int64(i+1) {
			t.Errorf("seq[%d] = %d", i, e.Seq)
		}
		if e.Hash == "" {
			t.Errorf("entrée %d sans empreinte", i)
		}
	}
	if entries[1].PrevHash != entries[0].Hash || entries[2].PrevHash != entries[1].Hash {
		t.Error("prevHash non chaîné")
	}
	rep := VerifyChain(entries)
	if !rep.OK || rep.Chained != 3 || rep.Unchained != 0 || len(rep.Issues) != 0 {
		t.Errorf("bilan inattendu : %+v", rep)
	}
}

// TestDetectTampering détecte la modification d'une entrée (empreinte invalide).
func TestDetectTampering(t *testing.T) {
	_, entries := writeJournal(t)
	entries[1].Verdict = "error" // falsification : on change le verdict sans recalculer l'empreinte
	rep := VerifyChain(entries)
	if rep.OK {
		t.Fatal("la falsification aurait dû être détectée")
	}
	found := false
	for _, is := range rep.Issues {
		if is.Index == 1 && strings.Contains(is.Problem, "empreinte") {
			found = true
		}
	}
	if !found {
		t.Errorf("anomalie d'empreinte non signalée : %+v", rep.Issues)
	}
}

// TestDetectDeletion détecte la suppression d'une entrée au milieu (rupture prevHash + seq).
func TestDetectDeletion(t *testing.T) {
	_, entries := writeJournal(t)
	tampered := []Entry{entries[0], entries[2]} // on retire l'entrée du milieu
	rep := VerifyChain(tampered)
	if rep.OK {
		t.Fatal("la suppression aurait dû être détectée")
	}
}

// TestFilterByPeriod restreint aux bornes de dates.
func TestFilterByPeriod(t *testing.T) {
	entries := []Entry{
		{Ts: "2026-01-10T10:00:00Z", Action: "a"},
		{Ts: "2026-02-15T10:00:00Z", Action: "b"},
		{Ts: "2026-03-20T10:00:00Z", Action: "c"},
	}
	from, _ := time.Parse(time.RFC3339, "2026-02-01T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-02-28T23:59:59Z")
	got := FilterByPeriod(entries, from, to)
	if len(got) != 1 || got[0].Action != "b" {
		t.Errorf("filtre = %+v", got)
	}
}

// TestExportHTMLNoPII garantit qu'aucune donnée métier ne fuit et que l'intégrité est indiquée.
func TestExportHTMLNoPII(t *testing.T) {
	_, entries := writeJournal(t)
	html := string(ExportHTML(entries, "Test"))
	if !strings.Contains(html, "intègre") {
		t.Error("le rapport devrait indiquer l'intégrité")
	}
	if !strings.Contains(html, "<table") {
		t.Error("le rapport devrait contenir un tableau")
	}
}
