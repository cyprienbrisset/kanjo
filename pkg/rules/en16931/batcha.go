package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Lot complémentaire : hors-champ (taux de remise/charge), attributs d'article, adresse de
// livraison, et listes de codes supplémentaires. Sémantique alignée sur le Schematron.

func init() {
	// BR-O-06/07 : une remise/charge hors champ ne porte pas de taux de TVA.
	rules.Register(acRateRule("BR-O-06", model.TaxOutsideScope, "hors champ", false, rateZero))
	rules.Register(acRateRule("BR-O-07", model.TaxOutsideScope, "hors champ", true, rateZero))

	rules.Register(br54ItemAttributes())
	rules.Register(br57DeliverToCountry())

	// BR-CL-15 : codes pays ISO 3166 (règle jumelle de BR-CL-14).
	rules.Register(brCL14Like("BR-CL-15"))
	// BR-CL-16 : moyen de paiement UNCL 4461.
	rules.Register(br16PaymentMeans())
	// BR-CL-18 : catégories de TVA UNCL 5305 (règle jumelle de BR-CL-17).
	rules.Register(brCL17Like("BR-CL-18"))
}

// br54ItemAttributes (BR-54) : chaque attribut d'article porte un nom ET une valeur.
func br54ItemAttributes() rules.Rule {
	return rules.Rule{
		ID: "BR-54", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-160", "BT-161"},
		Message: map[string]string{"fr": "Chaque attribut d'article doit porter un nom et une valeur."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				for j, a := range l.Attributes {
					if a.Name == "" || a.Value == "" {
						out = append(out, rules.Finding{RuleID: "BR-54", Severity: rules.SeverityError, Term: "BT-160",
							Message: fmt.Sprintf("Attribut d'article incomplet (ligne %s).", lineLabel(l, i)),
							Path:    fmt.Sprintf("lines[%d].attributes[%d]", i, j)})
					}
				}
			}
			return out
		},
	}
}

// br57DeliverToCountry (BR-57) : une adresse de livraison doit porter un code pays.
func br57DeliverToCountry() rules.Rule {
	return rules.Rule{
		ID: "BR-57", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-80"},
		Message: map[string]string{"fr": "Une adresse de livraison doit porter un code pays."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.DeliverTo != nil && !d.DeliverTo.Address.Empty() && d.DeliverTo.Address.CountryCode == "" {
				return []rules.Finding{{RuleID: "BR-57", Severity: rules.SeverityError, Term: "BT-80",
					Message: "Adresse de livraison sans code pays.", Path: "deliverTo.address.countryCode"}}
			}
			return nil
		},
	}
}

// brCL14Like construit une règle de validation des codes pays (ISO 3166) sous l'identifiant donné.
func brCL14Like(id string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-40", "BT-55", "BT-80"},
		Message: map[string]string{"fr": "Les codes pays doivent appartenir à la liste ISO 3166-1."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			check := func(cc, term, path string) {
				if cc != "" && !model.IsKnownCountry(cc) {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Code pays inconnu « %s ».", cc), Path: path, Actual: cc})
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

// brCL17Like construit une règle de validation des catégories de TVA (UNCL 5305) sous l'ID donné.
func brCL17Like(id string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-118", "BT-151"},
		Message: map[string]string{"fr": "Le code catégorie de TVA doit appartenir à la liste UNCL 5305."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, ts := range d.TaxBreakdown {
				if ts.Category != "" && !ts.Category.Valid() {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: "BT-118",
						Message: fmt.Sprintf("Code catégorie de TVA inconnu « %s ».", ts.Category),
						Path:    fmt.Sprintf("taxBreakdown[%d].category", i), Actual: string(ts.Category)})
				}
			}
			for i, l := range d.Lines {
				if l.TaxCategory != "" && !l.TaxCategory.Valid() {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: "BT-151",
						Message: fmt.Sprintf("Code catégorie de TVA inconnu « %s ».", l.TaxCategory),
						Path:    fmt.Sprintf("lines[%d].taxCategory", i), Actual: string(l.TaxCategory)})
				}
			}
			return out
		},
	}
}

// br16PaymentMeans (BR-CL-16) : le moyen de paiement (BT-81) appartient à la liste UNCL 4461.
func br16PaymentMeans() rules.Rule {
	return rules.Rule{
		ID: "BR-CL-16", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-81"},
		Message: map[string]string{"fr": "Le moyen de paiement doit appartenir à la liste UNCL 4461."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			pi := d.PaymentInstructions
			if pi == nil || pi.MeansCode == "" || model.IsKnownPaymentMeans(string(pi.MeansCode)) {
				return nil
			}
			return []rules.Finding{{RuleID: "BR-CL-16", Severity: rules.SeverityError, Term: "BT-81",
				Message: fmt.Sprintf("Moyen de paiement inconnu « %s ».", pi.MeansCode), Actual: string(pi.MeansCode)}}
		},
	}
}
