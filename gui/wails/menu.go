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
