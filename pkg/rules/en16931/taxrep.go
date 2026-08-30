package en16931

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles sur le représentant fiscal du vendeur (BG-11) : si un représentant fiscal est présent,
// il doit porter un nom, une adresse postale avec pays, et un identifiant de TVA.

func init() {
	rules.Register(taxRepRule("BR-18", "BT-62", func(p *model.Party) bool { return p.Name != "" },
		"Le nom du représentant fiscal est obligatoire."))
	rules.Register(taxRepRule("BR-19", "BG-12", func(p *model.Party) bool { return !p.Address.Empty() },
		"L'adresse postale du représentant fiscal est obligatoire."))
	rules.Register(taxRepRule("BR-20", "BT-69", func(p *model.Party) bool { return p.Address.CountryCode != "" },
		"Le code pays du représentant fiscal est obligatoire."))
	rules.Register(taxRepRule("BR-56", "BT-63", func(p *model.Party) bool { return p.VATID != "" },
		"L'identifiant de TVA du représentant fiscal est obligatoire."))
}

// taxRepRule émet une anomalie si un représentant fiscal est présent mais ne satisfait pas le prédicat.
func taxRepRule(id, term string, ok func(*model.Party) bool, msgFR string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": msgFR},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.TaxRep == nil || ok(d.TaxRep) {
				return nil
			}
			return []rules.Finding{{RuleID: id, Severity: rules.SeverityError, Term: term,
				Message: msgFR, Path: "taxRepresentative"}}
		},
	}
}
