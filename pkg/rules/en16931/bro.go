package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Famille BR-O — catégorie « hors champ de la TVA » (Not subject to VAT). Cette catégorie est
// EXCLUSIVE : une facture qui l'emploie ne porte pas d'identifiants de TVA et ne mélange pas
// d'autres catégories. Sémantique alignée sur le Schematron officiel.

func init() {
	rules.Register(brONoVATIDsRule("BR-O-02", func(d *model.Document) bool { return lineHasCategory(d, model.TaxOutsideScope) }, "BT-151"))
	rules.Register(brONoVATIDsRule("BR-O-03", func(d *model.Document) bool { return docACHasCategory(d, model.TaxOutsideScope, false) }, "BT-95"))
	rules.Register(brONoVATIDsRule("BR-O-04", func(d *model.Document) bool { return docACHasCategory(d, model.TaxOutsideScope, true) }, "BT-102"))
	rules.Register(brO11())
	rules.Register(brO12())
	rules.Register(brO13and14("BR-O-13", false, "BT-95"))
	rules.Register(brO13and14("BR-O-14", true, "BT-102"))
}

func lineHasCategory(d *model.Document, cat model.TaxCategoryCode) bool {
	for _, l := range d.Lines {
		if l.TaxCategory == cat {
			return true
		}
	}
	return false
}

func breakdownHasCategory(d *model.Document, cat model.TaxCategoryCode) bool {
	for _, ts := range d.TaxBreakdown {
		if ts.Category == cat {
			return true
		}
	}
	return false
}

// brONoVATIDsRule : si la catégorie hors champ est employée (au niveau indiqué), la facture ne
// doit porter ni identifiant TVA vendeur (BT-31) ni identifiant TVA acheteur (BT-48).
func brONoVATIDsRule(id string, triggered func(*model.Document) bool, term string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term, "BT-31", "BT-48"},
		Message: map[string]string{"fr": "Une catégorie « hors champ de la TVA » interdit les identifiants de TVA."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !triggered(d) {
				return nil
			}
			var out []rules.Finding
			if d.Seller.VATID != "" {
				out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: "BT-31",
					Message: "Identifiant TVA vendeur présent alors que la facture est hors champ de la TVA.", Path: "seller.vatId"})
			}
			if d.Buyer.VATID != "" {
				out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: "BT-48",
					Message: "Identifiant TVA acheteur présent alors que la facture est hors champ de la TVA.", Path: "buyer.vatId"})
			}
			return out
		},
	}
}

// brO11 : une ventilation hors champ interdit toute autre ventilation de TVA.
func brO11() rules.Rule {
	return rules.Rule{
		ID: "BR-O-11", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-118"},
		Message: map[string]string{"fr": "Une ventilation « hors champ » interdit toute autre ventilation de TVA."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !breakdownHasCategory(d, model.TaxOutsideScope) {
				return nil
			}
			for i, ts := range d.TaxBreakdown {
				if ts.Category != model.TaxOutsideScope {
					return []rules.Finding{{RuleID: "BR-O-11", Severity: rules.SeverityError, Term: "BT-118",
						Message: "Ventilation hors champ mélangée à d'autres catégories de TVA.",
						Path:    fmt.Sprintf("taxBreakdown[%d].category", i)}}
				}
			}
			return nil
		},
	}
}

// brO12 : une ventilation hors champ interdit toute ligne d'une autre catégorie.
func brO12() rules.Rule {
	return rules.Rule{
		ID: "BR-O-12", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-151"},
		Message: map[string]string{"fr": "Une ventilation « hors champ » interdit les lignes d'une autre catégorie."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !breakdownHasCategory(d, model.TaxOutsideScope) {
				return nil
			}
			var out []rules.Finding
			for i, l := range d.Lines {
				if l.TaxCategory != "" && l.TaxCategory != model.TaxOutsideScope {
					out = append(out, rules.Finding{RuleID: "BR-O-12", Severity: rules.SeverityError, Term: "BT-151",
						Message: "Ligne d'une catégorie autre que « hors champ » dans une facture hors champ.",
						Path:    fmt.Sprintf("lines[%d].taxCategory", i)})
				}
			}
			return out
		},
	}
}

// brO13and14 : une ventilation hors champ interdit les remises (13) ou charges (14) d'une autre catégorie.
func brO13and14(id string, wantCharge bool, term string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "Une ventilation « hors champ » interdit les remises/charges d'une autre catégorie."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if !breakdownHasCategory(d, model.TaxOutsideScope) {
				return nil
			}
			var out []rules.Finding
			for i, ac := range d.AllowanceCharges {
				if ac.IsCharge == wantCharge && ac.TaxCategory != "" && ac.TaxCategory != model.TaxOutsideScope {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: "Remise/charge d'une catégorie autre que « hors champ » dans une facture hors champ.",
						Path:    fmt.Sprintf("allowanceCharges[%d]", i)})
				}
			}
			return out
		},
	}
}
