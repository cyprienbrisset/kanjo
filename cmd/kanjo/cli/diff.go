package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/diff"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

// stringList collecte un flag répétable (--ignore) ; chaque occurrence peut aussi
// contenir une liste séparée par des virgules (--ignore BT-9,BT-112).
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(v string) error {
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			*s = append(*s, p)
		}
	}
	return nil
}

func runDiff(args []string) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	// --semantic est vrai par défaut : la comparaison se fait toujours sur le pivot (§G5),
	// ce qui permet de comparer deux formats différents (ex. une Factur-X vs une UBL).
	_ = fs.Bool("semantic", true, "comparaison sémantique sur le pivot (défaut : true)")
	ignoreFormatting := fs.Bool("ignore-formatting", false, "normaliser nombres et dates avant comparaison")
	format := fs.String("format", "", "format de sortie : table|json")
	var ignore stringList
	fs.Var(&ignore, "ignore", "terme(s) à exclure (répétable ou liste séparée par des virgules, ex : BT-9,BT-112)")

	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(inputs) != 2 {
		errf("diff : exactement deux fichiers d'entrée sont requis (gauche puis droite)")
		return ExitUsage
	}

	left, code := readDoc(inputs[0])
	if code != ExitOK {
		return code
	}
	right, code := readDoc(inputs[1])
	if code != ExitOK {
		return code
	}

	opts := diff.Options{
		IgnoreFormatting: *ignoreFormatting,
		Ignore:           ignoreSet(ignore),
	}
	rep := diff.Compare(left.Doc, right.Doc, opts)

	if outputFormat(*format) == "json" {
		printJSON(rep)
		return ExitOK
	}
	renderDiffTable(rep, inputs[0], inputs[1], left.Format, right.Format)
	// Les différences ne sont pas une erreur (§11.2) : code de sortie 0.
	return ExitOK
}

// readDoc lit et parse un fichier, renvoyant le résultat et ExitOK, ou un code d'erreur.
func readDoc(path string) (*read.Result, int) {
	data, err := readInput(path)
	if err != nil {
		errf("%v", err)
		return nil, ExitUnreadable
	}
	rd, err := read.ReadBytes(data, path)
	if err != nil {
		errf("%v", err)
		return nil, ExitUnreadable
	}
	return rd, ExitOK
}

func ignoreSet(list stringList) map[string]bool {
	if len(list) == 0 {
		return nil
	}
	m := make(map[string]bool, len(list))
	for _, t := range list {
		m[strings.ToUpper(t)] = true
	}
	return m
}

func renderDiffTable(rep diff.Report, leftName, rightName string, leftFmt, rightFmt read.Format) {
	w := os.Stdout
	fmt.Fprintf(w, "▸ %s (%s)  ⇔  %s (%s)\n\n", leftName, leftFmt, rightName, rightFmt)
	for _, t := range rep.Terms {
		mark := diffMark(t.Kind)
		label := t.Term
		if t.Label != "" {
			label = t.Label
		}
		fmt.Fprintf(w, "  %s %-32s %-20s %-20s\n", mark, truncate(label, 32), truncate(t.Left, 20), truncate(t.Right, 20))
	}
	if len(rep.Terms) > 0 {
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "%d perte(s) · %d divergence(s) · %d terme(s) identique(s)\n",
		rep.Losses, rep.Divergences, rep.Equal)
}

// diffMark renvoie la marque d'affichage : ⚑ pour une perte/ajout, ≠ pour une divergence,
// un espace pour l'égalité.
func diffMark(k diff.ChangeKind) string {
	switch k {
	case diff.KindLoss, diff.KindAdded:
		return "⚑"
	case diff.KindDivergence:
		return "≠"
	default:
		return " "
	}
}

// truncate raccourcit une chaîne à n runes en ajoutant une ellipse si nécessaire.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
