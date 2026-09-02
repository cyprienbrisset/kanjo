package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// isTTY indique si le descripteur est un terminal interactif (sans dépendance externe).
func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// readInput lit un fichier, ou l'entrée standard si le chemin est "-".
func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s: %w", path, err)
	}
	return data, nil
}

// sha256hex renvoie l'empreinte SHA-256 hexadécimale de data.
func sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// outputFormat détermine le format de sortie machine : json si --format json ou si la sortie
// n'est pas un terminal ; table sinon (§11.1).
func outputFormat(explicit string) string {
	switch explicit {
	case "json", "table", "yaml", "csv":
		return explicit
	case "":
		if isTTY(os.Stdout) {
			return "table"
		}
		return "json"
	default:
		return explicit
	}
}
