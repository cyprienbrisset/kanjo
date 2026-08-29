// Package preset gère les jeux de réglages de conversion nommés et réutilisables (§G6).
//
// MUST (§G6) : un preset ne contient JAMAIS de chemin absolu contenant un nom d'utilisateur,
// ni de secret. Il ne porte que des réglages de conversion ; la source et la destination sont
// fournies à l'exécution. Le stockage est local, en JSON lisible et versionnable.
package preset

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
)

const ext = ".kanjo-preset.json"

// Preset est un jeu de réglages de conversion réutilisable.
type Preset struct {
	Name     string   `json:"name"`
	To       string   `json:"to"`                 // format cible
	Profile  string   `json:"profile,omitempty"`  // profil
	Syntax   string   `json:"syntax,omitempty"`   // syntaxe (xrechnung)
	MaxLoss  string   `json:"maxLoss,omitempty"`  // politique de perte
	Naming   string   `json:"naming,omitempty"`   // gabarit de nommage
	Validate bool     `json:"validate,omitempty"` // valider la sortie
	RuleSets []string `json:"ruleSets,omitempty"` // jeux de règles de validation
}

// Erreurs.
var (
	ErrInvalidName = errors.New("nom de preset invalide")
	ErrNotFound    = errors.New("preset introuvable")
)

// Store est un dépôt local de presets.
type Store struct{ dir string }

// DefaultDir renvoie le dossier standard des presets (config utilisateur).
func DefaultDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "kanjo", "presets"), nil
}

// Open ouvre (ou prépare) un dépôt de presets dans le dossier donné.
func Open(dir string) *Store { return &Store{dir: dir} }

// validName vérifie qu'un nom est sûr comme nom de fichier (pas de séparateur, pas de secret
// évident, caractères restreints).
func validName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if !(r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func (s *Store) path(name string) string { return filepath.Join(s.dir, name+ext) }

// Save enregistre un preset (écriture atomique).
func (s *Store) Save(p Preset) error {
	if !validName(p.Name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, p.Name)
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(s.path(p.Name), append(data, '\n'), 0o644)
}

// Load charge un preset par nom.
func (s *Store) Load(name string) (Preset, error) {
	if !validName(name) {
		return Preset{}, fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	data, err := os.ReadFile(s.path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return Preset{}, fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return Preset{}, err
	}
	var p Preset
	if err := json.Unmarshal(data, &p); err != nil {
		return Preset{}, fmt.Errorf("preset %s illisible: %w", name, err)
	}
	return p, nil
}

// List renvoie tous les presets, triés par nom.
func (s *Store) List() ([]Preset, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Preset
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ext)
		if p, err := s.Load(name); err == nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Delete supprime un preset.
func (s *Store) Delete(name string) error {
	if !validName(name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	if err := os.Remove(s.path(name)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return err
	}
	return nil
}
