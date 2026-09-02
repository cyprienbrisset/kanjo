// Package audit tient le journal d'audit horodaté (§12, §17.5). Il permet de rejouer et de
// justifier un traitement des années plus tard.
//
// MUST (§17.4/§17.5/§G12) : AUCUN contenu de facture n'apparaît dans le journal — ni raison
// sociale, ni SIREN, ni montant, ni IBAN. Uniquement des chemins, des empreintes (SHA-256) et
// des identifiants techniques. Le type Entry n'a délibérément aucun champ de donnée métier.
package audit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/internal/fslock"
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

	// Chaînage d'intégrité (tamper-evident, §17.5). Seq est un numéro de séquence croissant ;
	// PrevHash référence l'empreinte de l'entrée précédente ; Hash est l'empreinte SHA-256 de la
	// présente entrée (calculée hors champ Hash). Une rupture de chaîne révèle une modification,
	// une suppression ou une réinsertion. Ces champs ne contiennent aucune donnée métier.
	Seq      int64  `json:"seq,omitempty"`
	PrevHash string `json:"prevHash,omitempty"`
	Hash     string `json:"hash,omitempty"`
}

// Journal est un journal d'audit append-only (JSONL), avec chaînage d'intégrité.
type Journal struct {
	mu       sync.Mutex
	f        *os.File
	path     string // chemin du fichier (relecture de la dernière entrée sous verrou inter-processus)
	lastHash string // empreinte de la dernière entrée (pour le chaînage)
	lastSeq  int64  // numéro de séquence de la dernière entrée
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
	j := &Journal{f: f, path: path}
	// Reprendre le chaînage à partir de la dernière entrée existante (relu ensuite sous verrou
	// à chaque écriture, ce qui couvre les écritures concurrentes d'autres processus).
	if last, ok := lastEntry(path); ok {
		j.lastSeq = last.Seq
		j.lastHash = last.Hash
	}
	return j, nil
}

// lastEntry lit la dernière entrée d'un journal sans charger tout le fichier (fenêtre de fin), pour
// reprendre le chaînage. Renvoie (Entry, false) si le journal est vide, absent ou illisible.
func lastEntry(path string) (Entry, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Entry{}, false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.Size() == 0 {
		return Entry{}, false
	}
	const window = 64 * 1024 // une entrée d'audit fait < 1 Ko ; la dernière tient dans cette fenêtre
	start := int64(0)
	if fi.Size() > window {
		start = fi.Size() - window
	}
	buf := make([]byte, fi.Size()-start)
	if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
		return Entry{}, false
	}
	lines := bytes.Split(buf, []byte{'\n'})
	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		var e Entry
		if json.Unmarshal(line, &e) != nil {
			return Entry{}, false
		}
		return e, true
	}
	return Entry{}, false
}

// Log complète l'entrée (horodatage, acteur, versions) et l'écrit.
//
// La section critique « lire la dernière entrée + calculer le chaînage + ajouter » est protégée par
// un mutex intra-processus ET un verrou de fichier inter-processus (§17.5) : deux invocations
// concurrentes de Kanjō ne peuvent plus attribuer le même numéro de séquence ni le même prevHash.
// Sous verrou, la vraie dernière entrée est relue depuis le fichier (un autre processus a pu écrire
// depuis l'ouverture), garantissant un chaînage cohérent et sans fourche.
func (j *Journal) Log(e Entry) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	unlock, err := fslock.Lock(j.f)
	if err != nil {
		return fmt.Errorf("verrouillage du journal d'audit: %w", err)
	}
	defer func() { _ = unlock() }()

	// Relire la dernière entrée réellement présente (écritures concurrentes d'autres processus).
	if last, ok := lastEntry(j.path); ok {
		j.lastSeq = last.Seq
		j.lastHash = last.Hash
	}

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
	// Chaînage d'intégrité : numéro de séquence + empreinte de l'entrée précédente + empreinte propre.
	e.Seq = j.lastSeq + 1
	e.PrevHash = j.lastHash
	e.Hash = entryHash(e)
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := j.f.Write(append(data, '\n')); err != nil {
		return err
	}
	j.lastSeq = e.Seq
	j.lastHash = e.Hash
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
