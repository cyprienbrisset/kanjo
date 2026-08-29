package write

import (
	"encoding/json"
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

func init() { Register("json", writeKanjoJSON) }

// writeKanjoJSON sérialise le document pivot en JSON Kanjō (compact ou indenté).
func writeKanjoJSON(doc *model.Document, opts Options) ([]byte, error) {
	if doc.SchemaVersion == "" {
		doc.SchemaVersion = model.SchemaVersion
	}
	var (
		out []byte
		err error
	)
	if opts.Indent {
		out, err = json.MarshalIndent(doc, "", "  ")
	} else {
		out, err = json.Marshal(doc)
	}
	if err != nil {
		return nil, fmt.Errorf("écriture JSON: %w", err)
	}
	return out, nil
}
