package main

import (
	"context"

	"github.com/cyprienbrisset/kanjo/internal/desktop"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App porte le contexte Wails et expose les méthodes bindées au frontend.
type App struct {
	ctx context.Context
	// pending : fichiers passés au lancement (association/double-clic). Le frontend les
	// réclame via PendingFiles() une fois son écouteur prêt (modèle « pull »).
	pending []string
}

func NewApp(launchArgs []string) *App {
	return &App{pending: filterInvoicePaths(launchArgs)}
}

// onStartup mémorise le contexte Wails et enregistre le gestionnaire de glisser-déposer.
// En Wails v2, la réception des fichiers déposés passe par runtime.OnFileDrop (et non par un
// champ d'options.App) ; DragAndDrop.EnableFileDrop dans les options active le mécanisme.
func (a *App) onStartup(ctx context.Context) {
	a.ctx = ctx
	wailsruntime.OnFileDrop(ctx, a.onFileDrop)
}

// PendingFiles renvoie les fichiers passés au lancement (association/double-clic) puis vide
// la file d'attente. Le frontend l'appelle une fois son écouteur enregistré (modèle « pull »).
//
// Pourquoi un pull plutôt qu'un push depuis OnDomReady : en Wails v2, OnDomReady se déclenche
// avant que le frontend n'ait enregistré son écouteur EventsOn (fait sur l'évènement `load`).
// Un EventsEmit émis à ce moment est mis en tampon par Wails, mais ce tampon a une limite :
// les petits lots passaient, les lots volumineux étaient silencieusement perdus — d'où
// l'affichage vide au chargement de plusieurs documents. Le pull supprime la course.
func (a *App) PendingFiles() ([]desktop.FileData, error) {
	if len(a.pending) == 0 {
		return nil, nil
	}
	files, err := desktop.ReadFiles(a.pending)
	a.pending = nil
	return files, err
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
