package cli

import (
	"flag"

	"github.com/cyprienbrisset/kanjo/cmd/kanjo/studio"
)

func runStudio(args []string) int {
	fs := flag.NewFlagSet("studio", flag.ContinueOnError)
	port := fs.Int("port", 0, "port d'écoute (0 = port libre)")
	bind := fs.String("bind", "127.0.0.1", "adresse de liaison")
	noBrowser := fs.Bool("no-browser", false, "ne pas ouvrir le navigateur")
	token := fs.String("token", "", "jeton de session (généré si absent)")
	iUnderstand := fs.Bool("i-understand", false, "autoriser une liaison hors boucle locale (déconseillé)")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	return studio.Run(studio.Options{
		Port:        *port,
		Bind:        *bind,
		NoBrowser:   *noBrowser,
		Token:       *token,
		IUnderstand: *iUnderstand,
	})
}
