package pipeline

import (
	"os"
	"path/filepath"
)

// Watcher détecte les fichiers stables déposés dans un dossier, par scrutation (polling) —
// pur Go, sans dépendance fsnotify (§10.3). Un fichier est considéré « prêt » lorsqu'il a été
// vu à un scan précédent avec la même taille (écriture terminée : taille stable sur 2 vérifs).
type Watcher struct {
	dir       string
	recursive bool
	prevSize  map[string]int64
	done      map[string]bool
}

// NewWatcher crée un observateur sur un dossier.
func NewWatcher(dir string, recursive bool) *Watcher {
	return &Watcher{
		dir:       dir,
		recursive: recursive,
		prevSize:  map[string]int64{},
		done:      map[string]bool{},
	}
}

// Ready scanne le dossier et renvoie les fichiers devenus stables depuis le scan précédent
// (et non encore traités). Ces fichiers sont alors marqués comme traités et ne seront plus
// renvoyés. Un fichier vu pour la première fois n'est jamais renvoyé immédiatement : il doit
// survivre à un scan avec une taille inchangée (anti-écriture-en-cours).
func (w *Watcher) Ready() ([]string, error) {
	current := map[string]int64{}
	if err := w.scan(w.dir, current); err != nil {
		return nil, err
	}

	var ready []string
	for path, size := range current {
		if w.done[path] {
			continue
		}
		if prev, seen := w.prevSize[path]; seen && prev == size {
			ready = append(ready, path)
			w.done[path] = true
		}
	}
	w.prevSize = current
	return ready, nil
}

// Forget retire un fichier de l'état (ex. après déplacement vers done/failed), afin qu'un
// nouveau fichier de même nom soit à nouveau détecté ultérieurement.
func (w *Watcher) Forget(path string) {
	delete(w.done, path)
	delete(w.prevSize, path)
}

func (w *Watcher) scan(dir string, out map[string]int64) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if w.recursive && !isSubStateDir(e.Name()) {
				if err := w.scan(full, out); err != nil {
					return err
				}
			}
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out[full] = info.Size()
	}
	return nil
}

// isSubStateDir ignore les sous-dossiers d'état du watcher.
func isSubStateDir(name string) bool {
	switch name {
	case "processing", "done", "failed", "output":
		return true
	default:
		return false
	}
}
