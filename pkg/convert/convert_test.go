package convert

import (
	"bytes"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/api"
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/write"

	_ "github.com/cyprienbrisset/kanjo/pkg/formats" // enregistre lecteurs/écrivains
	wcii "github.com/cyprienbrisset/kanjo/pkg/write/cii"
)

// sampleCII construit une facture CII minimale mais cohérente, sérialisée en octets.
func sampleCII(t *testing.T) []byte {
	t.Helper()
	rate := model.MustParseDecimal("20")
	d := model.NewDocument(model.KindInvoice)
	d.ID = "F2026-0100"
	d.TypeCode = model.TypeCommercialInvoice
	d.IssueDate, _ = model.ParseISO("2026-08-12")
	d.CurrencyCode = "EUR"
	d.Seller = model.Party{Name: "SAS Martin", VATID: "FR12501234567", Address: model.Address{CountryCode: "FR"}}
	d.Buyer = model.Party{Name: "Société Cliente", Address: model.Address{CountryCode: "FR"}}
	d.Lines = []model.Line{{
		ID: "1", Name: "Conseil", Quantity: model.DecimalFromInt(1), UnitCode: model.UnitPiece,
		NetPrice: model.MustParseAmount("100.00", "EUR"), TaxCategory: model.TaxStandard,
		TaxRate: &rate, NetAmount: model.MustParseAmount("100.00", "EUR"),
	}}
	d.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("100.00", "EUR"),
		TaxAmount:     model.MustParseAmount("20.00", "EUR"),
	}}
	d.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("100.00", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("100.00", "EUR"),
		TaxAmount:           model.MustParseAmount("20.00", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("120.00", "EUR"),
		DuePayableAmount:    model.MustParseAmount("120.00", "EUR"),
	}
	b, err := wcii.Write(d, write.Options{Profile: write.ProfileEN16931})
	if err != nil {
		t.Fatalf("préparation CII: %v", err)
	}
	return b
}

func TestConvertCIIToUBL(t *testing.T) {
	in := sampleCII(t)
	res, err := Convert(in, "in.xml", Options{To: "ubl"})
	if err != nil {
		t.Fatalf("Convert CII→UBL: %v", err)
	}
	if res.InputFormat != read.FormatCII {
		t.Errorf("format d'entrée = %s, veut cii", res.InputFormat)
	}
	if f := read.Detect(res.Output); f != read.FormatUBLInvoice {
		t.Errorf("sortie détectée = %s, veut ubl", f)
	}
	if len(res.Losses) != 0 {
		t.Errorf("pertes inattendues : %+v", res.Losses)
	}
	// La sortie doit se relire et conserver l'identifiant.
	rd, err := read.ReadBytes(res.Output, "out.xml")
	if err != nil {
		t.Fatalf("relecture sortie: %v", err)
	}
	if rd.Doc.ID != "F2026-0100" {
		t.Errorf("identifiant perdu à la conversion : %q", rd.Doc.ID)
	}
}

func TestConvertToJSON(t *testing.T) {
	res, err := Convert(sampleCII(t), "in.xml", Options{To: "json"})
	if err != nil {
		t.Fatalf("Convert CII→JSON: %v", err)
	}
	if !bytes.Contains(res.Output, []byte("F2026-0100")) {
		t.Errorf("JSON ne contient pas l'identifiant :\n%s", res.Output)
	}
}

func TestConvertUnknownTarget(t *testing.T) {
	if _, err := Convert(sampleCII(t), "in.xml", Options{To: "n-existe-pas"}); err == nil {
		t.Error("cible inconnue devrait échouer")
	}
}

func TestTargetSyntax(t *testing.T) {
	cases := map[string]string{
		"cii": "cii", "facturx": "cii", "ubl": "ubl", "peppol": "ubl",
		"xrechnung": "ubl", "json": "",
	}
	for in, want := range cases {
		if got := targetSyntax(in, ""); got != want {
			t.Errorf("targetSyntax(%q) = %q, veut %q", in, got, want)
		}
	}
	if got := targetSyntax("xrechnung", "cii"); got != "cii" {
		t.Errorf("targetSyntax xrechnung/cii = %q, veut cii", got)
	}
}

func TestComputeLossesAndPolicy(t *testing.T) {
	// Un champ non mappé de syntaxe CII est perdu vers une cible UBL.
	rd := &read.Result{Doc: model.NewDocument(model.KindInvoice), Format: read.FormatCII}
	rd.Doc.Extensions.Unmapped = []model.UnmappedField{{Syntax: "cii", XPath: "/a/b"}}

	losses := computeLosses(rd, Options{To: "ubl"})
	if len(losses) != 1 || losses[0].Code != "W-EXT-002" {
		t.Fatalf("perte attendue W-EXT-002, obtenu %+v", losses)
	}
	// Même syntaxe → aucune perte.
	if l := computeLosses(rd, Options{To: "cii"}); len(l) != 0 {
		t.Errorf("aucune perte attendue vers CII, obtenu %+v", l)
	}

	warn := []api.Loss{{Severity: "warning"}}
	blk := []api.Loss{{Severity: "error"}}
	tests := []struct {
		name   string
		losses []api.Loss
		opts   Options
		want   bool
	}{
		{"aucune perte", nil, Options{MaxLoss: MaxLossNone}, false},
		{"none rejette warning", warn, Options{MaxLoss: MaxLossNone}, true},
		{"minor tolère warning", warn, Options{MaxLoss: MaxLossMinor}, false},
		{"minor rejette error", blk, Options{MaxLoss: MaxLossMinor}, true},
		{"any tolère tout", blk, Options{MaxLoss: MaxLossAny}, false},
		{"AllowLoss tolère tout", blk, Options{AllowLoss: true, MaxLoss: MaxLossNone}, false},
	}
	for _, tc := range tests {
		if got := exceedsPolicy(tc.losses, tc.opts); got != tc.want {
			t.Errorf("%s: exceedsPolicy = %v, veut %v", tc.name, got, tc.want)
		}
	}
}

func TestConvertLossExceedsPolicy(t *testing.T) {
	// Une entrée CII sans extension non mappée ne perd rien : conversion vers CII OK même en none.
	if _, err := Convert(sampleCII(t), "in.xml", Options{To: "cii", MaxLoss: MaxLossNone}); err != nil {
		t.Errorf("conversion sans perte rejetée à tort : %v", err)
	}
}
