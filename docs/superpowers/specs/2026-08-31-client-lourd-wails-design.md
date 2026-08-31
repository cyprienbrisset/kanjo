# Client lourd Kanjō (application de bureau native) — Design

- **Date** : 2026-08-31
- **Statut** : validé (brainstorming), prêt pour plan d'implémentation
- **Lot** : L3 (Studio), sous-projet « client lourd »
- **Auteur** : session Claude Code + cyprienbrisset

## 1. Contexte et objectif

Kanjō dispose déjà de trois façades : CLI (`kanjo`), TUI (`kanjo tui`) et **Studio web**
(`kanjo studio` — serveur loopback + frontend `gui/frontend` servi dans le navigateur).
Le dossier `gui/wails/` est un placeholder vide : le **client lourd natif** n'existe pas.

Objectif : livrer une **application de bureau native** multiplateforme (macOS, Windows, Linux)
qui **réutilise** le frontend et l'API Studio existants, et ajoute ce qu'un navigateur ne peut
pas offrir :

1. **Fenêtre native lançable au clic** (`.app`/`.exe`/AppImage), sans terminal ni navigateur.
2. **Intégration OS** : associations de fichiers, glisser-déposer natif, dialogues de fichiers
   natifs, menu applicatif, icône dock/barre des tâches.
3. **Distribution empaquetée** : installeurs par OS, signature/notarisation en option.

Le périmètre fonctionnel initial est **la réutilisation du frontend existant**
(valider / inspecter / convertir). Les écrans riches G1-G13 du cahier des charges sont
**explicitement hors périmètre** (YAGNI ; sous-projets ultérieurs).

## 2. Contrainte structurante : CGO vs cœur pur Go

Le `CLAUDE.md` (règle 3) impose que le cœur compile en `CGO_ENABLED=0` sur 6 cibles
(`make build-all`). Or **Wails exige CGO** et les SDK natifs de chaque OS (WebKit macOS,
WebView2 Windows, WebKitGTK Linux) et **ne se cross-compile pas**.

**Décision** : l'application de bureau vit dans un **module Go séparé** (`gui/wails/go.mod`),
donc **invisible** de `go build ./...` et `make build-all` exécutés à la racine (Go ne descend
jamais dans un module imbriqué). Toute la dette CGO/native est confinée dans un artefact
**opt-in**. Le `go.mod` racine reste inchangé, la règle CI n°3 est mécaniquement préservée.

Cette décision d'architecture est actée par un **ADR daté** dans `docs/adr/`.

## 3. Approche retenue

**Approche A — Wails v2 + réutilisation de l'API Studio in-process.** (Approches B « bindings
Go natifs » et C « webview minimal » écartées : B forke le frontend sans bénéfice ; C ne fournit
ni intégration OS ni packaging.)

## 4. Architecture

### 4.1 Arborescence du module

```
gui/wails/
  go.mod                 # module distinct ; require wails ; replace kanjo => ../..
  go.sum
  main.go                # entrée Wails : options AssetServer, menu, hooks de cycle de vie
  app.go                 # struct App : OnStartup/OnShutdown, méthodes bindées
  wails.json             # config projet Wails (nom, auteur, frontend:dir)
  build/                 # assets natifs par OS
    appicon.png
    darwin/Info.plist    # CFBundleDocumentTypes (associations .xml/.pdf)
    windows/             # NSIS + manifeste associations
    linux/               # .desktop (MimeType), métadonnées AppImage/.deb
  internal/desktop/      # service métier testable, SANS dépendance webview
    service.go           # ProcessFiles(paths) -> résultats (validate/inspect/convert)
    service_test.go
```

### 4.2 Réutilisation (aucune duplication)

- `replace github.com/cyprienbrisset/kanjo => ../..` : l'app importe `pkg/*`,
  `cmd/kanjo/studio` et `gui/frontend` du module racine.
- **Frontend inchangé** : `main.go` configure `options.AssetServer{ Assets: frontend.Assets }`
  en pointant sur l'`embed.FS` existant (`gui/frontend`, `//go:embed all:dist`). Même
  `index.html`/`app.js`/`tokens.css` qu'en Studio-web.
- **API inchangée** : `options.AssetServer{ Middleware: … }` route `/api/*` vers
  `studio.NewHandler(token)` **réutilisé tel quel** (déjà couvert par `studio_test.go`).
- **Tout en intra-processus** : aucun port réseau ouvert, aucune télémétrie (règle 8).
  L'app fonctionne hors-ligne par construction.

### 4.3 Jeton de session

Un token est généré au démarrage et surfacé au frontend servi par Wails (URL initiale ou
variable globale injectée), de sorte que le middleware `withToken` du handler Studio reste
actif — même mécanisme d'authz qu'en Studio-web. La source exacte du token lue par `app.js`
sera vérifiée à l'implémentation et respectée à l'identique.

## 5. Intégration OS

Toute la logique passe par le service testable `internal/desktop` ; les appels `runtime.*` de
Wails ne sont qu'une glue fine.

- **Dialogues natifs** : méthode bindée `OpenInvoices()` → `runtime.OpenMultipleFilesDialog`
  (filtres `.xml`/`.pdf`) → `desktop.ProcessFiles` → résultat JSON renvoyé au frontend.
- **Glisser-déposer natif** : `OnFileDrop` (Wails ≥ 2.5) fournit les **chemins réels** des
  fichiers lâchés → même pipeline `desktop.ProcessFiles`.
- **Associations de fichiers** : déclarées dans `build/` par OS ; double-clic sur une facture
  lance Kanjō avec le chemin → traité au démarrage (`OnStartup` + args, gestion de la seconde
  instance) → écran de résultat direct.
- **Menu applicatif natif** : Fichier (Ouvrir, Ouvrir un dossier…), Édition, Aide
  (Documentation, À propos), Quitter — raccourcis standard par OS.
- **Coquille fenêtre** : titre, icône dock/barre des tâches, taille mémorisée.

## 6. Build, packaging, CI

- **Build par OS** (runners natifs, pas de cross-compilation) :
  - `macos-latest` → `.app` + `.dmg`
  - `windows-latest` → `.exe` + installeur NSIS
  - `ubuntu-latest` → binaire + AppImage / `.deb`
- **CI** : nouveau workflow `.github/workflows/desktop.yml`, déclenché sur tag `v*` et
  `workflow_dispatch`, **séparé** de `release.yml` (les binaires CLI pur-Go restent intacts).
  Un job **compile-gate** sur PR construit l'app sur les 3 OS (sans signer) → garde-fou de
  non-régression du build natif.
- **Signature (opt-in, secrets fournis par le mainteneur)**, dégradation gracieuse : secrets
  absents ⇒ artefacts **non signés** produits quand même, jamais d'échec silencieux.
  - macOS : Developer ID + notarisation (`APPLE_ID`, `APPLE_CERT`, `APPLE_TEAM_ID`…)
  - Windows : cert OV/EV (`WIN_CSC_*`)
  - Linux : non signé (usage)

## 7. Tests (règle 13 : code + tests + CHANGELOG + doc)

- `internal/desktop` : tests Go unitaires du service métier (lecture fichier → verdict),
  **sans webview** — c'est là qu'est la logique substantielle.
- Réutilisation des tests existants `cmd/kanjo/studio/studio_test.go` (couche API partagée).
- **Compile-gate CI** sur 3 OS = garde-fou de non-régression du build natif.
- La webview elle-même n'est pas testée unitairement (comme Studio-web) ; e2e éventuel plus tard.

## 8. Conformité CLAUDE.md

- **Règle 1** : ADR daté actant Wails/CGO + le module séparé, avant implémentation.
- **Règle 3** : cœur `CGO_ENABLED=0` × 6 cibles **inchangé** (module séparé, `go.mod` racine intact).
- **Règle 8** : aucun appel réseau, aucune télémétrie (tout in-process/loopback).
- **Règle 13** : code + tests + entrée CHANGELOG + doc utilisateur (section « Client lourd »,
  relation avec `kanjo studio`).

## 9. Hors périmètre (YAGNI)

Écrans G1-G13, SQLite, journal d'audit natif, auto-update, `repair`/`generate`/`anonymize`
natifs. Chacun fera l'objet d'un sous-projet dédié (spec → plan → implémentation).

## 10. Risques et limites connues

- Le **CLI Wails et les SDK natifs ne sont pas installés** dans l'environnement de développement
  actuel : le code + la configuration seront écrits intégralement, mais le `wails build` réel se
  validera en **CI** (runners natifs) et/ou sur la machine du mainteneur.
- La **signature/notarisation** dépend de certificats propres au mainteneur (non fournissables
  par l'agent) : câblée mais inactive tant que les secrets ne sont pas configurés.
- **Wails v2** est la cible (stable) ; Wails v3 reste en évolution et n'est pas retenu.
