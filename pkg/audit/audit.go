// Package audit tient le journal d'audit horodaté (§12, §17.5). Il permet de rejouer et de
// justifier un traitement des années plus tard.
//
// MUST (§17.4/§17.5/§G12) : AUCUN contenu de facture n'apparaît dans le journal — ni raison
// sociale, ni SIREN, ni montant, ni IBAN. Uniquement des chemins, des empreintes (SHA-256) et
// des identifiants techniques. Le type Entry n'a délibérément aucun champ de donnée métier.
package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/internal/version"
)

// Entry est une entrée d'audit (§17.5). Champs strictement techniques.
type Entry struct {
	Ts            string   `json:"ts"`
	Action        string   `json:"action"`
	Actor         string   `json:"actor"`
	InputSha256   string   `json:"inputSha256,omitempty"`
	OutputSha256  string   `json:"outputSha256,omitempty"`
	InputFormat   string   `json:"inputFormat,omitempty"`
	OutputFormat  string   `json:"outputFormat,omitempty"`
	Profile       string   `json:"profile,omitempty"`
	ToolVersion   string   `json:"toolVersion"`
	RulesVersion  string   `json:"rulesVersion"`
	Verdict       string   `json:"verdict"`
	LossCount     int      `json:"lossCount"`
	DisabledRules []string `json:"disabledRules"`
}

// Journal est un journal d'audit append-only (JSONL).
type Journal struct {
	mu sync.Mutex
	f  *os.File
}

// DefaultPath renvoie le chemin standard du journal (répertoire d'état, §15.1).
func DefaultPath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "audit.jsonl"), nil
}

// Open ouvre (ou crée) un journal en append.
func Open(path string) (*Journal, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("ouverture du journal d'audit %q: %w", path, err)
	}
	return &Journal{f: f}, nil
}

// Log complète l'entrée (horodatage, acteur, versions) et l'écrit.
func (j *Journal) Log(e Entry) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if e.Ts == "" {
		e.Ts = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if e.Actor == "" {
		e.Actor = currentActor()
	}
	e.ToolVersion = version.Tool
	e.RulesVersion = version.Rules
	if e.DisabledRules == nil {
		e.DisabledRules = []string{}
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := j.f.Write(append(data, '\n')); err != nil {
		return err
	}
	return j.f.Sync()
}

// Close ferme le journal.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.f.Close()
}

// Record est un raccourci : ouvre le journal par défaut, écrit une entrée, referme.
// Les erreurs d'audit ne doivent jamais interrompre un traitement ; l'appelant peut les ignorer.
func Record(e Entry) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	j, err := Open(path)
	if err != nil {
		return err
	}
	defer j.Close()
	return j.Log(e)
}

// Read charge toutes les entrées d'un journal (pour list/export).
func Read(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // ligne corrompue ignorée, le journal reste exploitable
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

// ExportCSV rend les entrées en CSV (sans donnée personnelle).
func ExportCSV(entries []Entry) []byte {
	var b strings.Builder
	b.WriteString("ts,action,actor,inputFormat,outputFormat,profile,verdict,lossCount,inputSha256,outputSha256\n")
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%d,%s,%s\n",
			e.Ts, e.Action, e.Actor, e.InputFormat, e.OutputFormat, e.Profile,
			e.Verdict, e.LossCount, e.InputSha256, e.OutputSha256))
	}
	return []byte(b.String())
}

// WriteJSONL réécrit un ensemble d'entrées en JSONL (pour export filtré, écriture atomique).
func WriteJSONL(path string, entries []Entry) error {
	var b strings.Builder
	for _, e := range entries {
		data, _ := json.Marshal(e)
		b.Write(data)
		b.WriteByte('\n')
	}
	return fsatomic.WriteFile(path, []byte(b.String()), 0o600)
}

func currentActor() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return "system:" + u.Username
	}
	if v := os.Getenv("USER"); v != "" {
		return "system:" + v
	}
	return "system:unknown"
}

func stateDir() (string, error) {
	// XDG_STATE_HOME sur Linux, sinon repli sur le cache utilisateur.
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "kanjo"), nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kanjo", "state"), nil
}
