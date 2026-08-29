// Package formats est le point d'assemblage du registre : il importe « à blanc » tous les
// lecteurs et écrivains concrets afin de déclencher leur enregistrement (init). Les façades
// (CLI, TUI, studio) et les tests d'intégration importent ce paquet une seule fois.
//
// Séparer le câblage évite les cycles : pkg/read et pkg/write ne connaissent pas leurs
// sous-paquets ; c'est ici qu'on les réunit.
package formats

import (
	// Lecteurs
	_ "github.com/cyprienbrisset/kanjo/pkg/read/cii"
	_ "github.com/cyprienbrisset/kanjo/pkg/read/facturx"
	_ "github.com/cyprienbrisset/kanjo/pkg/read/fatturapa"
	_ "github.com/cyprienbrisset/kanjo/pkg/read/ubl"
	_ "github.com/cyprienbrisset/kanjo/pkg/read/zugferd1"
	// Écrivains
	_ "github.com/cyprienbrisset/kanjo/pkg/write/cii"
	_ "github.com/cyprienbrisset/kanjo/pkg/write/peppol"
	_ "github.com/cyprienbrisset/kanjo/pkg/write/tabular"
	_ "github.com/cyprienbrisset/kanjo/pkg/write/ubl"
	_ "github.com/cyprienbrisset/kanjo/pkg/write/xrechnung"
	// Le JSON pivot est enregistré directement dans pkg/read et pkg/write (init).
)
