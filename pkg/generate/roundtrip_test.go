package generate_test

import (
	"testing"

	_ "github.com/cyprienbrisset/kanjo/pkg/formats" // enregistre lecteurs/écrivains
	"github.com/cyprienbrisset/kanjo/pkg/generate"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/all"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

// TestScenarioRoundTripConform vérifie que chaque scénario, écrit puis relu dans chaque
// syntaxe du socle, reste conforme — cela couvre notamment la préservation du motif
// d'exonération (BT-120) pour les catégories AE/K/G/E/O (non-régression).
func TestScenarioRoundTripConform(t *testing.T) {
	scenarios := []generate.Scenario{
		generate.ScenarioSimple,
		generate.ScenarioMultiTVA,
		generate.ScenarioAvoir,
		generate.ScenarioAutoliquidation,
		generate.ScenarioIntracommunautaire,
		generate.ScenarioAcompte,
	}
	for _, sc := range scenarios {
		for _, target := range []string{"cii", "ubl"} {
			doc, _ := generate.Generate(0, generate.Options{Scenario: sc, Seed: 11})
			data, err := write.WriteBytes(target, doc, write.DefaultOptions())
			if err != nil {
				t.Fatalf("%s/%s écriture: %v", sc, target, err)
			}
			rd, err := read.ReadBytes(data, "roundtrip")
			if err != nil {
				t.Fatalf("%s/%s relecture: %v", sc, target, err)
			}
			rep := rules.Validate(rd.Doc, "en16931")
			if rep.HasErrors() {
				t.Errorf("%s/%s : non conforme après aller-retour : %+v", sc, target, rep.Findings)
			}
		}
	}
}
