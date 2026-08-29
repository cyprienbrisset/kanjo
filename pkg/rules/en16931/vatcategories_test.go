package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
)

func TestBreakdownExistsPerCategory(t *testing.T) {
	// Une ligne à taux zéro sans ventilation Z → BR-Z-01.
	d := validDoc()
	zero := model.MustParseDecimal("0")
	d.Lines[0].TaxCategory = model.TaxZeroRated
	d.Lines[0].TaxRate = &zero
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-Z-01"] {
		t.Error("BR-Z-01 attendu (catégorie Z sans ventilation)")
	}
}

func TestSellerVATRequired(t *testing.T) {
	// Ligne au taux normal mais vendeur sans identifiant TVA/fiscal → BR-S-02.
	d := validDoc()
	d.Seller.VATID = ""
	d.Seller.TaxID = ""
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-S-02"] {
		t.Error("BR-S-02 attendu (vendeur sans identifiant TVA)")
	}
	// validDoc standard (vendeur avec TVA) ne déclenche pas BR-S-02.
	if findingsByRule(rules.Validate(validDoc(), "en16931"))["BR-S-02"] {
		t.Error("BR-S-02 ne devrait pas se déclencher quand le vendeur a une TVA")
	}
}

func TestReverseChargeRequiresBuyerVAT(t *testing.T) {
	// Autoliquidation utilisée mais acheteur sans TVA → BR-AE-02.
	d := validDoc()
	zero := model.MustParseDecimal("0")
	d.Lines[0].TaxCategory = model.TaxReverseCharge
	d.Lines[0].TaxRate = &zero
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-AE-02"] {
		t.Error("BR-AE-02 attendu (autoliquidation sans TVA acheteur)")
	}
	// Avec une TVA acheteur, BR-AE-02 ne se déclenche plus.
	d.Buyer.VATID = "DE123456789"
	if findingsByRule(rules.Validate(d, "en16931"))["BR-AE-02"] {
		t.Error("BR-AE-02 ne devrait plus se déclencher avec une TVA acheteur")
	}
}

func TestStandardNoExemptionReason(t *testing.T) {
	// Ventilation au taux normal avec motif d'exonération indu → BR-S-10.
	d := validDoc()
	d.TaxBreakdown[0].ExemptionReason = "Exonéré (indu)"
	if !findingsByRule(rules.Validate(d, "en16931"))["BR-S-10"] {
		t.Error("BR-S-10 attendu (motif d'exonération sur ventilation S)")
	}
}
