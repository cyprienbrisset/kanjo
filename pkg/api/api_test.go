package api

import (
	"encoding/json"
	"testing"
)

func TestNewEnvelope(t *testing.T) {
	e := NewEnvelope("validate", "2026-08-12T00:00:00Z")
	if e.Command != "validate" {
		t.Errorf("command = %q", e.Command)
	}
	if e.SchemaVersion == "" {
		t.Error("schemaVersion doit être renseigné depuis version.Schema")
	}
	// L'enveloppe doit se sérialiser avec les clés stables du contrat JSON.
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"schemaVersion"`, `"command"`, `"summary"`} {
		if !containsBytes(b, key) {
			t.Errorf("JSON ne contient pas la clé %s", key)
		}
	}
}

func TestSummaryAdd(t *testing.T) {
	var s Summary
	s.Add(StatusOK)
	s.Add(StatusOK)
	s.Add(StatusWarning)
	s.Add(StatusError)
	s.Add(Status("inconnu")) // ne compte que dans Total
	if s.Total != 5 || s.OK != 2 || s.Warning != 1 || s.Error != 1 {
		t.Errorf("résumé = %+v", s)
	}
}

func containsBytes(b []byte, sub string) bool {
	return len(b) > 0 && json.Valid(b) && indexOf(string(b), sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
