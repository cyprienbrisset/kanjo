package en16931_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// TestBRCL13 : schéma de classification d'article hors UNTDID 7143 échoue ; dans la liste passe.
func TestBRCL13(t *testing.T) {
	d := validDoc()
	d.Lines[0].ClassificationID = "12345"
	d.Lines[0].ClassificationScheme = "ZZZZ" // hors liste
	if rep := rules.Validate(d, "en16931"); !hasFinding(rep, "BR-CL-13") {
		t.Error("un schéma de classification hors UNTDID 7143 doit déclencher BR-CL-13")
	}
	d.Lines[0].ClassificationScheme = "STI" // dans la liste
	if rep := rules.Validate(d, "en16931"); hasFinding(rep, "BR-CL-13") {
		t.Error("un schéma valide (STI) ne doit pas déclencher BR-CL-13")
	}
}

// TestBRCL22 : code d'exonération hors VATEX échoue ; code VATEX passe.
func TestBRCL22(t *testing.T) {
	d := validDoc()
	d.TaxBreakdown[0].ExemptionReasonCode = "PASVATEX"
	if rep := rules.Validate(d, "en16931"); !hasFinding(rep, "BR-CL-22") {
		t.Error("un code d'exonération hors VATEX doit déclencher BR-CL-22")
	}
	d.TaxBreakdown[0].ExemptionReasonCode = "vatex-eu-132-1a" // valide (casse ignorée)
	if rep := rules.Validate(d, "en16931"); hasFinding(rep, "BR-CL-22") {
		t.Error("un code VATEX valide (même en minuscules) ne doit pas déclencher BR-CL-22")
	}
}

// TestBRCL25 : schéma d'adresse électronique hors CEF EAS échoue ; « EM » (dans la liste) passe.
func TestBRCL25(t *testing.T) {
	d := validDoc()
	d.Seller.ElectronicAddr = &model.ElectronicAddress{Value: "x", Scheme: "0000"}
	if rep := rules.Validate(d, "en16931"); !hasFinding(rep, "BR-CL-25") {
		t.Error("un schéma EAS hors liste doit déclencher BR-CL-25")
	}
	d.Seller.ElectronicAddr.Scheme = "EM" // dans la liste CEN complète
	if rep := rules.Validate(d, "en16931"); hasFinding(rep, "BR-CL-25") {
		t.Error("le schéma EAS « EM » (liste CEN) ne doit pas déclencher BR-CL-25")
	}
}

// TestBRB01 : catégorie B hors facture italienne échoue ; tout-IT passe.
func TestBRB01(t *testing.T) {
	d := validDoc()
	d.TaxBreakdown[0].Category = catSplitPaymentTest()
	// vendeur/acheteur en FR (validDoc) → non italien → doit échouer.
	if rep := rules.Validate(d, "en16931"); !hasFinding(rep, "BR-B-01") {
		t.Error("catégorie B hors facture italienne doit déclencher BR-B-01")
	}
	d.Seller.Address.CountryCode = "IT"
	d.Buyer.Address.CountryCode = "IT"
	if rep := rules.Validate(d, "en16931"); hasFinding(rep, "BR-B-01") {
		t.Error("catégorie B avec parties italiennes ne doit pas déclencher BR-B-01")
	}
}

// TestBRB02 : catégorie B et S simultanées échouent.
func TestBRB02(t *testing.T) {
	d := validDoc()
	d.Seller.Address.CountryCode = "IT"
	d.Buyer.Address.CountryCode = "IT"
	// validDoc a des lignes en catégorie S ; on ajoute une ventilation B → mélange interdit.
	d.TaxBreakdown = append(d.TaxBreakdown, model.TaxSubtotal{
		Category: catSplitPaymentTest(), Rate: model.MustParseDecimal("0"), RatePresent: true,
		TaxableAmount: model.ZeroAmount("EUR"), TaxAmount: model.ZeroAmount("EUR"),
	})
	if rep := rules.Validate(d, "en16931"); !hasFinding(rep, "BR-B-02") {
		t.Error("catégories B et S simultanées doivent déclencher BR-B-02")
	}
}

func catSplitPaymentTest() model.TaxCategoryCode { return model.TaxCategoryCode("B") }
