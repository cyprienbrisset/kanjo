package crossvalidate

import "testing"

func TestParseMustang(t *testing.T) {
	cases := []struct {
		out  string
		want bool
	}{
		{`<validation><summary status="valid"/></validation>`, true},
		{`<validation><summary status="invalid"/></validation>`, false},
		{"The document is valid.", true},
		{"The document is not valid.", false},
		{"", false},
	}
	for _, c := range cases {
		if got := parseMustang(c.out); got != c.want {
			t.Errorf("parseMustang(%q) = %v, veut %v", c.out, got, c.want)
		}
	}
}

func TestParseKoSIT(t *testing.T) {
	cases := []struct {
		out  string
		want bool
	}{
		{`<rep:acceptRecommendation>accept</rep:acceptRecommendation>`, true},
		{`<rep:acceptRecommendation>reject</rep:acceptRecommendation>`, false},
		{"", false},
	}
	for _, c := range cases {
		if got := parseKoSIT(c.out); got != c.want {
			t.Errorf("parseKoSIT(%q) = %v, veut %v", c.out, got, c.want)
		}
	}
}

// TestCompare vérifie la logique d'accord/désaccord, y compris les outils non exécutés.
func TestCompare(t *testing.T) {
	verdicts := []Verdict{
		{Tool: "mustangproject", Ran: true, Compliant: true},
		{Tool: "kosit", Ran: true, Compliant: false},
		{Tool: "other", Ran: false, Detail: "non configuré"},
	}
	agree, disagree, lines := Compare(true, verdicts)
	if agree != 1 || disagree != 1 {
		t.Errorf("accords=%d désaccords=%d, veut 1/1", agree, disagree)
	}
	if len(lines) != 3 {
		t.Errorf("lignes = %d, veut 3", len(lines))
	}
}

// TestAvailableWithoutConfig : sans variables d'environnement, aucun validateur externe.
func TestAvailableWithoutConfig(t *testing.T) {
	t.Setenv("KANJO_MUSTANG_JAR", "")
	t.Setenv("KANJO_KOSIT_JAR", "")
	if got := Available(); len(got) != 0 {
		t.Errorf("Available() = %v, veut vide", got)
	}
}
