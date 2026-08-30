package orderx_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

func validOrder() *model.Document {
	d := model.NewDocument(model.KindOrder)
	d.ID = "PO-1"
	d.TypeCode = model.TypeCode("220")
	d.IssueDate, _ = model.ParseISO("2026-09-14")
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{Name: "Fournisseur SARL"}
	d.Buyer = model.Party{Name: "Acheteur SA"}
	d.Lines = []model.Line{{ID: "1", Name: "Clavier", Quantity: model.DecimalFromInt(10), QuantityPresent: true}}
	return d
}

func has(rep rules.Report, id string) bool {
	for _, f := range rep.Findings {
		if f.RuleID == id {
			return true
		}
	}
	return false
}

// TestValidOrderConforme : une commande complète ne déclenche aucune règle Order-X.
func TestValidOrderConforme(t *testing.T) {
	rep := rules.Validate(validOrder())
	if len(rep.Findings) != 0 {
		t.Errorf("commande valide : anomalies inattendues %+v", rep.Findings)
	}
	// Le jeu appliqué doit être « orderx », pas les règles facture.
	for _, s := range rep.RuleSets {
		if s != rules.SetOrderX {
			t.Errorf("jeu inattendu appliqué à une commande : %s", s)
		}
	}
}

// TestOrderRulesFail : chaque champ manquant déclenche sa règle.
func TestOrderRulesFail(t *testing.T) {
	cases := []struct {
		id     string
		mutate func(*model.Document)
	}{
		{"OX-01", func(d *model.Document) { d.ID = "" }},
		{"OX-02", func(d *model.Document) { d.IssueDate = model.Date{} }},
		{"OX-03", func(d *model.Document) { d.TypeCode = "" }},
		{"OX-04", func(d *model.Document) { d.CurrencyCode = "" }},
		{"OX-05", func(d *model.Document) { d.Seller.Name = "" }},
		{"OX-06", func(d *model.Document) { d.Buyer.Name = "" }},
		{"OX-07", func(d *model.Document) { d.Lines = nil }},
		{"OX-08", func(d *model.Document) { d.Lines[0].Name = "" }},
		{"OX-09", func(d *model.Document) {
			d.Lines[0].Quantity = model.MustParseDecimal("0")
			d.Lines[0].QuantityPresent = false
		}},
	}
	for _, c := range cases {
		d := validOrder()
		c.mutate(d)
		if rep := rules.Validate(d); !has(rep, c.id) {
			t.Errorf("%s aurait dû se déclencher", c.id)
		}
	}
}

// TestInvoiceRulesNotAppliedToOrder : les règles facture ne s'appliquent pas à une commande.
func TestInvoiceRulesNotAppliedToOrder(t *testing.T) {
	d := validOrder() // pas de ventilation TVA ni totaux : casserait les BR-CO en facture
	rep := rules.Validate(d)
	for _, f := range rep.Findings {
		if len(f.RuleID) >= 2 && f.RuleID[:2] == "BR" {
			t.Errorf("règle facture %s appliquée à tort à une commande", f.RuleID)
		}
	}
}
