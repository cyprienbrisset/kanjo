//go:build cgo

// Commande kanjo-studio — client lourd desktop de Kanjō Studio (ADR-005, §19.1).
//
// Il embarque une fenêtre WebView native (WebKit sur macOS) et sert, en interne sur la boucle
// locale, EXACTEMENT le même frontend et la même API JSON que `kanjo studio`. Une seule base
// de code UI, deux façades. Ce binaire est le SEUL artefact compilé avec cgo ; le binaire
// `kanjo` (CLI + TUI + studio serveur) reste pur Go, CGO_ENABLED=0.
package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	webview "github.com/webview/webview_go"

	"github.com/cyprienbrisset/kanjo/cmd/kanjo/studio"
)

// pickedFile est un fichier choisi via le dialogue natif, transmis au frontend
// (contenu encodé en base64 pour traverser le pont JS↔Go).
type pickedFile struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

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

	// Pont natif : WKWebView (macOS) ne fournit pas de sélecteur de fichier pour
	// <input type=file>. On expose window.kanjoOpenFiles() au frontend, qui ouvre le
	// dialogue natif du système et renvoie le nom + le contenu (base64) des fichiers choisis.
	if err := w.Bind("kanjoOpenFiles", func() []pickedFile {
		out := make([]pickedFile, 0)
		for _, p := range nativePickFiles() {
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			out = append(out, pickedFile{Name: filepath.Base(p), Data: base64.StdEncoding.EncodeToString(b)})
		}
		return out
	}); err != nil {
		fmt.Fprintf(os.Stderr, "kanjo-studio : pont natif indisponible : %v\n", err)
	}

	w.Navigate(url)
	w.Run()
}

// nativePickFiles ouvre le sélecteur de fichiers natif du système d'exploitation et
// renvoie les chemins choisis. Renvoie nil si l'utilisateur annule ou si l'outil de
// dialogue est absent (aucune dépendance cgo supplémentaire : simple exec).
func nativePickFiles() []string {
	var out []byte
	var err error
	switch runtime.GOOS {
	case "darwin":
		const script = `set theFiles to choose file with prompt "Choisir des factures" with multiple selections allowed
set thePaths to ""
repeat with f in theFiles
	set thePaths to thePaths & POSIX path of f & linefeed
end repeat
return thePaths`
		out, err = exec.Command("osascript", "-e", script).Output()
	case "linux":
		out, err = exec.Command("zenity", "--file-selection", "--multiple",
			"--separator=\n", "--title=Choisir des factures").Output()
	case "windows":
		const ps = "Add-Type -AssemblyName System.Windows.Forms; " +
			"$d = New-Object System.Windows.Forms.OpenFileDialog; $d.Multiselect = $true; " +
			"$d.Filter = 'Factures (*.xml;*.pdf;*.json)|*.xml;*.pdf;*.json|Tous (*.*)|*.*'; " +
			"if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { $d.FileNames -join \"`n\" }"
		out, err = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Output()
	}
	if err != nil {
		return nil
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if p := strings.TrimSpace(line); p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}
