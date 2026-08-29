// Package frontend embarque les assets du frontend de Kanjō Studio via go:embed (ADR-006),
// afin de distribuer un seul artefact sans dépendance externe.
package frontend

import "embed"

//go:embed all:dist
var Assets embed.FS
