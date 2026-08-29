package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/convert"
	"github.com/cyprienbrisset/kanjo/pkg/pipeline"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func runWatch(args []string) int {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	presetName := fs.String("preset", "", "preset à appliquer (requis)")
	outDir := fs.String("out", "", "dossier de sortie (défaut : <inbox>/output)")
	doneDir := fs.String("done", "", "dossier des fichiers traités (défaut : <inbox>/done)")
	failedDir := fs.String("failed", "", "dossier des échecs (défaut : <inbox>/failed)")
	poll := fs.Duration("poll", 2*time.Second, "intervalle de scrutation")
	once := fs.Bool("once", false, "traiter l'existant puis sortir")
	recursive := fs.Bool("recursive", false, "surveiller récursivement")
	positionals, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positionals) != 1 {
		errf("watch : un dossier à surveiller est requis")
		return ExitUsage
	}
	if *presetName == "" {
		errf("watch : --preset est requis")
		return ExitUsage
	}
	inbox := positionals[0]

	store, err := openStore()
	if err != nil {
		errf("watch : %v", err)
		return ExitInternal
	}
	p, err := store.Load(*presetName)
	if err != nil {
		errf("watch : %v", err)
		return ExitUnreadable
	}
	opts := convert.Options{
		To:      p.To,
		Profile: write.Profile(orDefault(p.Profile, "en16931")),
		Syntax:  p.Syntax,
		MaxLoss: convert.MaxLoss(orDefault(p.MaxLoss, "minor")),
	}

	out := orDefault(*outDir, filepath.Join(inbox, "output"))
	done := orDefault(*doneDir, filepath.Join(inbox, "done"))
	failed := orDefault(*failedDir, filepath.Join(inbox, "failed"))
	for _, d := range []string{out, done, failed} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			errf("watch : %v", err)
			return ExitInternal
		}
	}

	fmt.Fprintf(os.Stdout, "番 surveillance de %s (preset %s) — Ctrl+C pour arrêter\n", inbox, p.Name)
	if !*once {
		fmt.Fprintln(os.Stdout, "   note : la surveillance s'arrête à la fermeture de cette commande (pas de service).")
	}

	watcher := pipeline.NewWatcher(inbox, *recursive)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(*poll)
	defer ticker.Stop()

	emptyRounds := 0
	for {
		ready, err := watcher.Ready()
		if err != nil {
			errf("watch : %v", err)
			return ExitInternal
		}
		if len(ready) == 0 {
			emptyRounds++
		} else {
			emptyRounds = 0
		}
		for _, path := range ready {
			processWatched(path, opts, out, done, failed)
		}
		// En mode --once, on sort après deux tours à vide (le temps de stabiliser puis traiter).
		if *once && emptyRounds >= 2 {
			return ExitOK
		}
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stdout, "\n番 surveillance arrêtée.")
			return ExitInterrupted
		case <-ticker.C:
		}
	}
}

func processWatched(path string, opts convert.Options, out, done, failed string) {
	data, err := os.ReadFile(path)
	if err != nil {
		moveTo(path, failed, err)
		return
	}
	cr, err := convert.Convert(data, path, opts)
	if err != nil {
		moveTo(path, failed, err)
		fmt.Fprintf(os.Stdout, "否 %s : %v\n", filepath.Base(path), err)
		return
	}
	outName := deriveName(path, opts.To)
	if err := fsatomic.WriteFile(filepath.Join(out, outName), cr.Output, 0o644); err != nil {
		moveTo(path, failed, err)
		return
	}
	if err := os.Rename(path, filepath.Join(done, filepath.Base(path))); err != nil {
		// Le fichier de sortie est écrit ; on signale seulement l'échec de rangement.
		errf("watch : rangement de %s : %v", path, err)
	}
	verdict := "適"
	if len(cr.Losses) > 0 {
		verdict = "保"
	}
	fmt.Fprintf(os.Stdout, "%s %s → %s\n", verdict, filepath.Base(path), outName)
}

// moveTo déplace un fichier en échec vers le dossier de quarantaine et écrit un .error.json.
func moveTo(path, failed string, cause error) {
	base := filepath.Base(path)
	_ = os.Rename(path, filepath.Join(failed, base))
	errInfo := map[string]string{
		"file":  base,
		"error": cause.Error(),
		"ts":    time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(errInfo, "", "  ")
	_ = fsatomic.WriteFile(filepath.Join(failed, base+".error.json"), data, 0o644)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
