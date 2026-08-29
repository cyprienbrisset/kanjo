package library

import (
	"path/filepath"
	"testing"
	"time"
)

func openTmp(t *testing.T) *Library {
	t.Helper()
	l, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}

func TestIndexSearchForget(t *testing.T) {
	l := openTmp(t)
	recs := []Record{
		{ID: "F2026-0001", IssueDate: "2026-08-01", SellerName: "SAS Martin", BuyerName: "Société Cliente",
			TotalTTC: "1200.00", Currency: "EUR", Format: "cii", Verdict: "ok", InputSha256: "h1"},
		{ID: "F2026-0002", IssueDate: "2026-09-15", SellerName: "SARL Dupont", BuyerName: "Société Cliente",
			TotalTTC: "500.00", Currency: "EUR", Format: "ubl", Verdict: "error", InputSha256: "h2"},
	}
	for _, r := range recs {
		if err := l.Index(r); err != nil {
			t.Fatal(err)
		}
	}
	if n, _ := l.Count(); n != 2 {
		t.Fatalf("count = %d, veut 2", n)
	}

	// Recherche texte.
	got, _ := l.Search(Query{Text: "Martin"})
	if len(got) != 1 || got[0].ID != "F2026-0001" {
		t.Errorf("recherche Martin = %+v", got)
	}
	// Filtre verdict.
	got, _ = l.Search(Query{Verdict: "error"})
	if len(got) != 1 || got[0].ID != "F2026-0002" {
		t.Errorf("filtre verdict = %+v", got)
	}
	// Filtre par période.
	got, _ = l.Search(Query{From: "2026-09-01", To: "2026-09-30"})
	if len(got) != 1 || got[0].ID != "F2026-0002" {
		t.Errorf("filtre période = %+v", got)
	}

	// Droit à l'effacement.
	n, err := l.Forget("Martin")
	if err != nil || n != 1 {
		t.Fatalf("Forget = %d (%v)", n, err)
	}
	if c, _ := l.Count(); c != 1 {
		t.Errorf("après Forget, count = %d, veut 1", c)
	}
}

func TestUpsertByHash(t *testing.T) {
	l := openTmp(t)
	_ = l.Index(Record{ID: "F1", InputSha256: "same", Verdict: "error"})
	_ = l.Index(Record{ID: "F1", InputSha256: "same", Verdict: "ok"}) // même empreinte → mise à jour
	if n, _ := l.Count(); n != 1 {
		t.Errorf("count = %d, veut 1 (upsert)", n)
	}
	got, _ := l.Search(Query{Text: "F1"})
	if len(got) != 1 || got[0].Verdict != "ok" {
		t.Errorf("upsert n'a pas mis à jour le verdict : %+v", got)
	}
}

func TestPurgeBefore(t *testing.T) {
	l := openTmp(t)
	_ = l.Index(Record{ID: "old", InputSha256: "o", ProcessedAt: "2020-01-01T00:00:00Z"})
	_ = l.Index(Record{ID: "new", InputSha256: "n", ProcessedAt: time.Now().UTC().Format(time.RFC3339)})
	n, err := l.PurgeBefore(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || n != 1 {
		t.Fatalf("PurgeBefore = %d (%v)", n, err)
	}
	if c, _ := l.Count(); c != 1 {
		t.Errorf("après purge, count = %d, veut 1", c)
	}
}
