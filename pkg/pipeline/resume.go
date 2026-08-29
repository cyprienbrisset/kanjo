package pipeline

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

// State persiste l'ensemble des fichiers déjà traités pour permettre `--resume` : un lot
// interrompu reprend sans retraiter les fichiers déjà écrits (§10.1 MUST). Le format est un
// journal append-only (un chemin par ligne), résistant à une interruption brutale.
type State struct {
	mu   sync.Mutex
	done map[string]bool
	f    *os.File
}

// LoadState ouvre (ou crée) le journal de reprise et charge l'ensemble déjà traité.
func LoadState(path string) (*State, error) {
	done := map[string]bool{}
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			if line := sc.Text(); line != "" {
				done[line] = true
			}
		}
		_ = f.Close()
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("ouverture du journal de reprise %q: %w", path, err)
	}
	return &State{done: done, f: f}, nil
}

// Done indique si un fichier a déjà été traité.
func (s *State) Done(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done[path]
}

// MarkDone enregistre un fichier comme traité (écriture immédiate pour survivre à un arrêt).
func (s *State) MarkDone(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done[path] {
		return nil
	}
	s.done[path] = true
	if _, err := fmt.Fprintln(s.f, path); err != nil {
		return err
	}
	return s.f.Sync()
}

// Filter retire les fichiers déjà traités d'une liste.
func (s *State) Filter(files []string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := files[:0:0]
	for _, f := range files {
		if !s.done[f] {
			out = append(out, f)
		}
	}
	return out
}

// Close ferme le journal de reprise.
func (s *State) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Close()
}
