package en16931

import (
	"fmt"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
)

// Règles sur les périodes de facturation (BG-14 document, BG-26 ligne) et sur les références
// de facture antérieure (BG-3).

func init() {
	rules.Register(brCO19())
	rules.Register(brCO20())
	rules.Register(br29())
	rules.Register(br30())
	rules.Register(br55())
}

// brCO19 (BR-CO-19) : si une période de facturation (BG-14) est présente, elle doit porter au
// moins une date de début (BT-73) ou de fin (BT-74).
func brCO19() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-19", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-73", "BT-74"},
		Message: map[string]string{"fr": "Une période de facturation doit indiquer une date de début ou de fin."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if d.Period != nil && d.Period.Start == nil && d.Period.End == nil {
				return []rules.Finding{{
					RuleID: "BR-CO-19", Severity: rules.SeverityError, Term: "BT-73",
					Message: "Période de facturation présente mais sans date de début ni de fin.",
					Path:    "period",
				}}
			}
			return nil
		},
	}
}

// brCO20 (BR-CO-20) : si une période de ligne (BG-26) est présente, elle doit porter au moins une
// date de début (BT-134) ou de fin (BT-135).
func brCO20() rules.Rule {
	return rules.Rule{
		ID: "BR-CO-20", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-134", "BT-135"},
		Message: map[string]string{"fr": "Une période de ligne doit indiquer une date de début ou de fin."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				if l.Period != nil && l.Period.Start == nil && l.Period.End == nil {
					out = append(out, rules.Finding{
						RuleID: "BR-CO-20", Severity: rules.SeverityError, Term: "BT-134",
						Message: fmt.Sprintf("Période de la ligne %s sans date de début ni de fin.", lineLabel(l, i)),
						Path:    fmt.Sprintf("lines[%d].period", i),
					})
				}
			}
			return out
		},
	}
}

// br29 (BR-29) : si la période de facturation porte un début (BT-73) et une fin (BT-74), la fin
// ne doit pas précéder le début.
func br29() rules.Rule {
	return rules.Rule{
		ID: "BR-29", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-73", "BT-74"},
		Message: map[string]string{"fr": "La fin de période ne doit pas précéder son début."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			if p := d.Period; p != nil && p.Start != nil && p.End != nil && p.End.Before(*p.Start) {
				return []rules.Finding{{
					RuleID: "BR-29", Severity: rules.SeverityError, Term: "BT-74",
					Message: "La date de fin de période précède la date de début.",
					Path:    "period.end",
				}}
			}
			return nil
		},
	}
}

// br30 (BR-30) : idem BR-29 pour chaque période de ligne (BT-134/BT-135).
func br30() rules.Rule {
	return rules.Rule{
		ID: "BR-30", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-134", "BT-135"},
		Message: map[string]string{"fr": "La fin de période de ligne ne doit pas précéder son début."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, l := range d.Lines {
				if p := l.Period; p != nil && p.Start != nil && p.End != nil && p.End.Before(*p.Start) {
					out = append(out, rules.Finding{
						RuleID: "BR-30", Severity: rules.SeverityError, Term: "BT-135",
						Message: fmt.Sprintf("Fin de période de la ligne %s antérieure au début.", lineLabel(l, i)),
						Path:    fmt.Sprintf("lines[%d].period.end", i),
					})
				}
			}
			return out
		},
	}
}

// br55 (BR-55) : chaque référence de facture antérieure (BG-3) doit porter un identifiant (BT-25).
func br55() rules.Rule {
	return rules.Rule{
		ID: "BR-55", Set: setEN, Severity: rules.SeverityError, Terms: []string{"BT-25"},
		Message: map[string]string{"fr": "Une référence de facture antérieure doit porter un identifiant."},
		Check: func(d *model.Document, _ *rules.Context) []rules.Finding {
			var out []rules.Finding
			for i, p := range d.Precedings {
				if p.ID == "" {
					out = append(out, rules.Finding{
						RuleID: "BR-55", Severity: rules.SeverityError, Term: "BT-25",
						Message: "Référence de facture antérieure sans identifiant.",
						Path:    fmt.Sprintf("precedingInvoices[%d].id", i),
					})
				}
			}
			return out
		},
	}
}
