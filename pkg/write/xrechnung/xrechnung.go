// Package xrechnung écrit une facture au format XRechnung 3.0, le CIUS allemand d'EN 16931.
// XRechnung existe en deux syntaxes (UBL et CII) ; on réutilise les writers UBL et CII
// existants en forçant le CustomizationID normatif propre à XRechnung.
package xrechnung

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	"github.com/cyprienbrisset/kanjo/pkg/write/cii"
	"github.com/cyprienbrisset/kanjo/pkg/write/ubl"
)

func init() { write.Register("xrechnung", Write) }

// customizationXRechnung30 est l'identifiant de spécification (CustomizationID) de
// XRechnung 3.0. Il est identique pour les syntaxes UBL et CII.
const customizationXRechnung30 = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_3.0"

// Write sérialise le document pivot en XRechnung 3.0. La syntaxe est choisie par
// opts.Syntax ("cii" → CII, toute autre valeur → UBL par défaut).
func Write(doc *model.Document, opts write.Options) ([]byte, error) {
	opts.CustomizationID = customizationXRechnung30
	if opts.Syntax == "cii" {
		return cii.Write(doc, opts)
	}
	return ubl.Write(doc, opts)
}
