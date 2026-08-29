package generate_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/generate"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
)

func TestScenariosProduceConformInvoices(t *testing.T) {
	scenarios := []generate.Scenario{
		generate.ScenarioSimple,
		generate.ScenarioMultiTVA,
		generate.ScenarioAvoir,
		generate.ScenarioAutoliquidation,
		generate.ScenarioIntracommunautaire,
		generate.ScenarioAcompte,
	}
	for _, sc := range scenarios {
		for i := 0; i < 5; i++ {
			doc, err := generate.Generate(i, generate.Options{Scenario: sc, Seed: 7})
			if err != nil {
				t.Fatalf("%s #%d: %v", sc, i, err)
			}
			rep := rules.Validate(doc, "en16931")
			if rep.HasErrors() {
				t.Errorf("%s #%d produit un document non conforme : %+v", sc, i, rep.Findings)
			}
		}
	}
}

func TestDeterministicBySeed(t *testing.T) {
	a, _ := generate.Generate(3, generate.Options{Scenario: generate.ScenarioSimple, Seed: 42})
	b, _ := generate.Generate(3, generate.Options{Scenario: generate.ScenarioSimple, Seed: 42})
	if a.Seller.Name != b.Seller.Name || !a.Totals.TaxInclusiveAmount.Equal(b.Totals.TaxInclusiveAmount) {
		t.Error("la génération n'est pas déterministe pour un même seed")
	}
}

func TestInvalidProducesNonConform(t *testing.T) {
	doc, _ := generate.Generate(1, generate.Options{Scenario: generate.ScenarioSimple, Seed: 1, Invalid: true})
	rep := rules.Validate(doc, "en16931")
	if !rep.HasErrors() {
		t.Error("--invalid devrait produire un document non conforme")
	}
}
