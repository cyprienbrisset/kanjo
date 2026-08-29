// Package cli implémente l'interface en ligne de commande de Kanjō (§11).
// Pour L1, l'analyse d'arguments repose sur la bibliothèque standard (flag) afin de
// conserver un binaire pur Go sans dépendance. La migration vers Cobra est prévue en L2.
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	_ "github.com/cyprienbrisset/kanjo/pkg/formats" // câblage du registre des formats (lecteurs/écrivains)
)

const usage = `kanjo — manipulation de factures électroniques (Factur-X, UBL, CII…)

Usage :
  kanjo <commande> [options]

Commandes :
  convert    Convertir des factures d'un format vers un autre
  extract    Extraire le XML d'une Factur-X
  embed      Embarquer un XML dans un PDF (Factur-X)
  inspect    Inspecter un document (synthèse, termes BT, XML)
  diff       Comparer sémantiquement deux documents (pertes, divergences)
  validate   Valider un document (EN 16931 + CIUS)
  preset     Gérer les jeux de réglages réutilisables
  watch      Surveiller un dossier et convertir les fichiers déposés
  repair     Corriger les anomalies sûres d'une facture
  anonymize  Anonymiser une facture (export RGPD-safe)
  generate   Générer des factures synthétiques (corpus de test)
  render     Produire la face lisible (HTML) d'une facture
  library    Indexer et retrouver les documents traités (bibliothèque locale)
  audit      Consulter le journal d'audit
  tui        Lancer l'interface texte (aussi : kanjo sans argument)
  studio     Lancer Kanjō Studio (interface graphique locale)
  doctor     Diagnostic de l'environnement
  version    Afficher les versions (outil, jeu de règles, schéma)
  help       Afficher cette aide

Options globales fréquentes :
  --format table|json    Format de sortie (défaut : table sur terminal, json sinon)

Exécutez « kanjo <commande> --help » pour l'aide d'une commande.
`

// Execute est le point d'entrée de la CLI. Il renvoie un code de sortie (§11.4).
func Execute(args []string) int {
	if len(args) == 0 {
		// Sans argument sur un terminal, on lance la TUI (§11.1) ; sinon, l'aide.
		if isTTY(os.Stdout) {
			return runTUI(nil)
		}
		fmt.Fprint(os.Stdout, usage)
		return ExitOK
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "tui":
		return runTUI(rest)
	case "studio":
		return runStudio(rest)
	case "convert":
		return runConvert(rest)
	case "extract":
		return runExtract(rest)
	case "embed":
		return runEmbed(rest)
	case "inspect":
		return runInspect(rest)
	case "diff":
		return runDiff(rest)
	case "validate":
		return runValidate(rest)
	case "preset":
		return runPreset(rest)
	case "watch":
		return runWatch(rest)
	case "repair":
		return runRepair(rest)
	case "anonymize":
		return runAnonymize(rest)
	case "generate":
		return runGenerate(rest)
	case "render":
		return runRender(rest)
	case "audit":
		return runAudit(rest)
	case "library":
		return runLibrary(rest)
	case "doctor":
		return runDoctor(rest)
	case "version":
		return runVersion(rest)
	case "help", "-h", "--help":
		fmt.Fprint(os.Stdout, usage)
		return ExitOK
	default:
		fmt.Fprintf(os.Stderr, "commande inconnue : %q\n\n%s", cmd, usage)
		return ExitUsage
	}
}

// parseInterspersed analyse des arguments où flags et positionnels sont entremêlés
// (`kanjo convert entree.xml --to ubl`). La bibliothèque standard s'arrête au premier
// positionnel ; cette boucle reprend le parsing après chaque positionnel rencontré.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}
	return positionals, nil
}

// printJSON écrit une valeur en JSON indenté sur la sortie standard.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "erreur d'encodage JSON : %v\n", err)
	}
}

// errf écrit un message d'erreur sur stderr (sans excuse, §12.9).
func errf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}
