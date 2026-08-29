package diff

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// buildSample construit une facture EN 16931 représentative (copie locale, cf. §G5).
func buildSample() *model.Document {
	rate := model.MustParseDecimal("20")
	due, _ := model.ParseISO("2026-09-11")
	doc := model.NewDocument(model.KindInvoice)
	doc.ID = "F2026-0042"
	doc.IssueDate, _ = model.ParseISO("2026-08-12")
	doc.TypeCode = model.TypeCommercialInvoice
	doc.CurrencyCode = "EUR"
	doc.BuyerReference = "SERVICE-COMPTA"
	doc.PurchaseOrderRef = "CMD-2026-99"
	doc.Notes = []model.Note{{Content: "Merci de votre confiance."}}

	doc.Seller = model.Party{
		Name:  "SAS Martin",
		VATID: "FR12501234567",
		Address: model.Address{
			Line1: "12 rue des Comptes", PostalCode: "75001", City: "Paris", CountryCode: "FR",
		},
	}
	doc.Buyer = model.Party{
		Name: "Société Cliente",
		Address: model.Address{
			Line1: "1 avenue du SaaS", PostalCode: "69002", City: "Lyon", CountryCode: "FR",
		},
	}

	doc.Lines = []model.Line{
		{
			ID: "1", Name: "Prestation de conseil", Quantity: model.DecimalFromInt(2),
			UnitCode: model.UnitPiece, NetPrice: model.MustParseAmount("500.00", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("1000.00", "EUR"),
		},
		{
			ID: "2", Name: "Licence annuelle", Quantity: model.DecimalFromInt(1),
			UnitCode: model.UnitPiece, NetPrice: model.MustParseAmount("249.90", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("249.90", "EUR"),
		},
	}
	doc.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:     model.MustParseAmount("249.98", "EUR"),
	}}
	doc.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:           model.MustParseAmount("249.98", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("1499.88", "EUR"),
		DuePayableAmount:    model.MustParseAmount("1499.88", "EUR"),
	}
	doc.DueDate = &due
	return doc
}

func TestCompareIdentical(t *testing.T) {
	rep := Compare(buildSample(), buildSample(), Options{})
	if rep.Losses != 0 {
		t.Errorf("documents identiques : %d perte(s), veut 0", rep.Losses)
	}
	if rep.Divergences != 0 {
		t.Errorf("documents identiques : %d divergence(s), veut 0", rep.Divergences)
	}
	if rep.Equal == 0 {
		t.Errorf("documents identiques : aucun terme égal compté, attendu > 0")
	}
	// Sans HideEqual, chaque terme égal doit figurer dans le rapport.
	for _, td := range rep.Terms {
		if td.Kind == KindLoss || td.Kind == KindDivergence || td.Kind == KindAdded {
			t.Errorf("terme inattendu %s de type %s sur documents identiques", td.Term, td.Kind)
		}
	}
}

func TestCompareLoss(t *testing.T) {
	left := buildSample()
	right := buildSample()
	// Retire la référence d'échéance à droite → une perte sur BT-9.
	right.DueDate = nil

	rep := Compare(left, right, Options{})
	if rep.Losses != 1 {
		t.Errorf("champ retiré à droite : %d perte(s), veut 1", rep.Losses)
	}
	if rep.Divergences != 0 {
		t.Errorf("champ retiré à droite : %d divergence(s), veut 0", rep.Divergences)
	}
	found := false
	for _, td := range rep.Terms {
		if td.Term == "BT-9" {
			found = true
			if td.Kind != KindLoss {
				t.Errorf("BT-9 : kind %s, veut %s", td.Kind, KindLoss)
			}
			if td.Left == "" || td.Right != "" {
				t.Errorf("BT-9 : left=%q right=%q, attendu left non vide et right vide", td.Left, td.Right)
			}
		}
	}
	if !found {
		t.Errorf("BT-9 absent du rapport alors qu'une perte est attendue")
	}
}

func TestCompareDivergence(t *testing.T) {
	left := buildSample()
	right := buildSample()
	// Modifie le total TTC à droite → une divergence sur BT-112.
	right.Totals.TaxInclusiveAmount = model.MustParseAmount("1600.00", "EUR")

	rep := Compare(left, right, Options{})
	if rep.Divergences != 1 {
		t.Errorf("total modifié : %d divergence(s), veut 1", rep.Divergences)
	}
	if rep.Losses != 0 {
		t.Errorf("total modifié : %d perte(s), veut 0", rep.Losses)
	}
	found := false
	for _, td := range rep.Terms {
		if td.Term == "BT-112" {
			found = true
			if td.Kind != KindDivergence {
				t.Errorf("BT-112 : kind %s, veut %s", td.Kind, KindDivergence)
			}
		}
	}
	if !found {
		t.Errorf("BT-112 absent du rapport alors qu'une divergence est attendue")
	}
}

func TestCompareIgnore(t *testing.T) {
	left := buildSample()
	right := buildSample()
	right.Totals.TaxInclusiveAmount = model.MustParseAmount("1600.00", "EUR")

	rep := Compare(left, right, Options{Ignore: map[string]bool{"BT-112": true}})
	if rep.Divergences != 0 {
		t.Errorf("BT-112 ignoré : %d divergence(s), veut 0", rep.Divergences)
	}
	for _, td := range rep.Terms {
		if td.Term == "BT-112" {
			t.Errorf("BT-112 figure dans le rapport alors qu'il est ignoré")
		}
	}
}

func TestCompareHideEqual(t *testing.T) {
	rep := Compare(buildSample(), buildSample(), Options{HideEqual: true})
	if len(rep.Terms) != 0 {
		t.Errorf("HideEqual sur documents identiques : %d terme(s) affiché(s), veut 0", len(rep.Terms))
	}
	if rep.Equal == 0 {
		t.Errorf("HideEqual : les termes égaux doivent rester comptés")
	}
}

func TestCompareIgnoreFormatting(t *testing.T) {
	left := buildSample()
	right := buildSample()
	// Même valeur monétaire mais échelle différente (1499.880 vs 1499.88).
	right.Totals.TaxInclusiveAmount = model.NewAmount(1499880, 3, "EUR")

	// Sans IgnoreFormatting : divergence textuelle.
	plain := Compare(left, right, Options{})
	if plain.Divergences != 1 {
		t.Errorf("sans IgnoreFormatting : %d divergence(s), veut 1", plain.Divergences)
	}
	// Avec IgnoreFormatting : égalité sémantique via Amount.Equal.
	norm := Compare(left, right, Options{IgnoreFormatting: true})
	if norm.Divergences != 0 {
		t.Errorf("avec IgnoreFormatting : %d divergence(s), veut 0", norm.Divergences)
	}
}
