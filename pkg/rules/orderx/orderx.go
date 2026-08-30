// Package orderx fournit le jeu de règles de validation des bons de commande Order-X
// (UN/CEFACT CrossIndustryOrder, Kind=order). Order-X n'étant pas une facture, les règles EN 16931
// (TVA, totaux) ne s'appliquent pas ; ce jeu vérifie la présence et la structure propres à une
// commande. Le moteur (pkg/rules) applique « orderx » aux seuls documents de type commande.
//
// Ces contrôles sont volontairement conservateurs : ils vérifient ce qu'une commande valide porte
// toujours (identifiant, date, parties, lignes, quantités), sans jamais inventer de verdict (§17.7).
package orderx

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

const setOX = rules.SetOrderX

func err(id, term, msg string) rules.Finding {
	return rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: term, Message: msg}
}

func reg(id, term, msg string, check func(*model.Document) []rules.Finding) {
	rules.Register(rules.Rule{
		ID: id, Set: setOX, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": msg},
		Check:   func(d *model.Document, _ *rules.Context) []rules.Finding { return check(d) },
	})
}

func init() {
	reg("OX-01", "BT-1", "Une commande doit porter un identifiant.", func(d *model.Document) []rules.Finding {
		if d.ID == "" {
			return []rules.Finding{err("OX-01", "BT-1", "Identifiant de commande absent.")}
		}
		return nil
	})
	reg("OX-02", "BT-2", "Une commande doit porter une date d'émission.", func(d *model.Document) []rules.Finding {
		if d.IssueDate.IsZero() {
			return []rules.Finding{err("OX-02", "BT-2", "Date d'émission de la commande absente.")}
		}
		return nil
	})
	reg("OX-03", "BT-3", "Une commande doit porter un code de type de document.", func(d *model.Document) []rules.Finding {
		if d.TypeCode == "" {
			return []rules.Finding{err("OX-03", "BT-3", "Code de type de commande absent.")}
		}
		return nil
	})
	reg("OX-04", "BT-5", "Une commande doit porter une devise.", func(d *model.Document) []rules.Finding {
		if d.CurrencyCode == "" {
			return []rules.Finding{err("OX-04", "BT-5", "Devise de la commande absente.")}
		}
		return nil
	})
	reg("OX-05", "BG-4", "Une commande doit désigner un vendeur.", func(d *model.Document) []rules.Finding {
		if d.Seller.Name == "" {
			return []rules.Finding{err("OX-05", "BG-4", "Nom du vendeur absent.")}
		}
		return nil
	})
	reg("OX-06", "BG-7", "Une commande doit désigner un acheteur.", func(d *model.Document) []rules.Finding {
		if d.Buyer.Name == "" {
			return []rules.Finding{err("OX-06", "BG-7", "Nom de l'acheteur absent.")}
		}
		return nil
	})
	reg("OX-07", "BG-25", "Une commande doit comporter au moins une ligne.", func(d *model.Document) []rules.Finding {
		if len(d.Lines) == 0 {
			return []rules.Finding{err("OX-07", "BG-25", "Aucune ligne de commande.")}
		}
		return nil
	})
	reg("OX-08", "BT-153", "Chaque ligne de commande doit désigner un article.", func(d *model.Document) []rules.Finding {
		var out []rules.Finding
		for _, l := range d.Lines {
			if l.Name == "" {
				out = append(out, err("OX-08", "BT-153", "Désignation d'article absente, ligne "+l.ID+"."))
			}
		}
		return out
	})
	reg("OX-09", "BT-129", "Chaque ligne de commande doit porter une quantité.", func(d *model.Document) []rules.Finding {
		var out []rules.Finding
		for _, l := range d.Lines {
			if !l.QuantityPresent && l.Quantity.IsZero() {
				out = append(out, err("OX-09", "BT-129", fmt.Sprintf("Quantité absente, ligne %s.", l.ID)))
			}
		}
		return out
	})
}
