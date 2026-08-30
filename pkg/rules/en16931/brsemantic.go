package en16931

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles dont le Schematron officiel du CEN ne fait qu'un contrôle structural ou trivial :
//   - BR-CO-05/06/07/08 : « le code motif et le motif indiquent le même type » — la référence CEN
//     elle-même code ces règles en test="true()" (l'équivalence sémantique n'est pas vérifiable).
//     On réplique ce comportement à l'identique (conformité stricte au référentiel).
//   - BR-CO-03 : la date d'exigibilité de TVA (BT-7) et son code (BT-8) sont mutuellement exclusifs.

func init() {
	rules.Register(alwaysPassRule("BR-CO-05", "BT-98", "Code motif et motif de remise (document) doivent indiquer le même type."))
	rules.Register(alwaysPassRule("BR-CO-06", "BT-105", "Code motif et motif de charge (document) doivent indiquer le même type."))
	rules.Register(alwaysPassRule("BR-CO-07", "BT-140", "Code motif et motif de remise (ligne) doivent indiquer le même type."))
	rules.Register(alwaysPassRule("BR-CO-08", "BT-145", "Code motif et motif de charge (ligne) doivent indiquer le même type."))
	rules.Register(brCO03())
	rules.Register(brCL24MIME())
}

// alwaysPassRule réplique une règle que le Schematron CEN implémente en test="true()".
func alwaysPassRule(id, term, msgFR string) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": msgFR},
		Check:   func(_ *model.Document, _ *rules.Context) []rules.Finding { return nil },
	}
}

// brCO03 (BR-CO-03) : BT-7 (date d'exigibilité) et BT-8 (code d'exigibilité) sont exclusifs. Le
// pivot ne modélise que BT-7 ; le conflit ne peut donc pas survenir, mais la règle est exprimée.
func brCO03() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-03", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-7", "BT-8"},
		Message: map[string]string{"fr": "La date d'exigibilité de la TVA et son code sont mutuellement exclusifs."},
		Check:   func(_ *model.Document, _ *rules.Context) []rules.Finding { return nil },
	}
}

// brCL24MIME (BR-CL-24) : le type MIME d'une pièce jointe embarquée appartient à la liste restreinte.
func brCL24MIME() rules.Rule {
	allowed := map[string]bool{
		"application/pdf": true, "image/png": true, "image/jpeg": true, "text/csv": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"application/vnd.oasis.opendocument.spreadsheet":                    true,
	}
	return rules.Rule{
		ID: "BR-CL-24", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-125-1"},
		Message: map[string]string{"fr": "Le type MIME d'une pièce jointe doit appartenir à la liste autorisée."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, a := range d.Attachments {
				m := strings.TrimSpace(a.MIMEType)
				if m != "" && !allowed[m] {
					out = append(out, rules.Finding{RuleID: "BR-CL-24", Severity: rules.SeverityError, Term: "BT-125-1",
						Message: fmt.Sprintf("Type MIME non autorisé « %s ».", m), Path: fmt.Sprintf("attachments[%d].mimeType", i), Actual: m})
				}
			}
			return out
		},
	}
}
