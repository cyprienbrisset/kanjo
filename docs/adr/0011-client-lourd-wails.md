# ADR 0011 — Client lourd : Wails, CGO et module Go séparé

- Statut : accepté
- Date : 2026-08-31
- Contexte lié : docs/superpowers/specs/2026-08-31-client-lourd-wails-design.md

## Contexte

Kanjō doit fournir une application de bureau native (fenêtre lançable au clic,
intégration OS, distribution empaquetée) sur macOS/Windows/Linux. Le cœur du projet
impose `CGO_ENABLED=0` sur 6 cibles (règle 3). Wails, retenu pour la webview native,
exige CGO et les SDK natifs de chaque OS, et ne se cross-compile pas.

## Décision

1. L'application de bureau est un **module Go séparé** (`gui/wails/go.mod`) qui référence
   le module racine via `replace`. Elle est donc **exclue** de `go build ./...` et de
   `make build-all` exécutés à la racine : le cœur reste pur Go, `CGO_ENABLED=0` × 6 cibles.
2. Elle **réutilise** le frontend (`gui/frontend`) et le handler HTTP (`cmd/kanjo/studio`)
   servis **en intra-processus** — aucune duplication, aucun port réseau, aucune télémétrie.
3. La logique métier testable (lecture de fichiers) vit dans `internal/desktop` (module
   racine, pur Go, couverte par la CI).

## Conséquences

- Le build natif se fait sur des runners par OS (`desktop.yml`), séparément des binaires CLI.
- La signature/notarisation est opt-in (secrets du mainteneur), en dégradation gracieuse.
- Wails v2 (stable) est la cible ; v3 non retenu.
