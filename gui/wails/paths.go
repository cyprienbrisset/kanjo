package main

import (
	"path/filepath"
	"strings"
)

// filterInvoicePaths ne garde que les chemins d'extension reconnue (.xml/.pdf/.json),
// écartant le nom de l'exécutable et les drapeaux passés au lancement.
func filterInvoicePaths(args []string) []string {
	var out []string
	for _, a := range args {
		switch strings.ToLower(filepath.Ext(a)) {
		case ".xml", ".pdf", ".json":
			out = append(out, a)
		}
	}
	return out
}
