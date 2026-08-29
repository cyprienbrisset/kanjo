//go:build !cgo

// Repli sans cgo : le client lourd desktop nécessite une WebView native (cgo). Ce fichier
// garantit que `CGO_ENABLED=0 go build ./...` reste valide (ADR-002) — le binaire `kanjo`
// pur Go n'est pas affecté.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Le client lourd « Kanjō Studio » nécessite une compilation avec cgo :")
	fmt.Fprintln(os.Stderr, "  CGO_ENABLED=1 go build -o \"Kanjō Studio\" ./cmd/kanjo-studio")
	fmt.Fprintln(os.Stderr, "Sans interface graphique native, utilisez : kanjo studio (navigateur) ou kanjo tui.")
	os.Exit(6)
}
