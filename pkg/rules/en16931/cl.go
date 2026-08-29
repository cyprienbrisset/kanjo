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
	rules.Register(brCL17())
}

// brCL01 : le code devise (BT-5) doit être un code ISO 4217 (contrôle syntaxique : 3 lettres
// majuscules). La liste complète sera injectée depuis les listes de codes générées (L2).
func brCL01() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-01", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-5"},
		Message: map[string]string{"fr": "Le code devise doit être un code ISO 4217 valide."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			c := d.CurrencyCode
			if c == "" || !isISO4217Shape(c) {
				return []rules.Finding{{
					RuleID: "BR-CL-01", Severity: rules.SeverityError, Term: "BT-5",
					Message: fmt.Sprintf("Code devise inconnu « %s ».", c),
					Actual:  c, Fixable: false,
				}}
			}
			return nil
		},
	}
}

// brCL03 : le code type de facture (BT-3) doit appartenir à la liste UNTDID 1001 supportée.
func brCL03() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-03", Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{"BT-3"},
		Message: map[string]string{"fr": "Le code type de facture doit appartenir à la liste UNTDID 1001."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.TypeCode == "" || d.TypeCode.Valid() {
				return nil
			}
			return []rules.Finding{{
				RuleID: "BR-CL-03", Severity: rules.SeverityError, Term: "BT-3",
				Message: fmt.Sprintf("Code type de facture inconnu « %s ».", d.TypeCode),
				Actual:  string(d.TypeCode), Fixable: false,
			}}
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

// isISO4217Shape vérifie la forme d'un code devise : exactement 3 lettres majuscules A-Z.
func isISO4217Shape(c string) bool {
	if len(c) != 3 {
		return false
	}
	for i := 0; i < 3; i++ {
		if c[i] < 'A' || c[i] > 'Z' {
			return false
		}
	}
	return true
}
