package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Listes de codes volumineuses (listes officielles complètes, extraites du Schematron CEN
// UBL+CII et embarquées) : unité de mesure (Rec 20/21), sujet de note (UNCL 4451), schéma
// d'objet (UNTDID 1153).
func init() {
	rules.Register(codeListRule("BR-CL-23", "BT-130", model.IsKnownUnit, func(d *model.Document) []codeAt {
		var s []codeAt
		for i, l := range d.Lines {
			s = append(s, codeAt{string(l.UnitCode), fmt.Sprintf("lines[%d].unitCode", i)})
		}
		return s
	}))
	rules.Register(codeListRule("BR-CL-08", "BT-21", model.IsKnownNoteSubject, func(d *model.Document) []codeAt {
		var s []codeAt
		for i, n := range d.Notes {
			s = append(s, codeAt{n.SubjectCode, fmt.Sprintf("notes[%d].subjectCode", i)})
		}
		return s
	}))
	rules.Register(codeListRule("BR-CL-07", "BT-128-1", model.IsKnownObjectScheme, func(d *model.Document) []codeAt {
		var s []codeAt
		for i, l := range d.Lines {
			s = append(s, codeAt{l.ObjectScheme, fmt.Sprintf("lines[%d].objectScheme", i)})
		}
		return s
	}))
}

type codeAt struct{ code, path string }

func codeListRule(id, term string, known func(string) bool, get func(*model.Document) []codeAt) rules.Rule {
	return rules.Rule{
		ID: id, Set: setEN, Severity: rules.SeverityError, Terms: []string{term},
		Message: map[string]string{"fr": "Le code doit appartenir à sa liste officielle."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for _, c := range get(d) {
				if c.code != "" && !known(c.code) {
					out = append(out, rules.Finding{RuleID: id, Severity: rules.SeverityError, Term: term,
						Message: fmt.Sprintf("Code inconnu « %s ».", c.code), Path: c.path, Actual: c.code})
				}
			}
			return out
		},
	}
}
