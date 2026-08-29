// Package studio sert Kanjō Studio : un serveur HTTP local (127.0.0.1) exposant une API JSON
// et le frontend embarqué (ADR-005/006). Pur Go, CGO_ENABLED=0.
//
// Sécurité (§17.3) : liaison sur la boucle locale uniquement par défaut, jeton de session
// exigé pour toute route /api/*, aucune sortie réseau. Toute autre adresse de liaison exige
// un consentement explicite (--i-understand).
package studio

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cyprienbrisset/kanjo/gui/frontend"
	"github.com/cyprienbrisset/kanjo/internal/version"
	"github.com/cyprienbrisset/kanjo/pkg/api"
	"github.com/cyprienbrisset/kanjo/pkg/convert"
	_ "github.com/cyprienbrisset/kanjo/pkg/formats"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

// Options paramètre le serveur.
type Options struct {
	Port        int
	Bind        string
	NoBrowser   bool
	Token       string
	IUnderstand bool
}

const maxUpload = 64 << 20 // 64 Mo

// Run démarre le serveur et bloque jusqu'à son arrêt. Renvoie un code de sortie.
func Run(opts Options) int {
	bind := opts.Bind
	if bind == "" {
		bind = "127.0.0.1"
	}
	if !isLoopback(bind) && !opts.IUnderstand {
		fmt.Fprintf(os.Stderr, "studio : liaison sur %s refusée. Utilisez --i-understand pour exposer Kanjō Studio hors de la boucle locale (déconseillé).\n", bind)
		return 2
	}
	if opts.Token == "" {
		opts.Token = newToken()
	}

	addr := fmt.Sprintf("%s:%d", bind, opts.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "studio : écoute sur %s : %v\n", addr, err)
		return 1
	}
	url := fmt.Sprintf("http://%s/?token=%s", ln.Addr().String(), opts.Token)

	srv := &http.Server{Handler: NewHandler(opts.Token), ReadHeaderTimeout: 10 * time.Second}
	fmt.Fprintf(os.Stdout, "勘定 Kanjō Studio — %s\n", url)
	if !isLoopback(bind) {
		fmt.Fprintln(os.Stderr, "⚠ Kanjō Studio est exposé hors de la boucle locale.")
	}
	if !opts.NoBrowser {
		openBrowser(url)
	}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "studio : %v\n", err)
		return 1
	}
	return 0
}

// NewHandler construit le routeur HTTP. Exposé pour les tests (httptest).
func NewHandler(token string) http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(frontend.Assets, "dist")

	// API (jeton requis).
	mux.HandleFunc("/api/version", withToken(token, handleVersion))
	mux.HandleFunc("/api/formats", withToken(token, handleFormats))
	mux.HandleFunc("/api/validate", withToken(token, handleValidate))
	mux.HandleFunc("/api/inspect", withToken(token, handleInspect))
	mux.HandleFunc("/api/convert", withToken(token, handleConvert))

	// Frontend embarqué.
	fileServer := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			serveIndex(w, sub, token)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	return mux
}

func serveIndex(w http.ResponseWriter, sub fs.FS, token string) {
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		http.Error(w, "index introuvable", http.StatusInternalServerError)
		return
	}
	// Injecte le jeton de session dans la page servie.
	page := strings.ReplaceAll(string(data), "__TOKEN__", token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, page)
}

// withToken protège une route par le jeton de session (§17.3).
func withToken(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Kanjo-Token")
		if got == "" {
			got = r.URL.Query().Get("token")
		}
		if got != token {
			http.Error(w, "jeton de session invalide", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, version.Get())
}

func handleFormats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"read":  []string{"cii", "ubl", "facturx", "json"},
		"write": write.Targets(),
	})
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	name := r.Header.Get("X-Kanjo-Filename")
	if name == "" {
		name = "upload"
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxUpload))
	if err != nil {
		http.Error(w, "lecture du corps : "+err.Error(), http.StatusBadRequest)
		return
	}

	env := api.NewEnvelope("validate", time.Now().UTC().Format(time.RFC3339))
	res := api.Result{Input: name}
	rd, rerr := read.ReadBytes(data, name)
	if rerr != nil {
		res.Status, res.Error = api.StatusError, rerr.Error()
	} else {
		res.Format = string(rd.Format)
		rep := rules.Validate(rd.Doc)
		for _, f := range rep.Findings {
			res.Findings = append(res.Findings, api.Finding{
				RuleID: f.RuleID, Severity: f.Severity.String(), Message: f.Message,
				Term: f.Term, Expected: f.Expected, Actual: f.Actual,
			})
		}
		switch {
		case rep.HasErrors():
			res.Status = api.StatusError
		case len(rep.Findings) > 0:
			res.Status = api.StatusWarning
		default:
			res.Status = api.StatusOK
		}
	}
	env.Results = append(env.Results, res)
	env.Summary.Add(res.Status)
	writeJSON(w, env)
}

// handleInspect renvoie le pivot d'un document et ses anomalies (pour l'inspecteur G3).
func handleInspect(w http.ResponseWriter, r *http.Request) {
	data, name, ok := readBody(w, r)
	if !ok {
		return
	}
	rd, err := read.ReadBytes(data, name)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	rep := rules.Validate(rd.Doc)
	findings := make([]api.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		findings = append(findings, api.Finding{
			RuleID: f.RuleID, Severity: f.Severity.String(), Message: f.Message,
			Term: f.Term, Path: f.Path, Expected: f.Expected, Actual: f.Actual, Fixable: f.Fixable,
		})
	}
	verdict := "ok"
	switch {
	case rep.HasErrors():
		verdict = "error"
	case len(rep.Findings) > 0:
		verdict = "warning"
	}
	writeJSON(w, map[string]any{
		"format": string(rd.Format), "profile": rd.Profile,
		"verdict": verdict, "document": rd.Doc, "findings": findings,
	})
}

// handleConvert convertit le corps vers la cible ?to= et renvoie le résultat encodé en base64.
func handleConvert(w http.ResponseWriter, r *http.Request) {
	data, name, ok := readBody(w, r)
	if !ok {
		return
	}
	target := r.URL.Query().Get("to")
	if target == "" {
		target = "ubl"
	}
	res, err := convert.Convert(data, name, convert.Options{
		To:      target,
		Profile: write.Profile(orDefaultStr(r.URL.Query().Get("profile"), "en16931")),
		Syntax:  r.URL.Query().Get("syntax"),
	})
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error(), "losses": lossesOf(res)})
		return
	}
	writeJSON(w, map[string]any{
		"outputBase64": base64.StdEncoding.EncodeToString(res.Output),
		"losses":       res.Losses,
		"inputFormat":  string(res.InputFormat),
	})
}

func lossesOf(res *convert.Result) []api.Loss {
	if res == nil {
		return nil
	}
	return res.Losses
}

func readBody(w http.ResponseWriter, r *http.Request) ([]byte, string, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return nil, "", false
	}
	name := r.Header.Get("X-Kanjo-Filename")
	if name == "" {
		name = "upload"
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxUpload))
	if err != nil {
		http.Error(w, "lecture du corps : "+err.Error(), http.StatusBadRequest)
		return nil, "", false
	}
	return data, name, true
}

func orDefaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// NewToken génère un jeton de session (exposé pour le client lourd desktop).
func NewToken() string { return newToken() }

func newToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "kanjo-session"
	}
	return hex.EncodeToString(b)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	_ = exec.Command(cmd, append(args, url)...).Start()
}
