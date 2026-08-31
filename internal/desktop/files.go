// Package desktop porte la logique du client lourd indépendante de toute webview :
// lecture de fichiers vers la forme attendue par le frontend ({name, data base64}).
// Pur Go, aucune dépendance CGO — testable et compilé par la CI du cœur.
package desktop

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// MaxFileSize borne la taille d'un fichier lu (parité avec studio.maxUpload : 64 Mo).
const MaxFileSize = 64 << 20

// FileData est une pièce transmise au frontend : nom d'affichage + contenu en base64.
// Les balises JSON `name`/`data` correspondent au contrat de app.js (window.kanjoOpenFiles).
type FileData struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// ReadFiles lit chaque chemin et renvoie son contenu encodé en base64. Échec immédiat
// si un chemin est inaccessible ou dépasse MaxFileSize.
func ReadFiles(paths []string) ([]FileData, error) {
	return readFiles(paths, MaxFileSize)
}

// readFiles porte la logique de ReadFiles avec une borne de taille paramétrable,
// afin de tester la branche « fichier trop gros » sans créer un fichier de 64 Mo.
func readFiles(paths []string, maxSize int64) ([]FileData, error) {
	out := make([]FileData, 0, len(paths))
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("accès à %s: %w", p, err)
		}
		if fi.Size() > maxSize {
			return nil, fmt.Errorf("%s dépasse la taille maximale (%d octets)", p, maxSize)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("lecture de %s: %w", p, err)
		}
		out = append(out, FileData{Name: filepath.Base(p), Data: base64.StdEncoding.EncodeToString(b)})
	}
	return out, nil
}
