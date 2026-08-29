//go:build cgo

// Commande kanjo-studio — client lourd desktop de Kanjō Studio (ADR-005, §19.1).
//
// Il embarque une fenêtre WebView native (WebKit sur macOS) et sert, en interne sur la boucle
// locale, EXACTEMENT le même frontend et la même API JSON que `kanjo studio`. Une seule base
// de code UI, deux façades. Ce binaire est le SEUL artefact compilé avec cgo ; le binaire
// `kanjo` (CLI + TUI + studio serveur) reste pur Go, CGO_ENABLED=0.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	webview "github.com/webview/webview_go"

	"github.com/cyprienbrisset/kanjo/cmd/kanjo/studio"
)

func main() {
	// Serveur interne sur 127.0.0.1:<port libre> avec jeton de session (§17.3).
	token := studio.NewToken()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "kanjo-studio : écoute locale impossible : %v\n", err)
		os.Exit(1)
	}
	srv := &http.Server{Handler: studio.NewHandler(token)}
	go func() { _ = srv.Serve(ln) }()

	url := fmt.Sprintf("http://%s/?token=%s", ln.Addr().String(), token)

	// Fenêtre native.
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Kanjō Studio")
	w.SetSize(1240, 820, webview.HintNone)
	w.Navigate(url)
	w.Run()
}
