package kanjo_test

import (
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/rules"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/kanjo"
)

func has(rep rules.Report, id string) bool {
	for _, f := range rep.Findings {
		if f.RuleID == id {
			return true
		}
	}
	return false
}

func baseDoc() *model.Document {
	d := model.NewDocument(model.KindInvoice)
	d.ID = "F1"
	d.CurrencyCode = "EUR"
	d.IssueDate, _ = model.ParseISO("2026-08-12")
	return d
}

func TestDueBeforeIssueWarns(t *testing.T) {
	d := baseDoc()
	before, _ := model.ParseISO("2026-08-01")
	d.DueDate = &before
	rep := rules.Validate(d, "kanjo")
	if !has(rep, "KANJO-DATE-01") {
		t.Errorf("KANJO-DATE-01 attendu. Findings: %+v", rep.Findings)
	}
}

func TestDueAfterIssueOK(t *testing.T) {
	d := baseDoc()
	after, _ := model.ParseISO("2026-09-11")
	d.DueDate = &after
	rep := rules.Validate(d, "kanjo")
	if has(rep, "KANJO-DATE-01") {
		t.Errorf("KANJO-DATE-01 ne devrait pas se déclencher : %+v", rep.Findings)
	}
}

func TestIBANValidation(t *testing.T) {
	// IBAN français valide.
	d := baseDoc()
	d.PaymentInstructions = &model.PaymentInstructions{
		CreditTransfers: []model.CreditTransfer{{IBAN: "FR7630006000011234567890189"}},
	}
	if has(rules.Validate(d, "kanjo"), "KANJO-IBAN-01") {
		t.Error("un IBAN valide ne devrait pas déclencher KANJO-IBAN-01")
	}

	// IBAN invalide (checksum cassé).
	d.PaymentInstructions.CreditTransfers[0].IBAN = "FR7630006000011234567890188"
	if !has(rules.Validate(d, "kanjo"), "KANJO-IBAN-01") {
		t.Error("un IBAN invalide devrait déclencher KANJO-IBAN-01")
	}
}
