package pipeline

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Discover développe une liste d'entrées (fichiers, dossiers, globs) en une liste triée et
// dédoublonnée de fichiers. Les dossiers sont parcourus récursivement si recursive est vrai.
// Les filtres include/exclude sont des motifs glob appliqués au nom de base.
func Discover(inputs []string, recursive bool, include, exclude []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if seen[p] || !matches(filepath.Base(p), include, exclude) {
			return
		}
		seen[p] = true
		out = append(out, p)
	}

	for _, in := range inputs {
		if in == "-" { // stdin : conservé tel quel
			add("-")
			continue
		}
		if hasGlobMeta(in) {
			matchesGlob, err := filepath.Glob(in)
			if err != nil {
				return nil, fmt.Errorf("motif invalide %q: %w", in, err)
			}
			for _, m := range matchesGlob {
				if err := addPath(m, recursive, add); err != nil {
					return nil, err
				}
			}
			continue
		}
		if err := addPath(in, recursive, add); err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

// addPath ajoute un fichier, ou parcourt un dossier.
func addPath(p string, recursive bool, add func(string)) error {
	info, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("accès à %q: %w", p, err)
	}
	if !info.IsDir() {
		add(p)
		return nil
	}
	if !recursive {
		entries, err := os.ReadDir(p)
		if err != nil {
			return fmt.Errorf("lecture du dossier %q: %w", p, err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				add(filepath.Join(p, e.Name()))
			}
		}
		return nil
	}
	return filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			add(path)
		}
		return nil
	})
}

func hasGlobMeta(s string) bool { return strings.ContainsAny(s, "*?[") }

// matches applique les filtres include (au moins un doit correspondre si non vide) et
// exclude (aucun ne doit correspondre).
func matches(name string, include, exclude []string) bool {
	for _, pat := range exclude {
		if ok, _ := filepath.Match(pat, name); ok {
			return false
		}
	}
	if len(include) == 0 {
		return true
	}
	for _, pat := range include {
		if ok, _ := filepath.Match(pat, name); ok {
			return true
		}
	}
	return false
}
