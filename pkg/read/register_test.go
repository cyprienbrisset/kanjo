package read

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// TestRegisterDuplicatePanics : enregistrer deux lecteurs pour le même format est un bogue de
// programmation et doit paniquer (au lieu d'écraser silencieusement le premier).
func TestRegisterDuplicatePanics(t *testing.T) {
	const f Format = "test-dup-format"
	reader := func(_ []byte, _ string) (*model.Document, error) { return nil, nil }

	Register(f, reader)
	defer func() {
		mu.Lock()
		delete(registry, f) // nettoyage du registre global
		mu.Unlock()
		if r := recover(); r == nil {
			t.Fatal("un second enregistrement du même format aurait dû paniquer")
		}
	}()
	Register(f, reader) // second enregistrement → panique attendue
}
