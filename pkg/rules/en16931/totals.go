package en16931

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de présence des montants totaux du document (BG-22).
//   - BR-12 : somme des montants nets de ligne (BT-106).
//   - BR-13 : total hors TVA (BT-109).
//   - BR-14 : total TVA comprise (BT-112).
//   - BR-15 : net à payer (BT-115).
// Le calcul de ces totaux est vérifié séparément par les BR-CO-10 à BR-CO-16.

func init() {
	rules.Register(presence("BR-12", "BT-106", "Le total des montants nets de ligne est obligatoire.",
		func(d *model.Document) bool { return d.Totals.LineExtensionAmount.Currency != "" }))
	rules.Register(presence("BR-13", "BT-109", "Le total hors TVA est obligatoire.",
		func(d *model.Document) bool { return d.Totals.TaxExclusiveAmount.Currency != "" }))
	rules.Register(presence("BR-14", "BT-112", "Le total TVA comprise est obligatoire.",
		func(d *model.Document) bool { return d.Totals.TaxInclusiveAmount.Currency != "" }))
	rules.Register(presence("BR-15", "BT-115", "Le net à payer est obligatoire.",
		func(d *model.Document) bool { return d.Totals.DuePayableAmount.Currency != "" }))
}
