// Commande kanjo — point d'entrée du binaire (CLI + TUI + studio).
// Voir docs/CAHIER-DES-CHARGES.md §11 et §24.
package main

import (
	"os"

	"github.com/cyprienbrisset/kanjo/cmd/kanjo/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
