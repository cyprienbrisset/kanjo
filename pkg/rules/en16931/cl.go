package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de liste de codes (BR-CL-*) : le code doit appartenir à sa liste officielle.

func init() {
	rules.Register(brCL01())
	rules.Register(brCL03())
	rules.Register(brCL04())
	rules.Register(brCL14())
	rules.Register(brCL17())
}

// brCL01 : le code type de facture (BT-3) doit appartenir à la liste UNTDID 1001.
func brCL01() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-01", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-3"},
		Message: map[string]string{"fr": "Le code type de facture doit appartenir à la liste UNTDID 1001."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.TypeCode == "" || d.TypeCode.Valid() {
				return nil
			}
			return []rules.Finding{{
				RuleID: "BR-CL-01", Severity: rules.SeverityError, Term: "BT-3",
				Message: fmt.Sprintf("Code type de facture inconnu « %s ».", d.TypeCode),
				Actual:  string(d.TypeCode), Fixable: false,
			}}
		},
	}
}

// brCL03 : les devises des montants (currencyID) doivent être des codes ISO 4217. Dans le pivot,
// tous les montants portent la devise du document ; on valide donc celle-ci.
func brCL03() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-03", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-5"},
		Message: map[string]string{"fr": "La devise des montants doit être un code ISO 4217."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			c := d.CurrencyCode
			if c == "" || model.IsKnownCurrency(c) {
				return nil
			}
			return []rules.Finding{{
				RuleID: "BR-CL-03", Severity: rules.SeverityError, Term: "BT-5",
				Message: fmt.Sprintf("Devise des montants inconnue « %s ».", c), Actual: c,
			}}
		},
	}
}

// brCL04 : le code devise de la facture (BT-5) doit être un code ISO 4217 alpha-3.
func brCL04() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-04", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-5"},
		Message: map[string]string{"fr": "Le code devise de la facture doit être un code ISO 4217."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			c := d.CurrencyCode
			if c == "" || model.IsKnownCurrency(c) {
				return nil
			}
			return []rules.Finding{{
				RuleID: "BR-CL-04", Severity: rules.SeverityError, Term: "BT-5",
				Message: fmt.Sprintf("Code devise inconnu « %s ».", c), Actual: c,
			}}
		},
	}
}

// brCL14 : les codes pays (vendeur BT-40, acheteur BT-55, livraison BT-80) doivent appartenir à
// la liste ISO 3166-1.
func brCL14() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-14", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-40", "BT-55", "BT-80"},
		Message: map[string]string{"fr": "Les codes pays doivent appartenir à la liste ISO 3166-1."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			check := func(cc, term, path string) {
				if cc != "" && !model.IsKnownCountry(cc) {
					out = append(out, rules.Finding{
						RuleID: "BR-CL-14", Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Code pays inconnu « %s ».", cc), Path: path, Actual: cc,
					})
				}
			}
			check(d.Seller.Address.CountryCode, "BT-40", "seller.address.countryCode")
			check(d.Buyer.Address.CountryCode, "BT-55", "buyer.address.countryCode")
			if d.DeliverTo != nil {
				check(d.DeliverTo.Address.CountryCode, "BT-80", "deliverTo.address.countryCode")
			}
			return out
		},
	}
}

// brCL17 : chaque code catégorie de TVA (BT-118/151) doit appartenir à la liste UNTDID 5305.
func brCL17() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-17", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-118", "BT-151"},
		Message: map[string]string{"fr": "Le code catégorie de TVA doit appartenir à la liste UNTDID 5305."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category != "" && !ts.Category.Valid() {
					out = append(out, rules.Finding{
						RuleID: "BR-CL-17", Severity: rules.SeverityError, Term: "BT-118",
						Message: fmt.Sprintf("Code catégorie de TVA inconnu « %s » (ventilation #%d).", ts.Category, i+1),
						Path:    fmt.Sprintf("taxBreakdown[%d].category", i), Actual: string(ts.Category),
					})
				}
			}
			for i, l := range d.Lines {
				if l.TaxCategory != "" && !l.TaxCategory.Valid() {
					out = append(out, rules.Finding{
						RuleID: "BR-CL-17", Severity: rules.SeverityError, Term: "BT-151",
						Message: fmt.Sprintf("Code catégorie de TVA inconnu « %s » (ligne %s).", l.TaxCategory, l.ID),
						Path:    fmt.Sprintf("lines[%d].taxCategory", i), Actual: string(l.TaxCategory),
					})
				}
			}
			return out
		},
	}
}
