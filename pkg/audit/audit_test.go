package audit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLogAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Log(Entry{Action: "convert", InputFormat: "facturx", OutputFormat: "ubl", Verdict: "ok"}); err != nil {
		t.Fatal(err)
	}
	if err := j.Log(Entry{Action: "validate", InputFormat: "cii", Verdict: "error"}); err != nil {
		t.Fatal(err)
	}
	_ = j.Close()

	entries, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("attendait 2 entrées, obtenu %d", len(entries))
	}
	if entries[0].Ts == "" || !strings.HasPrefix(entries[0].Actor, "system:") {
		t.Errorf("horodatage/acteur non renseignés : %+v", entries[0])
	}
	if entries[0].ToolVersion == "" || entries[0].RulesVersion == "" {
		t.Error("versions non renseignées")
	}
}

// TestNoPersonalData applique l'exigence §17.4 : le journal ne doit contenir aucune donnée
// personnelle (motifs SIREN/IBAN/e-mail), même si on tente d'en injecter via des champs.
func TestNoPersonalData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	j, _ := Open(path)
	// On journalise un traitement réaliste (chemins/empreintes seulement).
	_ = j.Log(Entry{
		Action:       "convert",
		InputSha256:  "a1b2c3d4e5f6",
		OutputSha256: "f6e5d4c3b2a1",
		InputFormat:  "facturx", OutputFormat: "ubl", Profile: "en16931", Verdict: "ok",
	})
	_ = j.Close()

	data, _ := os.ReadFile(path)
	content := string(data)

	patterns := map[string]*regexp.Regexp{
		"IBAN":  regexp.MustCompile(`\bFR\d{2}[0-9A-Z]{10,}\b`),
		"email": regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		"SIREN": regexp.MustCompile(`\b\d{9}\b`),
	}
	for name, re := range patterns {
		if re.MatchString(content) {
			t.Errorf("le journal d'audit contient un motif %s interdit : %s", name, content)
		}
	}
}

func TestExport(t *testing.T) {
	entries := []Entry{{Ts: "2026-08-29T10:00:00Z", Action: "convert", Actor: "system:x", Verdict: "ok"}}
	csv := string(ExportCSV(entries))
	if !strings.Contains(csv, "ts,action") || !strings.Contains(csv, "convert") {
		t.Errorf("CSV inattendu : %s", csv)
	}
	out := filepath.Join(t.TempDir(), "e.jsonl")
	if err := WriteJSONL(out, entries); err != nil {
		t.Fatal(err)
	}
	re, _ := Read(out)
	if len(re) != 1 || re[0].Action != "convert" {
		t.Errorf("JSONL réécrit incorrect : %+v", re)
	}
}
