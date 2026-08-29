package cli

import "github.com/cyprienbrisset/kanjo/cmd/kanjo/tui"

// runTUI lance l'interface texte sur un chemin (défaut : dossier courant).
func runTUI(args []string) int {
	path := "."
	if len(args) > 0 && args[0] != "" {
		path = args[0]
	}
	return tui.Run(path)
}
