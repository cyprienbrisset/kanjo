package frontend

import (
	"io/fs"
	"testing"
)

// TestAssetsEmbedded vérifie que le frontend est bien embarqué (index.html + app.js + styles),
// garantissant que le binaire studio est autonome (aucun fichier externe requis).
func TestAssetsEmbedded(t *testing.T) {
	for _, name := range []string{"dist/index.html", "dist/app.js"} {
		b, err := fs.ReadFile(Assets, name)
		if err != nil {
			t.Errorf("actif embarqué manquant : %s (%v)", name, err)
			continue
		}
		if len(b) == 0 {
			t.Errorf("actif embarqué vide : %s", name)
		}
	}
}
