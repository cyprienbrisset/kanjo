package read

import (
	"encoding/json"
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

func init() { Register(FormatKanjoJSON, readKanjoJSON) }

// readKanjoJSON désérialise un document pivot JSON Kanjō.
func readKanjoJSON(data []byte, sourceName string) (*model.Document, error) {
	var doc model.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("lecture JSON %s: %w", sourceName, err)
	}
	if doc.SchemaVersion != model.SchemaVersion {
		return nil, fmt.Errorf("lecture JSON %s: version de schéma %q inattendue (attendu %q)",
			sourceName, doc.SchemaVersion, model.SchemaVersion)
	}
	// Le JSON pivot porte quantité (BT-129) et taux de ventilation (BT-119) comme champs présents :
	// on rétablit les indicateurs de présence (non sérialisés) pour ne pas déclencher à tort BR-22/BR-48.
	for i := range doc.Lines {
		doc.Lines[i].QuantityPresent = true
	}
	for i := range doc.TaxBreakdown {
		doc.TaxBreakdown[i].RatePresent = true
	}
	doc.Provenance = model.NewProvenance(sourceName, "json", "")
	return &doc, nil
}
