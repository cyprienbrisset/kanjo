package tabular_test

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	_ "github.com/cyprienbrisset/kanjo/pkg/write/tabular"
)

// buildSample construit une facture EN 16931 à 2 lignes, version locale inspirée de
// pkg/read/cii/roundtrip_test.go.
func buildSample() *model.Document {
	rate := model.MustParseDecimal("20")
	due, _ := model.ParseISO("2026-09-11")
	doc := model.NewDocument(model.KindInvoice)
	doc.ID = "F2026-0042"
	doc.IssueDate, _ = model.ParseISO("2026-08-12")
	doc.TypeCode = model.TypeCommercialInvoice
	doc.CurrencyCode = "EUR"

	doc.Seller = model.Party{
		Name:    "SAS Martin",
		VATID:   "FR12501234567",
		Address: model.Address{City: "Paris", CountryCode: "FR"},
	}
	doc.Buyer = model.Party{
		Name:    "Société Cliente",
		Address: model.Address{City: "Lyon", CountryCode: "FR"},
	}

	doc.Lines = []model.Line{
		{
			ID: "1", Name: "Prestation de conseil", Quantity: model.DecimalFromInt(2),
			UnitCode: model.UnitPiece, NetPrice: model.MustParseAmount("500.00", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("1000.00", "EUR"),
		},
		{
			ID: "2", Name: "Licence annuelle", Quantity: model.DecimalFromInt(1),
			UnitCode: model.UnitPiece, NetPrice: model.MustParseAmount("249.90", "EUR"),
			TaxCategory: model.TaxStandard, TaxRate: &rate,
			NetAmount: model.MustParseAmount("249.90", "EUR"),
		},
	}
	doc.TaxBreakdown = []model.TaxSubtotal{{
		Category: model.TaxStandard, Rate: rate,
		TaxableAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:     model.MustParseAmount("249.98", "EUR"),
	}}
	doc.Totals = model.Totals{
		LineExtensionAmount: model.MustParseAmount("1249.90", "EUR"),
		TaxExclusiveAmount:  model.MustParseAmount("1249.90", "EUR"),
		TaxAmount:           model.MustParseAmount("249.98", "EUR"),
		TaxInclusiveAmount:  model.MustParseAmount("1499.88", "EUR"),
		DuePayableAmount:    model.MustParseAmount("1499.88", "EUR"),
	}
	doc.DueDate = &due
	return doc
}

// colIndex renvoie l'indice d'une colonne dans l'en-tête, ou -1.
func colIndex(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

func parseCSV(t *testing.T, b []byte) [][]string {
	t.Helper()
	r := csv.NewReader(bytes.NewReader(b))
	recs, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parsing CSV: %v", err)
	}
	return recs
}

func TestWriteTwoLines(t *testing.T) {
	doc := buildSample()

	b, err := write.WriteBytes("csv", doc, write.DefaultOptions())
	if err != nil {
		t.Fatalf("écriture CSV: %v", err)
	}
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("BOM UTF-8 inattendu en tête du CSV")
	}

	recs := parseCSV(t, b)
	if len(recs) != 3 {
		t.Fatalf("nombre de lignes = %d, veut 3 (1 en-tête + 2 données)", len(recs))
	}

	header := recs[0]
	idIdx := colIndex(header, "invoiceId")
	ttcIdx := colIndex(header, "totalTTC")
	if idIdx < 0 || ttcIdx < 0 {
		t.Fatalf("colonnes invoiceId/totalTTC introuvables dans l'en-tête: %v", header)
	}

	for _, rec := range recs[1:] {
		if rec[idIdx] != "F2026-0042" {
			t.Errorf("invoiceId = %q, veut F2026-0042", rec[idIdx])
		}
		if rec[ttcIdx] != "1499.88" {
			t.Errorf("totalTTC = %q, veut 1499.88", rec[ttcIdx])
		}
	}

	// Vérifie les valeurs de ligne sur la première ligne de données.
	lineIDIdx := colIndex(header, "lineId")
	if recs[1][lineIDIdx] != "1" {
		t.Errorf("lineId ligne 1 = %q, veut 1", recs[1][lineIDIdx])
	}
	if recs[2][lineIDIdx] != "2" {
		t.Errorf("lineId ligne 2 = %q, veut 2", recs[2][lineIDIdx])
	}
}

func TestWriteNoLines(t *testing.T) {
	doc := buildSample()
	doc.Lines = nil

	b, err := write.WriteBytes("csv", doc, write.DefaultOptions())
	if err != nil {
		t.Fatalf("écriture CSV: %v", err)
	}

	recs := parseCSV(t, b)
	if len(recs) != 2 {
		t.Fatalf("nombre de lignes = %d, veut 2 (1 en-tête + 1 donnée)", len(recs))
	}

	header := recs[0]
	row := recs[1]

	if v := row[colIndex(header, "invoiceId")]; v != "F2026-0042" {
		t.Errorf("invoiceId = %q, veut F2026-0042", v)
	}
	if v := row[colIndex(header, "totalTTC")]; v != "1499.88" {
		t.Errorf("totalTTC = %q, veut 1499.88", v)
	}
	// Champs de ligne vides.
	for _, col := range []string{"lineId", "lineName", "quantity", "unitCode", "netPrice", "taxCategory", "taxRate", "lineNetAmount"} {
		if v := row[colIndex(header, col)]; v != "" {
			t.Errorf("colonne %s = %q, veut vide", col, v)
		}
	}
}
