package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de liste de codes réintroduites avec les listes CEN complètes (Schematron officiel) :
// leur absence antérieure venait de listes incomplètes qui produisaient des faux positifs. Les
// listes embarquées (pkg/model/codelists_data) couvrent exactement le référentiel CEN.
func init() {
	// BR-CL-13 : le schéma de classification d'article (BT-158) ∈ UNTDID 7143.
	rules.Register(codeListRule("BR-CL-13", "BT-158", model.IsKnownClassScheme, func(d *model.Document) []codeAt {
		var s []codeAt
		for i, l := range d.Lines {
			s = append(s, codeAt{l.ClassificationScheme, fmt.Sprintf("lines[%d].classificationScheme", i)})
		}
		return s
	}))

	// BR-CL-22 : le code d'exonération de TVA (BT-121) ∈ liste CEF VATEX.
	rules.Register(codeListRule("BR-CL-22", "BT-121", model.IsKnownVATEX, func(d *model.Document) []codeAt {
		var s []codeAt
		for i, ts := range d.TaxBreakdown {
			s = append(s, codeAt{ts.ExemptionReasonCode, fmt.Sprintf("taxBreakdown[%d].exemptionReasonCode", i)})
		}
		return s
	}))

	// BR-CL-25 : le schéma de l'identifiant d'adresse électronique (BT-34/BT-49) ∈ liste CEF EAS.
	rules.Register(codeListRule("BR-CL-25", "BT-49", model.IsKnownEASFull, func(d *model.Document) []codeAt {
		var s []codeAt
		if d.Seller.ElectronicAddr != nil {
			s = append(s, codeAt{d.Seller.ElectronicAddr.Scheme, "seller.electronicAddress.scheme"})
		}
		if d.Buyer.ElectronicAddr != nil {
			s = append(s, codeAt{d.Buyer.ElectronicAddr.Scheme, "buyer.electronicAddress.scheme"})
		}
		return s
	}))
}
