// Package fsatomic fournit l'écriture atomique de fichiers (ADR-010) : écriture dans un
// fichier temporaire du même répertoire puis renommage. Aucune sortie partielle ne peut
// subsister en cas d'interruption. Pur Go, bibliothèque standard uniquement.
package fsatomic

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile écrit data dans path de façon atomique. Le fichier temporaire est créé dans le
// même répertoire (pour garantir que rename() reste atomique, même montage) puis renommé.
func WriteFile(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".kanjo-*.tmp")
	if err != nil {
		return fmt.Errorf("création du fichier temporaire dans %s: %w", dir, err)
	}
	tmpName := tmp.Name()

	// Nettoyage du temporaire en cas d'échec à n'importe quelle étape.
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("écriture: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("fermeture: %w", err)
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renommage vers %s: %w", path, err)
	}
	return nil
}
