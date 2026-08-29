package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/cyprienbrisset/kanjo/internal/version"
)

func runVersion(args []string) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	format := fs.String("format", "", "format de sortie : table|json")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	info := version.Get()
	if outputFormat(*format) == "json" {
		printJSON(info)
		return ExitOK
	}
	fmt.Fprintf(os.Stdout, "kanjo %s\n", info.Tool)
	fmt.Fprintf(os.Stdout, "  commit       %s\n", info.Commit)
	fmt.Fprintf(os.Stdout, "  build        %s\n", info.BuildDate)
	fmt.Fprintf(os.Stdout, "  jeu de règles %s\n", info.Rules)
	fmt.Fprintf(os.Stdout, "  schéma       %s\n", info.Schema)
	return ExitOK
}
