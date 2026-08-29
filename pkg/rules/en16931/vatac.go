package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de TVA appliquées aux remises et charges de NIVEAU DOCUMENT, par catégorie (EN 16931) :
//   - « -03 » (remise BG-20) et « -04 » (charge BG-21) : identifiants requis ;
//   - « -06 » (remise) et « -07 » (charge) : contrainte de taux (BT-96 / BT-103).
// Sémantique alignée sur le Schematron officiel du CEN.

func init() {
	// Identifiant TVA/fiscal du vendeur (catégories imposables usuelles).
	for _, c := range []struct {
		cat        model.TaxCategoryCode
		label      string
		id03, id04 string
	}{
		{model.TaxStandard, "au taux normal", "BR-S-03", "BR-S-04"},
		{model.TaxZeroRated, "à taux zéro", "BR-Z-03", "BR-Z-04"},
		{model.TaxExempt, "exonérée", "BR-E-03", "BR-E-04"},
		{model.TaxExport, "à l'export", "BR-G-03", "BR-G-04"},
	} {
		rules.Register(acSellerVATRule(c.id03, c.cat, c.label, false))
		rules.Register(acSellerVATRule(c.id04, c.cat, c.label, true))
	}
	// Vendeur + acheteur (autoliquidation, intracommunautaire).
	for _, c := range []struct {
		cat        model.TaxCategoryCode
		label      string
		id03, id04 string
	}{
		{model.TaxReverseCharge, "en autoliquidation", "BR-AE-03", "BR-AE-04"},
		{model.TaxIntraCommunity, "intracommunautaire", "BR-IC-03", "BR-IC-04"},
	} {
		rules.Register(acSellerBuyerVATRule(c.id03, c.cat, c.label, false))
		rules.Register(acSellerBuyerVATRule(c.id04, c.cat, c.label, true))
	}
	// Taux positif au taux normal.
	rules.Register(acRateRule("BR-S-06", model.TaxStandard, "au taux normal", false, ratePositive))
	rules.Register(acRateRule("BR-S-07", model.TaxStandard, "au taux normal", true, ratePositive))
	// Taux nul pour les catégories à taux zéro / exonérées / etc.
	for _, c := range []struct {
		cat        model.TaxCategoryCode
		label      string
		id06, id07 string
	}{
		{model.TaxZeroRated, "à taux zéro", "BR-Z-06", "BR-Z-07"},
		{model.TaxExempt, "exonérée", "BR-E-06", "BR-E-07"},
		{model.TaxReverseCharge, "en autoliquidation", "BR-AE-06", "BR-AE-07"},
		{model.TaxExport, "à l'export", "BR-G-06", "BR-G-07"},
		{model.TaxIntraCommunity, "intracommunautaire", "BR-IC-06", "BR-IC-07"},
	} {
		rules.Register(acRateRule(c.id06, c.cat, c.label, false, rateZero))
		rules.Register(acRateRule(c.id07, c.cat, c.label, true, rateZero))
	}
}

// docACHasCategory indique la présence d'une remise (isCharge=false) ou charge (true) de niveau
// document portant la catégorie donnée.
func docACHasCategory(d *model.Document, cat model.TaxCategoryCode, isCharge bool) bool {
	for _, ac := range d.AllowanceCharges {
		if ac.IsCharge == isCharge && ac.TaxCategory == cat {
			return true
		}
	}
	return false
}

func acKind(isCharge bool) string {
	if isCharge {
		return "charge"
	}
	return "remise"
}

func acSellerVATRule(id string, cat model.TaxCategoryCode, label string, isCharge bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-31", "BT-32"},
		Message: map[string]string{"fr": fmt.Sprintf("Une %s %s exige un identifiant TVA/fiscal du vendeur.", acKind(isCharge), label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !docACHasCategory(d, cat, isCharge) {
				return nil
			}
			if d.Seller.VATID != "" || d.Seller.TaxID != "" {
				return nil
			}
			return []rules.Finding{{
				RuleID: id, Severity: rules.SeverityError, Term: "BT-31",
				Message: fmt.Sprintf("%s %s sans identifiant TVA/fiscal du vendeur.", acKind(isCharge), label),
				Path:    "seller.vatId",
			}}
		},
	}
}

func acSellerBuyerVATRule(id string, cat model.TaxCategoryCode, label string, isCharge bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-31", "BT-48"},
		Message: map[string]string{"fr": fmt.Sprintf("Une %s %s exige les identifiants TVA du vendeur et de l'acheteur.", acKind(isCharge), label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !docACHasCategory(d, cat, isCharge) {
				return nil
			}
			var out []rules.Finding
			if d.Seller.VATID == "" && d.Seller.TaxID == "" {
				out = append(out, rules.Finding{
					RuleID: id, Severity: rules.SeverityError, Term: "BT-31",
					Message: fmt.Sprintf("%s %s sans identifiant TVA/fiscal du vendeur.", acKind(isCharge), label),
					Path:    "seller.vatId",
				})
			}
			if d.Buyer.VATID == "" && d.Buyer.LegalID == "" {
				out = append(out, rules.Finding{
					RuleID: id, Severity: rules.SeverityError, Term: "BT-48",
					Message: fmt.Sprintf("%s %s sans identifiant TVA/légal de l'acheteur.", acKind(isCharge), label),
					Path:    "buyer.vatId",
				})
			}
			return out
		},
	}
}

func ratePositive(r *model.Decimal) bool { return r != nil && r.Unscaled > 0 }
func rateZero(r *model.Decimal) bool     { return r == nil || r.IsZero() }

func acRateRule(id string, cat model.TaxCategoryCode, label string, isCharge bool, ok func(*model.Decimal) bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-96", "BT-103"},
		Message: map[string]string{"fr": fmt.Sprintf("Le taux de TVA d'une %s %s est invalide.", acKind(isCharge), label)},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ac := range d.AllowanceCharges {
				if ac.IsCharge != isCharge || ac.TaxCategory != cat {
					continue
				}
				if !ok(ac.TaxRate) {
					out = append(out, rules.Finding{
						RuleID: id, Severity: rules.SeverityError, Term: "BT-96",
						Message: fmt.Sprintf("Taux de TVA invalide sur une %s %s.", acKind(isCharge), label),
						Path:    fmt.Sprintf("allowanceCharges[%d].taxRate", i),
					})
				}
			}
			return out
		},
	}
}
