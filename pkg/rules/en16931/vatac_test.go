package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

// docWithAllowanceCat renvoie un document valide muni d'une remise de niveau document de la
// catégorie et du taux donnés.
func docWithAllowanceCat(cat model.TaxCategoryCode, rate string) *model.Document {
	d := validDoc()
	r := model.MustParseDecimal(rate)
	d.AllowanceCharges = []model.AllowanceCharge{
		{IsCharge: false, Amount: model.MustParseAmount("10.00", "EUR"),
			TaxCategory: cat, TaxRate: &r, ReasonCode: "95", Reason: "Remise"},
	}
	return d
}

func TestDocAllowanceSellerVAT(t *testing.T) {
	// Remise au taux normal, vendeur sans identifiant → BR-S-03.
	d := docWithAllowanceCat(model.TaxStandard, "20")
	d.Seller.VATID = ""
	d.Seller.TaxID = ""
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-S-03"] {
		t.Error("BR-S-03 attendu (remise S, vendeur sans TVA)")
	}
}

func TestDocAllowanceRatePositive(t *testing.T) {
	// Remise au taux normal avec taux 0 → BR-S-06.
	d := docWithAllowanceCat(model.TaxStandard, "0")
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-S-06"] {
		t.Error("BR-S-06 attendu (remise S à taux 0)")
	}
}

func TestDocAllowanceRateZero(t *testing.T) {
	// Remise en autoliquidation avec taux non nul → BR-AE-06.
	d := docWithAllowanceCat(model.TaxReverseCharge, "20")
	d.Buyer.VATID = "DE123456789" // satisfaire BR-AE-03
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-AE-06"] {
		t.Error("BR-AE-06 attendu (remise autoliquidation à taux non nul)")
	}
}
