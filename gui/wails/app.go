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

// onStartup mémorise le contexte Wails et enregistre le gestionnaire de glisser-déposer.
// En Wails v2, la réception des fichiers déposés passe par runtime.OnFileDrop (et non par un
// champ d'options.App) ; DragAndDrop.EnableFileDrop dans les options active le mécanisme.
func (a *App) onStartup(ctx context.Context) {
	a.ctx = ctx
	wailsruntime.OnFileDrop(ctx, a.onFileDrop)
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
