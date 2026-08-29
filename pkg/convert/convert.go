// Package convert orchestre les conversions entre formats en passant par le pivot, et
// produit un rapport de perte explicite (§7.3, O8 « ne jamais mentir »).
package convert

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/api"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

// MaxLoss est une politique de tolérance à la perte.
type MaxLoss string

const (
	MaxLossNone  MaxLoss = "none"  // aucune perte tolérée
	MaxLossMinor MaxLoss = "minor" // pertes mineures (warning) tolérées, pas les pertes bloquantes
	MaxLossAny   MaxLoss = "any"   // toute perte tolérée
)

// Options paramètre une conversion.
type Options struct {
	To        string        // cible : cii|ubl|facturx|xrechnung|peppol|json|...
	Profile   write.Profile // profil cible
	Syntax    string        // "ubl"|"cii" pour xrechnung
	AllowLoss bool          // équivaut à MaxLossAny
	MaxLoss   MaxLoss       // défaut MaxLossMinor
}

// Result agrège le produit d'une conversion.
type Result struct {
	Doc         *model.Document
	Output      []byte
	Losses      []api.Loss
	InputFormat read.Format
	Profile     string
}

// ErrLossExceedsPolicy est renvoyée quand la perte dépasse la politique configurée (exit 5).
var ErrLossExceedsPolicy = errors.New("perte au-delà du seuil configuré")

// Convert lit les données, calcule les pertes et écrit vers la cible.
func Convert(data []byte, sourceName string, opts Options) (*Result, error) {
	if opts.MaxLoss == "" {
		opts.MaxLoss = MaxLossMinor
	}
	if opts.Profile == "" {
		opts.Profile = write.ProfileEN16931
	}

	rd, err := read.ReadBytes(data, sourceName)
	if err != nil {
		return nil, err
	}

	losses := computeLosses(rd, opts)
	if exceedsPolicy(losses, opts) {
		return &Result{Doc: rd.Doc, Losses: losses, InputFormat: rd.Format, Profile: rd.Profile},
			fmt.Errorf("%w: %s → %s", ErrLossExceedsPolicy, rd.Format, opts.To)
	}

	out, err := write.WriteBytes(opts.To, rd.Doc, write.Options{
		Profile: opts.Profile,
		Syntax:  opts.Syntax,
		Indent:  true,
	})
	if err != nil {
		return nil, err
	}

	return &Result{
		Doc:         rd.Doc,
		Output:      out,
		Losses:      losses,
		InputFormat: rd.Format,
		Profile:     rd.Profile,
	}, nil
}

// targetSyntax renvoie la syntaxe XML sous-jacente d'une cible ("cii", "ubl" ou "").
func targetSyntax(target, xrechnungSyntax string) string {
	switch target {
	case "cii", "facturx":
		return "cii"
	case "ubl", "peppol":
		return "ubl"
	case "xrechnung":
		if xrechnungSyntax != "" {
			return xrechnungSyntax
		}
		return "ubl"
	default:
		return ""
	}
}

// computeLosses détermine les pertes d'une conversion. Pour L1, les pertes détectées sont
// les champs non mappés (Extensions.Unmapped) dont la syntaxe source diffère de la cible :
// ils ne peuvent pas être reportés. La matrice de dégradation complète (§7.3) arrive en L2.
func computeLosses(rd *read.Result, opts Options) []api.Loss {
	var losses []api.Loss
	tgtSyntax := targetSyntax(opts.To, opts.Syntax)

	for _, uf := range rd.Doc.Extensions.Unmapped {
		if tgtSyntax != "" && uf.Syntax != "" && !strings.EqualFold(uf.Syntax, tgtSyntax) {
			losses = append(losses, api.Loss{
				Code:     "W-EXT-002",
				Severity: "warning",
				Message:  fmt.Sprintf("Extension %s non représentable en %s : %s", uf.Syntax, tgtSyntax, uf.XPath),
				Path:     uf.XPath,
			})
		}
	}
	return losses
}

// exceedsPolicy indique si l'ensemble des pertes dépasse la politique choisie.
func exceedsPolicy(losses []api.Loss, opts Options) bool {
	if opts.AllowLoss || opts.MaxLoss == MaxLossAny {
		return false
	}
	if len(losses) == 0 {
		return false
	}
	switch opts.MaxLoss {
	case MaxLossNone:
		return true // toute perte dépasse
	default: // MaxLossMinor : seules les pertes bloquantes dépassent
		for _, l := range losses {
			if l.Severity == "error" {
				return true
			}
		}
		return false
	}
}
