// Package write définit l'interface commune des écrivains de formats et un registre.
// Comme pour read, chaque format concret vit dans un sous-paquet qui s'enregistre via
// Register() ; ce paquet n'importe aucun sous-paquet (pas de cycle).
package write

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// Profile est un profil Factur-X / niveau de richesse.
type Profile string

const (
	ProfileMinimum  Profile = "minimum"
	ProfileBasicWL  Profile = "basicwl"
	ProfileBasic    Profile = "basic"
	ProfileEN16931  Profile = "en16931"
	ProfileExtended Profile = "extended"
)

// Options paramètre l'écriture.
type Options struct {
	Profile Profile // profil cible (défaut en16931)
	Syntax  string  // "ubl" | "cii" pour les cibles à double syntaxe (xrechnung)
	Indent  bool    // indentation lisible du XML/JSON

	// CustomizationID, s'il est non vide, remplace l'identifiant de spécification dérivé
	// du profil (CII : ram:ID de GuidelineSpecifiedDocumentContextParameter ; UBL :
	// cbc:CustomizationID). Utilisé par les CIUS EN 16931 (XRechnung, Peppol BIS Billing).
	CustomizationID string
	// ProfileID, s'il est non vide, est émis en UBL comme cbc:ProfileID. Utilisé notamment
	// par Peppol BIS Billing 3.0.
	ProfileID string
}

// DefaultOptions renvoie des options raisonnables.
func DefaultOptions() Options {
	return Options{Profile: ProfileEN16931, Indent: true}
}

// Writer sérialise un document pivot vers un format cible.
type Writer func(doc *model.Document, opts Options) ([]byte, error)

var (
	mu       sync.RWMutex
	registry = map[string]Writer{}
)

// ErrUnsupportedTarget est renvoyée quand aucun écrivain n'existe pour une cible.
var ErrUnsupportedTarget = errors.New("cible d'écriture non prise en charge")

// Register associe un écrivain à un nom de cible ("cii", "ubl", "json", …).
func Register(target string, w Writer) {
	mu.Lock()
	defer mu.Unlock()
	registry[target] = w
}

// Get renvoie l'écrivain enregistré pour une cible, ou nil.
func Get(target string) Writer {
	mu.RLock()
	defer mu.RUnlock()
	return registry[target]
}

// Targets liste les cibles enregistrées, triées.
func Targets() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// WriteBytes sérialise doc vers la cible demandée.
func WriteBytes(target string, doc *model.Document, opts Options) ([]byte, error) {
	w := Get(target)
	if w == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedTarget, target)
	}
	return w(doc, opts)
}
