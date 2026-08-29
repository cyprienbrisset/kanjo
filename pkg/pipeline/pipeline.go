// Package pipeline exécute le traitement par lot : découverte des fichiers, pool de workers
// borné, agrégation des résultats, reprise sur incident et sûreté aux paniques (§10, §18.2).
//
// Contrainte MUST (§20.2) : aucun chargement intégral du lot en mémoire. Chaque worker traite
// un fichier à la fois ; la consommation mémoire reste constante quel que soit le nombre de
// fichiers.
package pipeline

import (
	"fmt"
	"runtime"
	"sort"
	"sync"

	"github.com/cyprienbrisset/kanjo/pkg/api"
)

// Options paramètre l'exécution d'un lot.
type Options struct {
	Workers  int  // nombre de workers (0 = NumCPU)
	FailFast bool // arrêter au premier échec
}

// Processor traite un fichier (identifié par son chemin) et renvoie un résultat unitaire.
// Le processeur lit lui-même le fichier : le pipeline ne charge jamais tout le lot en mémoire.
type Processor func(path string) api.Result

// Report agrège les résultats d'un lot.
type Report struct {
	Results []api.Result
	Summary api.Summary
}

// Run exécute proc sur chaque fichier via un pool de workers borné. Une panique dans un
// worker est convertie en résultat en erreur (jamais un arrêt du lot), sauf FailFast.
func Run(files []string, proc Processor, opts Options) Report {
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(files) && len(files) > 0 {
		workers = len(files)
	}

	jobs := make(chan string)
	results := make(chan api.Result)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	var stopOnce sync.Once

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				res := safeProcess(proc, path)
				select {
				case results <- res:
				case <-stop:
					return
				}
				if opts.FailFast && res.Status == api.StatusError {
					stopOnce.Do(func() { close(stop) })
				}
			}
		}()
	}

	// Alimente les jobs dans une goroutine séparée pour ne pas bloquer.
	go func() {
		defer close(jobs)
		for _, f := range files {
			select {
			case jobs <- f:
			case <-stop:
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var rep Report
	for res := range results {
		rep.Results = append(rep.Results, res)
		rep.Summary.Add(res.Status)
	}
	// Ordre déterministe par chemin d'entrée.
	sort.SliceStable(rep.Results, func(i, j int) bool { return rep.Results[i].Input < rep.Results[j].Input })
	return rep
}

// safeProcess exécute proc en interceptant toute panique (§18.2 MUST : aucune panique ne doit
// interrompre un lot ; elle devient une erreur de fichier).
func safeProcess(proc Processor, path string) (res api.Result) {
	defer func() {
		if r := recover(); r != nil {
			res = api.Result{Input: path, Status: api.StatusError, Error: fmt.Sprintf("panique interne : %v", r)}
		}
	}()
	return proc(path)
}
