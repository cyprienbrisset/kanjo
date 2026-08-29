package en16931

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles de présence obligatoire d'EN 16931 (BR-*). Aucune donnée n'est inventée : ces règles
// se contentent de constater l'absence d'un terme obligatoire (Fixable=false).

func init() {
	rules.Register(presence("BR-02", "BT-1", "Le numéro de facture est obligatoire.",
		func(d *model.Document) bool { return d.ID != "" }))
	rules.Register(presence("BR-03", "BT-2", "La date d'émission est obligatoire.",
		func(d *model.Document) bool { return !d.IssueDate.IsZero() }))
	rules.Register(presence("BR-04", "BT-3", "Le code type de facture est obligatoire.",
		func(d *model.Document) bool { return d.TypeCode != "" }))
	rules.Register(presence("BR-05", "BT-5", "Le code devise est obligatoire.",
		func(d *model.Document) bool { return d.CurrencyCode != "" }))
	rules.Register(presence("BR-06", "BT-27", "Le nom du vendeur est obligatoire.",
		func(d *model.Document) bool { return d.Seller.Name != "" }))
	rules.Register(presence("BR-07", "BT-44", "Le nom de l'acheteur est obligatoire.",
		func(d *model.Document) bool { return d.Buyer.Name != "" }))
	rules.Register(presence("BR-09", "BT-40", "Le code pays du vendeur est obligatoire.",
		func(d *model.Document) bool { return d.Seller.Address.CountryCode != "" }))
	rules.Register(presence("BR-11", "BT-55", "Le code pays de l'acheteur est obligatoire.",
		func(d *model.Document) bool { return d.Buyer.Address.CountryCode != "" }))
	rules.Register(presence("BR-16", "BG-25", "Une facture doit comporter au moins une ligne.",
		func(d *model.Document) bool { return len(d.Lines) > 0 }))
}

// presence construit une règle de présence : si le prédicat est faux, elle émet une anomalie.
func presence(id, term, msgFR string, ok func(*model.Document) bool) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError,
		Terms:   []string{term},
		Message: map[string]string{"fr": msgFR},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if ok(d) {
				return nil
			}
			return []rules.Finding{{
				RuleID: id, Severity: rules.SeverityError, Message: msgFR, Term: term, Fixable: false,
			}}
		},
	}
}
