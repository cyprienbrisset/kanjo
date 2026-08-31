# Client lourd Kanjō (Wails) — Plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Livrer une application de bureau native (macOS/Windows/Linux) qui enveloppe le frontend et l'API Studio existants, ajoute l'intégration OS (dialogues natifs, glisser-déposer, associations de fichiers, menu) et un packaging par OS.

**Architecture:** Module Go **séparé** `gui/wails/` (CGO/Wails), invisible du cœur pur Go. Le frontend `gui/frontend` et le handler `cmd/kanjo/studio` sont **réutilisés tels quels** (servis en intra-processus par Wails). Un pont natif déjà anticipé par `app.js` (`window.kanjoOpenFiles`) est fourni par un binding Go. La logique testable (lecture de fichiers) vit dans un paquet racine `internal/desktop`, testé par la CI pure-Go.

**Tech Stack:** Go 1.23+, Wails v2 (CGO), WebKit/WebView2/WebKitGTK, GitHub Actions (runners natifs), NSIS/dmg/AppImage.

---

## Contexte pour l'ingénieur (à lire avant de commencer)

Faits établis par l'exploration du code — **ne pas les redécouvrir** :

1. `cmd/kanjo/studio/studio.go` expose déjà :
   - `func NewHandler(token string) http.Handler` — sert le frontend embarqué + `/api/{version,formats,validate,inspect,convert}`.
   - `func NewToken() string` — génère un jeton de session (commentaire : « exposé pour le client lourd desktop »).
   - `serveIndex` remplace `__TOKEN__` dans `index.html` par le jeton (balise `<meta name="kanjo-token">`).
2. `gui/frontend/embed.go` : `//go:embed all:dist` → `var Assets embed.FS`.
3. `gui/frontend/dist/app.js` **anticipe déjà le client lourd** :
   - Lit le jeton via `document.querySelector('meta[name="kanjo-token"]').content`.
   - Si `window.kanjoOpenFiles` existe : masque `<input type=file>`, montre le bouton `#pick`, et sur clic appelle `window.kanjoOpenFiles()` en attendant une `Promise<[{name, data}]>` où `data` est du **base64**. Il décode puis appelle `inspectBytes(name, buffer)` qui POSTe vers `/api/inspect`.
   - `app.js` est un **script classique** (`<script src="app.js">`), donc ses fonctions de haut niveau (`inspectBytes`, `handleFiles`, `show`, `renderDocList`) sont des **globales** (`window.inspectBytes`, …).
4. `gui/frontend/dist/index.html` : `<meta name="kanjo-token" content="__TOKEN__">`, en-tête avec `-webkit-app-region:drag` (prêt pour fenêtre native), `<script src="app.js">` en fin de `<body>`.
5. `gui/wails/` ne contient qu'un `.gitkeep` (placeholder vide).
6. Le cahier des charges référence ADR-001 à ADR-010 ; le nouvel ADR est donc **0011**.

**Contrainte dure (`CLAUDE.md` règle 3)** : `make build-all` en `CGO_ENABLED=0` doit continuer de compiler sur 6 cibles. Wails exige CGO → il est **isolé dans un module Go séparé** (`gui/wails/go.mod`), donc **exclu** de `go build ./...` et de la matrice pure-Go à la racine.

**Limite d'environnement** : le CLI Wails et les SDK natifs ne sont pas installés ici. Les tâches 1 et 2 sont **exécutables et vérifiables immédiatement** (pur Go / assets). Les tâches 3-8 (module Wails, build natif) s'écrivent intégralement mais se **valident en CI / sur machine du mainteneur** ; leurs « Run » Wails sont marqués `[CI]`.

---

## Structure des fichiers

| Fichier | Rôle | Module |
|---|---|---|
| `docs/adr/0011-client-lourd-wails.md` | ADR : Wails + CGO + module séparé | racine (doc) |
| `internal/desktop/files.go` | `FileData` + `ReadFiles(paths)` (lecture → base64) | **racine** (pur Go) |
| `internal/desktop/files_test.go` | tests de `ReadFiles` | racine |
| `gui/frontend/dist/native-bridge.js` | pont natif (alias `kanjoOpenFiles` + écoute d'événements) | racine (asset) |
| `gui/wails/go.mod` / `go.sum` | module Wails séparé (`replace` vers racine) | **wails** |
| `gui/wails/wails.json` | config projet Wails | wails |
| `gui/wails/app.go` | struct `App` : cycle de vie, `OpenFiles`, drop, args | wails |
| `gui/wails/main.go` | options Wails : AssetServer=handler Studio, menu, bind | wails |
| `gui/wails/build/` | icônes + Info.plist + .desktop + manifeste associations | wails |
| `.github/workflows/desktop.yml` | build 3 OS (compile-gate + artefacts) | racine |
| `Makefile` | cible `desktop` | racine |
| `CHANGELOG.md`, `docs/documentation.html` | entrée + section utilisateur | racine |

---

## Task 0 : ADR — décision Wails/CGO + module séparé

**Files:**
- Create: `docs/adr/0011-client-lourd-wails.md`

- [ ] **Step 1 : Écrire l'ADR**

```markdown
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
```

- [ ] **Step 2 : Commit**

```bash
git add docs/adr/0011-client-lourd-wails.md
git commit -m "docs(adr): 0011 — client lourd Wails, CGO et module séparé"
```

---

## Task 1 : `internal/desktop` — lecture de fichiers (pur Go, testable maintenant)

**Files:**
- Create: `internal/desktop/files.go`
- Test: `internal/desktop/files_test.go`

- [ ] **Step 1 : Écrire le test qui échoue**

```go
package desktop

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFilesEncodesBase64(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "facture.xml")
	content := []byte(`<?xml version="1.0"?><rsm:CrossIndustryInvoice/>`)
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := ReadFiles([]string{p})
	if err != nil {
		t.Fatalf("ReadFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("attendu 1 fichier, obtenu %d", len(files))
	}
	if files[0].Name != "facture.xml" {
		t.Errorf("Name = %q, veut facture.xml", files[0].Name)
	}
	got, err := base64.StdEncoding.DecodeString(files[0].Data)
	if err != nil {
		t.Fatalf("Data n'est pas du base64 valide: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("contenu décodé = %q, veut %q", got, content)
	}
}

func TestReadFilesMissingPathErrors(t *testing.T) {
	if _, err := ReadFiles([]string{"/introuvable/xyz.xml"}); err == nil {
		t.Fatal("un chemin introuvable doit produire une erreur")
	}
}
```

- [ ] **Step 2 : Lancer le test — vérifier l'échec**

Run: `go test ./internal/desktop/ -run TestReadFiles -v`
Expected: FAIL (`undefined: ReadFiles`, `undefined: FileData`).

- [ ] **Step 3 : Écrire l'implémentation minimale**

```go
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
	out := make([]FileData, 0, len(paths))
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("accès à %s: %w", p, err)
		}
		if fi.Size() > MaxFileSize {
			return nil, fmt.Errorf("%s dépasse la taille maximale (%d octets)", p, int64(MaxFileSize))
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("lecture de %s: %w", p, err)
		}
		out = append(out, FileData{Name: filepath.Base(p), Data: base64.StdEncoding.EncodeToString(b)})
	}
	return out, nil
}
```

- [ ] **Step 4 : Lancer le test — vérifier le succès**

Run: `go test ./internal/desktop/ -run TestReadFiles -v`
Expected: PASS (les deux tests).

- [ ] **Step 5 : Vérifier que le cœur compile toujours en pur Go**

Run: `CGO_ENABLED=0 go build ./...`
Expected: aucune erreur (le nouveau paquet n'introduit aucune dépendance CGO).

- [ ] **Step 6 : Commit**

```bash
git add internal/desktop/files.go internal/desktop/files_test.go
git commit -m "feat(desktop): lecture de fichiers vers {name, data base64} pour le client lourd"
```

---

## Task 2 : Pont natif frontend (`native-bridge.js`)

Ce fichier est un **asset statique** ajouté au frontend embarqué. `index.html` **ne le référence pas** → Studio-web ne le charge jamais (comportement web inchangé). Seul le module Wails l'injectera (Task 5). Il définit l'alias `window.kanjoOpenFiles` attendu par `app.js`, et écoute un événement Wails pour les fichiers ouverts par l'OS (double-clic, glisser-déposer natif).

**Files:**
- Create: `gui/frontend/dist/native-bridge.js`
- Test: `gui/frontend/native_bridge_test.go`

- [ ] **Step 1 : Écrire le test qui échoue (l'asset doit être embarqué et servi)**

```go
package frontend

import (
	"io/fs"
	"strings"
	"testing"
)

func TestNativeBridgeEmbedded(t *testing.T) {
	b, err := fs.ReadFile(Assets, "dist/native-bridge.js")
	if err != nil {
		t.Fatalf("native-bridge.js doit être embarqué: %v", err)
	}
	s := string(b)
	// Contrat avec app.js : l'alias attendu et le binding Wails ciblé.
	for _, want := range []string{"window.kanjoOpenFiles", "window.go.main.App.OpenFiles", "kanjo:open-files"} {
		if !strings.Contains(s, want) {
			t.Errorf("native-bridge.js doit référencer %q", want)
		}
	}
}
```

- [ ] **Step 2 : Lancer le test — vérifier l'échec**

Run: `go test ./gui/frontend/ -run TestNativeBridgeEmbedded -v`
Expected: FAIL (`native-bridge.js` absent de l'embed).

- [ ] **Step 3 : Écrire le pont natif**

```javascript
'use strict';
// Pont natif du client lourd Kanjō. Chargé UNIQUEMENT par l'application Wails
// (injecté avant app.js) ; jamais référencé par index.html, donc Studio-web l'ignore.
// Il n'a d'effet que si les bindings Wails (window.go) sont présents.

// 1) Sélecteur de fichiers natif : app.js appelle window.kanjoOpenFiles() et attend
//    une Promise<[{name, data(base64)}]>. On l'aliase vers le binding Go OpenFiles.
//    (La fonction est définie tout de suite ; window.go n'est requis qu'à l'appel.)
window.kanjoOpenFiles = function () {
  if (!(window.go && window.go.main && window.go.main.App)) return Promise.resolve([]);
  return window.go.main.App.OpenFiles();
};

// 2) Fichiers ouverts par l'OS (double-clic sur une association, glisser-déposer natif) :
//    le Go émet l'événement 'kanjo:open-files' avec [{name, data(base64)}]. On réutilise
//    les fonctions globales déjà définies par app.js (script classique) pour valider+afficher.
window.addEventListener('load', function () {
  if (!(window.runtime && typeof window.runtime.EventsOn === 'function')) return;
  window.runtime.EventsOn('kanjo:open-files', function (files) {
    if (!files || !files.length || typeof window.inspectBytes !== 'function') return;
    Promise.all(files.map(function (f) {
      var bin = atob(f.data);
      var bytes = new Uint8Array(bin.length);
      for (var i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
      return window.inspectBytes(f.name, bytes.buffer);
    })).then(function () {
      if (typeof window.show === 'function') window.show('ken');
      if (typeof window.renderDocList === 'function') window.renderDocList();
    });
  });
});
```

- [ ] **Step 4 : Lancer le test — vérifier le succès**

Run: `go test ./gui/frontend/ -run TestNativeBridgeEmbedded -v`
Expected: PASS.

- [ ] **Step 5 : Commit**

```bash
git add gui/frontend/dist/native-bridge.js gui/frontend/native_bridge_test.go
git commit -m "feat(frontend): pont natif (kanjoOpenFiles + événement kanjo:open-files) pour le client lourd"
```

---

## Task 3 : Scaffold du module Wails séparé

**Files:**
- Create: `gui/wails/go.mod`
- Create: `gui/wails/wails.json`
- Remove: `gui/wails/.gitkeep`

- [ ] **Step 1 : Créer `gui/wails/go.mod`**

```
module github.com/cyprienbrisset/kanjo/gui/wails

go 1.23

require github.com/wailsapp/wails/v2 v2.10.1

require github.com/cyprienbrisset/kanjo v0.0.0

replace github.com/cyprienbrisset/kanjo => ../..
```

> Note : la version exacte de Wails sera figée par `go mod tidy` (Step 4). Les dépendances
> transitives de Wails (CGO) restent confinées à ce `go.sum`, jamais au `go.mod` racine.

- [ ] **Step 2 : Créer `gui/wails/wails.json`**

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "Kanjo",
  "outputfilename": "Kanjo",
  "frontend:dir": "",
  "author": {
    "name": "Wakastellar"
  },
  "info": {
    "productName": "Kanjō",
    "productVersion": "0.0.0",
    "comments": "Facturation électronique — client lourd (réutilise le cœur Kanjō)."
  }
}
```

> `frontend:dir` est laissé vide : les assets sont servis en Go via l'AssetServer (Task 5),
> pas par un pipeline npm. Wails ne lancera aucun build front.

- [ ] **Step 3 : Supprimer le placeholder**

```bash
git rm gui/wails/.gitkeep
```

- [ ] **Step 4 : Résoudre les dépendances `[CI]` (nécessite le réseau + toolchain Wails)**

Run (sur machine avec Go + réseau) : `cd gui/wails && go mod tidy`
Expected: `go.sum` généré, version Wails figée. **Ne bloque pas** les tâches racine.

- [ ] **Step 5 : Commit**

```bash
git add gui/wails/go.mod gui/wails/wails.json
git rm --cached gui/wails/.gitkeep 2>/dev/null || true
git commit -m "feat(wails): scaffold du module de bureau séparé (isolation CGO)"
```

---

## Task 4 : `app.go` — struct App, dialogue natif, drop, associations

**Files:**
- Create: `gui/wails/app.go`

- [ ] **Step 1 : Écrire `app.go`**

```go
package main

import (
	"context"

	"github.com/cyprienbrisset/kanjo/internal/desktop"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App porte le contexte Wails et expose les méthodes bindées au frontend.
type App struct {
	ctx context.Context
	// pending : fichiers passés au lancement (association/double-clic) avant que le DOM
	// ne soit prêt à recevoir l'événement.
	pending []string
}

func NewApp(launchArgs []string) *App {
	return &App{pending: filterInvoicePaths(launchArgs)}
}

// onStartup mémorise le contexte Wails.
func (a *App) onStartup(ctx context.Context) {
	a.ctx = ctx
}

// onDomReady : le frontend est chargé ; on pousse les éventuels fichiers de lancement.
func (a *App) onDomReady(_ context.Context) {
	if len(a.pending) == 0 {
		return
	}
	a.emitFiles(a.pending)
	a.pending = nil
}

// OpenFiles ouvre le dialogue natif et renvoie les fichiers choisis en base64.
// Appelé par le frontend via window.kanjoOpenFiles (voir native-bridge.js).
func (a *App) OpenFiles() ([]desktop.FileData, error) {
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Ouvrir des factures",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Factures (*.xml, *.pdf, *.json)", Pattern: "*.xml;*.pdf;*.json"},
		},
	})
	if err != nil {
		return nil, err
	}
	return desktop.ReadFiles(paths)
}

// onFileDrop reçoit les chemins réels d'un glisser-déposer natif sur la fenêtre.
func (a *App) onFileDrop(_, _ int, paths []string) {
	a.emitFiles(filterInvoicePaths(paths))
}

// emitFiles lit les fichiers et émet 'kanjo:open-files' vers le frontend.
func (a *App) emitFiles(paths []string) {
	if len(paths) == 0 {
		return
	}
	files, err := desktop.ReadFiles(paths)
	if err != nil || len(files) == 0 {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "kanjo:open-files", files)
}
```

- [ ] **Step 2 : Créer `gui/wails/paths.go` (helper testable)**

```go
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
```

- [ ] **Step 3 : Test du helper (dans le module wails)**

Create `gui/wails/paths_test.go` :

```go
package main

import "testing"

func TestFilterInvoicePaths(t *testing.T) {
	in := []string{"/usr/bin/Kanjo", "-flag", "/a/facture.xml", "/b/scan.PDF", "/c/note.txt"}
	got := filterInvoicePaths(in)
	if len(got) != 2 || got[0] != "/a/facture.xml" || got[1] != "/b/scan.PDF" {
		t.Fatalf("filtrage inattendu: %v", got)
	}
}
```

- [ ] **Step 4 : Lancer le test `[CI]` (module wails, nécessite `go mod tidy` fait)**

Run: `cd gui/wails && go test -run TestFilterInvoicePaths ./...`
Expected: PASS. (`paths_test.go` ne dépend pas de CGO ; compile dès que le module est résolu.)

- [ ] **Step 5 : Commit**

```bash
git add gui/wails/app.go gui/wails/paths.go gui/wails/paths_test.go
git commit -m "feat(wails): App (dialogue natif, glisser-déposer, associations) + filtrage des chemins"
```

---

## Task 5 : `main.go` — options Wails, réutilisation du handler Studio, injection du pont

**Files:**
- Create: `gui/wails/main.go`

- [ ] **Step 1 : Écrire `main.go`**

```go
package main

import (
	"embed"
	"net/http"
	"os"
	"strings"

	"github.com/cyprienbrisset/kanjo/cmd/kanjo/studio"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// icône embarquée pour la fenêtre/dock (placée en Task 6).
//
//go:embed build/appicon.png
var appIcon []byte

// bridgeMiddleware injecte native-bridge.js AVANT app.js dans index.html, uniquement
// dans l'application native. Le handler Studio (réutilisé) sert par ailleurs l'asset.
func bridgeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			rec := &capture{header: http.Header{}, code: 200}
			next.ServeHTTP(rec, r)
			html := strings.Replace(
				rec.buf.String(),
				`<script src="app.js">`,
				`<script src="native-bridge.js"></script><script src="app.js">`,
				1,
			)
			for k, v := range rec.header {
				w.Header()[k] = v
			}
			w.Header().Del("Content-Length")
			w.WriteHeader(rec.code)
			_, _ = w.Write([]byte(html))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	app := NewApp(os.Args[1:])
	handler := studio.NewHandler(studio.NewToken())

	err := wails.Run(&options.App{
		Title:  "Kanjō",
		Width:  1100,
		Height: 720,
		AssetServer: &assetserver.Options{
			Handler:    handler,
			Middleware: bridgeMiddleware,
		},
		OnStartup:        app.onStartup,
		OnDomReady:       app.onDomReady,
		EnableDefaultContextMenu: false,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		OnFileDrop: app.onFileDrop,
		Bind:       []interface{}{app},
		Menu:       buildMenu(app),
	})
	if err != nil {
		panic(err)
	}
	_ = appIcon
}
```

- [ ] **Step 2 : Créer le petit `capture` (ResponseWriter tampon) `gui/wails/capture.go`**

```go
package main

import (
	"bytes"
	"net/http"
)

// capture bufferise une réponse HTTP pour permettre la réécriture d'index.html.
type capture struct {
	header http.Header
	buf    bytes.Buffer
	code   int
}

func (c *capture) Header() http.Header       { return c.header }
func (c *capture) WriteHeader(code int)      { c.code = code }
func (c *capture) Write(b []byte) (int, error) { return c.buf.Write(b) }
```

- [ ] **Step 3 : Créer le menu natif `gui/wails/menu.go`**

```go
package main

import (
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// buildMenu construit le menu applicatif natif : Fichier (Ouvrir…, Quitter) et Aide.
func buildMenu(a *App) *menu.Menu {
	m := menu.NewMenu()
	file := m.AddSubmenu("Fichier")
	file.AddText("Ouvrir des factures…", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		files, err := a.OpenFiles()
		if err == nil && len(files) > 0 {
			wailsruntime.EventsEmit(a.ctx, "kanjo:open-files", files)
		}
	})
	file.AddSeparator()
	file.AddText("Quitter", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		wailsruntime.Quit(a.ctx)
	})
	help := m.AddSubmenu("Aide")
	help.AddText("Documentation", nil, func(_ *menu.CallbackData) {
		wailsruntime.BrowserOpenURL(a.ctx, "https://cyprienbrisset.github.io/kanjo/")
	})
	return m
}
```

- [ ] **Step 4 : Build de l'application `[CI]`**

Run (macOS/Windows/Linux avec Wails installé) : `cd gui/wails && wails build`
Expected: binaire natif produit (`build/bin/Kanjo.app` / `Kanjo.exe` / `Kanjo`).

- [ ] **Step 5 : Lancer et piloter l'app `[CI/manuel]`**

- Ouvrir l'app → la fenêtre Studio s'affiche, le bouton « Choisir un fichier… » est visible.
- Cliquer → un dialogue **natif** s'ouvre ; choisir `testdata/corpus/published/valides/fatturapa/F2026-0001.xml`.
- Attendu : l'écran Inspecteur montre le document avec le verdict (statut conforme).

- [ ] **Step 6 : Commit**

```bash
git add gui/wails/main.go gui/wails/capture.go gui/wails/menu.go
git commit -m "feat(wails): fenêtre native réutilisant le handler Studio + injection du pont + menu"
```

---

## Task 6 : Assets natifs et associations de fichiers

**Files:**
- Create: `gui/wails/build/appicon.png` (1024×1024, dérivé de `gui/frontend/dist/img/logo-emblem.png`)
- Create: `gui/wails/build/darwin/Info.plist`
- Create: `gui/wails/build/linux/kanjo.desktop`
- Create: `gui/wails/build/windows/installer/associations.nsh`

- [ ] **Step 1 : Icône**

```bash
cp gui/frontend/dist/img/logo-emblem.png gui/wails/build/appicon.png
# Redimensionner en 1024×1024 si un outil est disponible (sips sur macOS) :
# sips -z 1024 1024 gui/wails/build/appicon.png
```

- [ ] **Step 2 : `Info.plist` macOS — associations .xml/.pdf**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Kanjō</string>
  <key>CFBundleExecutable</key><string>Kanjo</string>
  <key>CFBundleIdentifier</key><string>com.wakastellar.kanjo</string>
  <key>CFBundleDocumentTypes</key>
  <array>
    <dict>
      <key>CFBundleTypeName</key><string>Facture électronique</string>
      <key>CFBundleTypeRole</key><string>Viewer</string>
      <key>LSItemContentTypes</key>
      <array><string>public.xml</string><string>com.adobe.pdf</string><string>public.json</string></array>
    </dict>
  </array>
</dict>
</plist>
```

- [ ] **Step 3 : `.desktop` Linux — MimeType**

```ini
[Desktop Entry]
Type=Application
Name=Kanjō
Exec=Kanjo %F
Icon=kanjo
Categories=Office;Finance;
MimeType=application/xml;application/pdf;application/json;
```

- [ ] **Step 4 : NSIS Windows — associations**

```nsis
; associations.nsh — inclus par le template NSIS de Wails.
!macro CustomInstall
  WriteRegStr HKCR ".xml\OpenWithProgids" "Kanjo.Invoice" ""
  WriteRegStr HKCR "Kanjo.Invoice" "" "Facture électronique"
  WriteRegStr HKCR "Kanjo.Invoice\shell\open\command" "" '"$INSTDIR\Kanjo.exe" "%1"'
!macroend
```

- [ ] **Step 5 : Build vérifiant la prise en compte des assets `[CI]`**

Run: `cd gui/wails && wails build`
Expected: `.app` embarque l'`Info.plist` (vérifier `CFBundleDocumentTypes` dans le bundle).

- [ ] **Step 6 : Commit**

```bash
git add gui/wails/build/
git commit -m "feat(wails): icône, Info.plist, .desktop et associations de fichiers par OS"
```

---

## Task 7 : CI — build de bureau multiplateforme

**Files:**
- Create: `.github/workflows/desktop.yml`

- [ ] **Step 1 : Écrire le workflow**

```yaml
name: Desktop

# Compile le client lourd sur les 3 OS. Compile-gate sur PR touchant gui/wails ;
# artefacts publiés sur tag v*. Signature opt-in (secrets), sinon artefacts non signés.
on:
  push:
    tags: ["v*"]
  pull_request:
    paths: ["gui/wails/**", "gui/frontend/**", "internal/desktop/**", ".github/workflows/desktop.yml"]
  workflow_dispatch:

permissions:
  contents: write

jobs:
  build:
    strategy:
      fail-fast: false
      matrix:
        include:
          - os: macos-latest
            platform: darwin/universal
          - os: windows-latest
            platform: windows/amd64
          - os: ubuntu-latest
            platform: linux/amd64
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Dépendances Linux (WebKitGTK)
        if: runner.os == 'Linux'
        run: sudo apt-get update && sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev
      - name: Installer Wails
        run: go install github.com/wailsapp/wails/v2/cmd/wails@latest
      - name: go mod tidy (module wails)
        working-directory: gui/wails
        run: go mod tidy
      - name: Build
        working-directory: gui/wails
        run: wails build -platform ${{ matrix.platform }}
      - name: Publier l'artefact (compile-gate + tag)
        uses: actions/upload-artifact@v4
        with:
          name: kanjo-desktop-${{ matrix.os }}
          path: gui/wails/build/bin/*
```

- [ ] **Step 2 : Vérifier la syntaxe YAML localement**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/desktop.yml'))" && echo OK`
Expected: `OK`.

- [ ] **Step 3 : Commit**

```bash
git add .github/workflows/desktop.yml
git commit -m "ci(desktop): build client lourd sur macOS/Windows/Linux (compile-gate + artefacts)"
```

> Signature/notarisation (opt-in) : à ajouter en étapes conditionnelles `if: secrets.APPLE_ID != ''`
> (macOS : `gon`/`codesign` + notarisation) et `if: secrets.WIN_CSC_LINK != ''` (Windows :
> `signtool`). Documenté dans l'ADR ; non bloquant tant que les secrets ne sont pas fournis.

---

## Task 8 : Makefile, CHANGELOG, documentation

**Files:**
- Modify: `Makefile`
- Modify: `CHANGELOG.md`
- Modify: `docs/documentation.html`

- [ ] **Step 1 : Cible Makefile**

Ajouter à `Makefile` :

```makefile
.PHONY: desktop
desktop: ## Construire le client lourd (nécessite Wails + SDK natif de l'OS courant)
	cd gui/wails && go mod tidy && wails build
```

- [ ] **Step 2 : Vérifier que la cible existe**

Run: `make -n desktop`
Expected: affiche les commandes `cd gui/wails && …` sans les exécuter.

- [ ] **Step 3 : Entrée CHANGELOG (section `[Non publié]`)**

Ajouter sous `## [Non publié]` :

```markdown
### Ajouté (client lourd)
- **Application de bureau native (Wails)** : fenêtre native macOS/Windows/Linux réutilisant
  le frontend et l'API Studio (aucune duplication, tout en intra-processus, hors-ligne).
  Intégration OS : dialogue de fichiers natif (`window.kanjoOpenFiles`), glisser-déposer natif,
  associations de fichiers `.xml`/`.pdf`/`.json`, menu applicatif. Isolée dans un module Go
  séparé (`gui/wails`) : le cœur reste pur Go `CGO_ENABLED=0` × 6 cibles (ADR-0011).
```

- [ ] **Step 4 : Section documentation utilisateur**

Dans `docs/documentation.html`, ajouter une section « Client lourd » expliquant : installation
depuis les artefacts `Desktop`, différence avec `kanjo studio` (même UI, mais fenêtre native +
intégration OS), et double-clic sur une facture pour l'ouvrir. (Reprendre le style des sections
existantes ; contenu autonome, sans dépendance réseau.)

- [ ] **Step 5 : Vérifier le cœur pur Go une dernière fois**

Run: `CGO_ENABLED=0 go build ./... && go test ./internal/desktop/ ./gui/frontend/`
Expected: build OK, tests PASS. (Le module `gui/wails` reste hors de `./...`.)

- [ ] **Step 6 : Commit**

```bash
git add Makefile CHANGELOG.md docs/documentation.html
git commit -m "docs(desktop): cible make, entrée CHANGELOG et section documentation du client lourd"
```

---

## Auto-revue (couverture du spec)

- **§2 isolation CGO** → Tasks 0, 3, et vérifications `CGO_ENABLED=0 go build ./...` (Tasks 1/8). ✓
- **§4.2 réutilisation frontend + handler** → Task 5 (`AssetServer.Handler = studio.NewHandler`). ✓
- **§4.3 jeton** → `studio.NewToken()` + `serveIndex` (meta `__TOKEN__`), inchangé. ✓
- **§5 dialogues natifs** → Task 4 `OpenFiles` + Task 2 pont ; **glisser-déposer** → `OnFileDrop` (Task 4/5) ; **associations** → Task 6 + `onDomReady`/`emitFiles` (Task 4) ; **menu** → Task 5 `menu.go`. ✓
- **§6 build/CI/packaging + signature opt-in** → Task 7. ✓
- **§7 tests** → Task 1 (ReadFiles), Task 2 (embed), Task 4 (filterInvoicePaths), réutilisation `studio_test.go`, compile-gate 3 OS. ✓
- **§8 conformité** → ADR (Task 0), pur Go préservé (vérifs), CHANGELOG + doc (Task 8). ✓
- **§9 hors périmètre** → aucun G1-G13/SQLite/auto-update dans le plan. ✓

Aucune référence à un symbole non défini : `FileData`/`ReadFiles` (Task 1) utilisés en Tasks 4/5 ; `OpenFiles` (Task 4) référencé par `native-bridge.js` (Task 2) et le menu (Task 5) ; événement `kanjo:open-files` cohérent entre Task 2 (écoute) et Task 4/5 (émission).
