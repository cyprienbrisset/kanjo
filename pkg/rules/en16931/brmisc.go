package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Lot complémentaire de règles EN 16931 : bénéficiaire, virement, adresses électroniques,
// identifiant d'article, et exclusivité intracommunautaire. Sémantique alignée sur le Schematron.

func init() {
	rules.Register(br17Payee())
	rules.Register(br61CreditTransferAccount())
	rules.Register(eaddrSchemeRule("BR-62", "BT-34", func(d *model.Document) *model.ElectronicAddress { return d.Seller.ElectronicAddr }))
	rules.Register(eaddrSchemeRule("BR-63", "BT-49", func(d *model.Document) *model.ElectronicAddress { return d.Buyer.ElectronicAddr }))
	rules.Register(br64ItemStandardScheme())
	rules.Register(brIC11())
	rules.Register(brIC12())
}

// br17Payee (BR-17) : si un bénéficiaire (BG-10) est présent, son nom (BT-59) est obligatoire.
func br17Payee() rules.Rule {
	return rules.Rule{
		ID: "BR-17", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-59"},
		Message: map[string]string{"fr": "Le nom du bénéficiaire est obligatoire si un bénéficiaire est indiqué."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.Payee != nil && d.Payee.Name == "" {
				return []rules.Finding{{RuleID: "BR-17", Severity: rules.SeverityError, Term: "BT-59",
					Message: "Bénéficiaire présent sans nom.", Path: "payee.name"}}
			}
			return nil
		},
	}
}

func isCreditTransfer(c model.PaymentMeansCode) bool {
	return c == model.PayCredit || c == model.PaySEPACT || c == "31"
}

// br61CreditTransferAccount (BR-61) : un moyen de paiement par virement impose un identifiant de
// compte (BT-84).
func br61CreditTransferAccount() rules.Rule {
	return rules.Rule{
		ID: "BR-61", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-84"},
		Message: map[string]string{"fr": "Un paiement par virement impose un identifiant de compte."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			pi := d.PaymentInstructions
			if pi == nil || !isCreditTransfer(pi.MeansCode) {
				return nil
			}
			for _, ct := range pi.CreditTransfers {
				if ct.IBAN != "" {
					return nil
				}
			}
			return []rules.Finding{{RuleID: "BR-61", Severity: rules.SeverityError, Term: "BT-84",
				Message: "Virement sans identifiant de compte (IBAN).", Path: "paymentInstructions.creditTransfers"}}
		},
	}
}

// eaddrSchemeRule (BR-62/63) : une adresse électronique doit porter un identifiant de schéma.
func eaddrSchemeRule(id, term string, get func(*model.Document) *model.ElectronicAddress) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "Une adresse électronique doit porter un identifiant de schéma."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if a := get(d); a != nil && a.Value != "" && a.Scheme == "" {
				return []rules.Finding{{RuleID: id, Severity: rules.SeverityError, Term: term,
					Message: "Adresse électronique sans identifiant de schéma."}}
			}
			return nil
		},
	}
}

// br64ItemStandardScheme (BR-64) : un identifiant normalisé d'article (BT-157) doit porter un schéma.
func br64ItemStandardScheme() rules.Rule {
	return rules.Rule{
		ID: "BR-64", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-157"},
		Message: map[string]string{"fr": "Un identifiant normalisé d'article doit porter un identifiant de schéma."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				if l.StandardID != "" && l.StandardScheme == "" {
					out = append(out, rules.Finding{RuleID: "BR-64", Severity: rules.SeverityError, Term: "BT-157",
						Message: fmt.Sprintf("Identifiant d'article sans schéma (ligne %s).", lineLabel(l, i)),
						Path:    fmt.Sprintf("lines[%d].standardScheme", i)})
				}
			}
			return out
		},
	}
}

// brIC11 (BR-IC-11) : une facture intracommunautaire doit indiquer une date de livraison (BT-72)
// ou une période de facturation (BG-14).
func brIC11() rules.Rule {
	return rules.Rule{
		ID: "BR-IC-11", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-72", "BG-14"},
		Message: map[string]string{"fr": "Une facture intracommunautaire doit indiquer une date de livraison ou une période."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !breakdownHasCategory(d, model.TaxIntraCommunity) {
				return nil
			}
			hasDate := d.DeliverTo != nil && d.DeliverTo.DeliveryDate != nil
			hasPeriod := d.Period != nil && (d.Period.Start != nil || d.Period.End != nil)
			if hasDate || hasPeriod {
				return nil
			}
			return []rules.Finding{{RuleID: "BR-IC-11", Severity: rules.SeverityError, Term: "BT-72",
				Message: "Facture intracommunautaire sans date de livraison ni période."}}
		},
	}
}

// brIC12 (BR-IC-12) : une facture intracommunautaire doit indiquer le pays de livraison (BT-80).
func brIC12() rules.Rule {
	return rules.Rule{
		ID: "BR-IC-12", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-80"},
		Message: map[string]string{"fr": "Une facture intracommunautaire doit indiquer le pays de livraison."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !breakdownHasCategory(d, model.TaxIntraCommunity) {
				return nil
			}
			if d.DeliverTo == nil || d.DeliverTo.Address.CountryCode == "" {
				return []rules.Finding{{RuleID: "BR-IC-12", Severity: rules.SeverityError, Term: "BT-80",
					Message: "Facture intracommunautaire sans pays de livraison.", Path: "deliverTo.address.countryCode"}}
			}
			return nil
		},
	}
}
