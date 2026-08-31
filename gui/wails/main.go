package main

import (
	_ "embed"
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
		OnStartup:                app.onStartup,
		OnDomReady:               app.onDomReady,
		EnableDefaultContextMenu: false,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		Bind: []interface{}{app},
		Menu: buildMenu(app),
	})
	if err != nil {
		panic(err)
	}
	_ = appIcon
}
