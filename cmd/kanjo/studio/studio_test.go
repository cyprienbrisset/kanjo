package studio

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/api"
	"github.com/cyprienbrisset/kanjo/pkg/generate"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

const tok = "test-token"

func TestVersionRequiresToken(t *testing.T) {
	h := NewHandler(tok)
	// Sans jeton → 403.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/version", nil))
	if rec.Code != http.StatusForbidden {
		t.Errorf("sans jeton : code %d, veut 403", rec.Code)
	}
	// Avec jeton → 200.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/version", nil)
	req.Header.Set("X-Kanjo-Token", tok)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "tool") {
		t.Errorf("avec jeton : code %d, corps %s", rec.Code, rec.Body.String())
	}
}

func TestValidateEndpoint(t *testing.T) {
	doc, _ := generate.Generate(0, generate.Options{Scenario: generate.ScenarioSimple, Seed: 1})
	body, _ := write.WriteBytes("cii", doc, write.DefaultOptions())

	h := NewHandler(tok)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/validate", bytes.NewReader(body))
	req.Header.Set("X-Kanjo-Token", tok)
	req.Header.Set("X-Kanjo-Filename", "F1.xml")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	var env api.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Summary.Total != 1 || env.Results[0].Status != api.StatusOK {
		t.Errorf("résultat inattendu : %+v", env)
	}
}

func TestInspectEndpoint(t *testing.T) {
	doc, _ := generate.Generate(0, generate.Options{Scenario: generate.ScenarioSimple, Seed: 2})
	body, _ := write.WriteBytes("cii", doc, write.DefaultOptions())
	h := NewHandler(tok)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/inspect", bytes.NewReader(body))
	req.Header.Set("X-Kanjo-Token", tok)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["format"] != "cii" || out["verdict"] != "ok" || out["document"] == nil {
		t.Errorf("inspect inattendu : %v", out)
	}
}

func TestConvertEndpoint(t *testing.T) {
	doc, _ := generate.Generate(0, generate.Options{Scenario: generate.ScenarioSimple, Seed: 2})
	body, _ := write.WriteBytes("cii", doc, write.DefaultOptions())
	h := NewHandler(tok)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/convert?to=ubl", bytes.NewReader(body))
	req.Header.Set("X-Kanjo-Token", tok)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["outputBase64"] == nil || out["outputBase64"] == "" {
		t.Errorf("convert n'a pas renvoyé de sortie : %v", out)
	}
}

func TestIndexInjectsToken(t *testing.T) {
	h := NewHandler(tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	body := rec.Body.String()
	if strings.Contains(body, "__TOKEN__") {
		t.Error("le jeton n'a pas été injecté dans la page")
	}
	if !strings.Contains(body, tok) {
		t.Error("la page servie ne contient pas le jeton de session")
	}
}

func TestLoopbackSecurity(t *testing.T) {
	if isLoopback("8.8.8.8") {
		t.Error("8.8.8.8 ne doit pas être considéré comme loopback")
	}
	if !isLoopback("127.0.0.1") || !isLoopback("localhost") {
		t.Error("127.0.0.1 et localhost doivent être loopback")
	}
}
