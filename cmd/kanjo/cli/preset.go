package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/preset"
)

// presetJSON sérialise un preset en JSON indenté.
func presetJSON(p preset.Preset) []byte {
	data, _ := json.MarshalIndent(p, "", "  ")
	return append(data, '\n')
}

// parsePresetJSON désérialise un preset depuis du JSON.
func parsePresetJSON(data []byte) (preset.Preset, error) {
	var p preset.Preset
	if err := json.Unmarshal(data, &p); err != nil {
		return preset.Preset{}, fmt.Errorf("preset illisible: %w", err)
	}
	return p, nil
}

// openStore ouvre le dépôt de presets standard.
func openStore() (*preset.Store, error) {
	dir, err := preset.DefaultDir()
	if err != nil {
		return nil, err
	}
	return preset.Open(dir), nil
}

func runPreset(args []string) int {
	if len(args) == 0 {
		errf("preset : sous-commande requise (list|show|save|delete|export|import)")
		return ExitUsage
	}
	sub, rest := args[0], args[1:]
	store, err := openStore()
	if err != nil {
		errf("preset : %v", err)
		return ExitInternal
	}

	switch sub {
	case "list":
		return presetList(store)
	case "show":
		return presetShow(store, rest)
	case "save":
		return presetSave(store, rest)
	case "delete":
		return presetDelete(store, rest)
	case "export":
		return presetExport(store, rest)
	case "import":
		return presetImport(store, rest)
	default:
		errf("preset : sous-commande inconnue %q", sub)
		return ExitUsage
	}
}

func presetList(store *preset.Store) int {
	list, err := store.List()
	if err != nil {
		errf("preset list : %v", err)
		return ExitInternal
	}
	if len(list) == 0 {
		fmt.Fprintln(os.Stdout, "Aucun preset. Créez-en un avec « kanjo preset save <nom> --to <format> ».")
		return ExitOK
	}
	for _, p := range list {
		fmt.Fprintf(os.Stdout, "型 %-20s → %s", p.Name, p.To)
		if p.Profile != "" {
			fmt.Fprintf(os.Stdout, " (%s)", p.Profile)
		}
		fmt.Fprintln(os.Stdout)
	}
	return ExitOK
}

func presetShow(store *preset.Store, args []string) int {
	if len(args) != 1 {
		errf("preset show : un nom est requis")
		return ExitUsage
	}
	p, err := store.Load(args[0])
	if err != nil {
		errf("preset show : %v", err)
		return ExitUnreadable
	}
	printJSON(p)
	return ExitOK
}

func presetSave(store *preset.Store, args []string) int {
	fs := flag.NewFlagSet("preset save", flag.ContinueOnError)
	to := fs.String("to", "", "format cible (requis)")
	profile := fs.String("profile", "en16931", "profil")
	syntax := fs.String("syntax", "", "syntaxe (xrechnung)")
	maxLoss := fs.String("max-loss", "minor", "politique de perte")
	naming := fs.String("naming", "", "gabarit de nommage")
	validate := fs.Bool("validate", false, "valider la sortie")
	names, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(names) != 1 {
		errf("preset save : un nom est requis")
		return ExitUsage
	}
	if *to == "" {
		errf("preset save : --to est requis")
		return ExitUsage
	}
	p := preset.Preset{
		Name: names[0], To: *to, Profile: *profile, Syntax: *syntax,
		MaxLoss: *maxLoss, Naming: *naming, Validate: *validate,
	}
	if err := store.Save(p); err != nil {
		errf("preset save : %v", err)
		return ExitUsage
	}
	fmt.Fprintf(os.Stdout, "型 preset « %s » enregistré.\n", p.Name)
	return ExitOK
}

func presetDelete(store *preset.Store, args []string) int {
	if len(args) != 1 {
		errf("preset delete : un nom est requis")
		return ExitUsage
	}
	if err := store.Delete(args[0]); err != nil {
		errf("preset delete : %v", err)
		return ExitUnreadable
	}
	fmt.Fprintf(os.Stdout, "preset « %s » supprimé.\n", args[0])
	return ExitOK
}

func presetExport(store *preset.Store, args []string) int {
	fs := flag.NewFlagSet("preset export", flag.ContinueOnError)
	out := fs.String("out", "", "fichier de sortie (requis)")
	names, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(names) != 1 || *out == "" {
		errf("preset export : usage « kanjo preset export <nom> --out <fichier> »")
		return ExitUsage
	}
	p, err := store.Load(names[0])
	if err != nil {
		errf("preset export : %v", err)
		return ExitUnreadable
	}
	data := presetJSON(p)
	if err := fsatomic.WriteFile(*out, data, 0o644); err != nil {
		errf("preset export : %v", err)
		return ExitInternal
	}
	fmt.Fprintf(os.Stdout, "preset « %s » exporté vers %s.\n", p.Name, *out)
	return ExitOK
}

func presetImport(store *preset.Store, args []string) int {
	if len(args) != 1 {
		errf("preset import : un fichier est requis")
		return ExitUsage
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		errf("preset import : %v", err)
		return ExitUnreadable
	}
	p, err := parsePresetJSON(data)
	if err != nil {
		errf("preset import : %v", err)
		return ExitUnreadable
	}
	if err := store.Save(p); err != nil {
		errf("preset import : %v", err)
		return ExitUsage
	}
	fmt.Fprintf(os.Stdout, "型 preset « %s » importé.\n", p.Name)
	return ExitOK
}
