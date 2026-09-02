package write

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// TestRegisterDuplicatePanics : enregistrer deux écrivains pour la même cible est un bogue de
// programmation et doit paniquer (au lieu d'écraser silencieusement le premier).
func TestRegisterDuplicatePanics(t *testing.T) {
	const target = "test-dup-target"
	writer := func(_ *model.Document, _ Options) ([]byte, error) { return nil, nil }

	Register(target, writer)
	defer func() {
		mu.Lock()
		delete(registry, target) // nettoyage du registre global
		mu.Unlock()
		if r := recover(); r == nil {
			t.Fatal("un second enregistrement de la même cible aurait dû paniquer")
		}
	}()
	Register(target, writer) // second enregistrement → panique attendue
}
