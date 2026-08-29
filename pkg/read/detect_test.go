package read

import "testing"

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Format
	}{
		{"CII", `<?xml version="1.0"?><rsm:CrossIndustryInvoice xmlns:rsm="urn:...">`, FormatCII},
		{"ZUGFeRD 1.0", `<?xml version="1.0"?><rsm:CrossIndustryDocument xmlns:rsm="urn:...">`, FormatZUGFeRD1},
		{"UBL Invoice", `<?xml version="1.0"?><Invoice xmlns="urn:oasis:names:...:Invoice-2">`, FormatUBLInvoice},
		{"UBL CreditNote", `<?xml version="1.0"?><CreditNote xmlns="urn:oasis:names:...:CreditNote-2">`, FormatUBLCreditNote},
		{"PDF", "%PDF-1.7\n...", FormatFacturX},
		{"Kanjo JSON", `{"schemaVersion":"github.com/cyprienbrisset/kanjo/1"}`, FormatKanjoJSON},
		{"JSON quelconque", `{"foo":"bar"}`, FormatUnknown},
		{"inconnu", `bonjour`, FormatUnknown},
		{"vide", ``, FormatUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Detect([]byte(c.in)); got != c.want {
				t.Errorf("Detect(%q) = %s, veut %s", c.name, got, c.want)
			}
		})
	}
}

func TestDetectIgnoresBOMAndWhitespace(t *testing.T) {
	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte("\n\t  <rsm:CrossIndustryInvoice/>")...)
	if got := Detect(withBOM); got != FormatCII {
		t.Errorf("BOM + blancs : Detect = %s, veut cii", got)
	}
}

func TestReadBytesUnknownFormat(t *testing.T) {
	if _, err := ReadBytes([]byte("ni xml ni json"), "x.dat"); err == nil {
		t.Error("un format non reconnu devrait échouer")
	}
}
