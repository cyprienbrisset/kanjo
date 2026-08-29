package xmlsafe

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type invoiceDoc struct {
	ID     string `xml:"ID"`
	Seller string `xml:"Seller"`
}

// TestAttackCorpus vérifie que toutes les charges malveillantes du corpus sont rejetées
// et que le document sûr est accepté (exigence MUST §17.1).
func TestAttackCorpus(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "fuzz", "xxe")
	cases := []struct {
		file      string
		wantError bool
	}{
		{"xxe-file-read.xml", true},    // DOCTYPE + entité SYSTEM
		{"billion-laughs.xml", true},   // expansion d'entités
		{"xxe-external-dtd.xml", true}, // DTD externe
		{"safe-utf8.xml", false},       // document légitime
	}
	for _, c := range cases {
		data, err := os.ReadFile(filepath.Join(dir, c.file))
		if err != nil {
			t.Fatalf("lecture %s: %v", c.file, err)
		}
		err = Check(data, DefaultLimits())
		if c.wantError && err == nil {
			t.Errorf("%s: attendait un rejet, obtenu nil", c.file)
		}
		if !c.wantError && err != nil {
			t.Errorf("%s: attendait acceptation, obtenu %v", c.file, err)
		}
	}
}

func TestDoctypeRejected(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><!DOCTYPE x><root/>`)
	if err := Check(data, DefaultLimits()); !errors.Is(err, ErrDoctype) {
		t.Errorf("attendait ErrDoctype, obtenu %v", err)
	}
}

func TestDepthLimit(t *testing.T) {
	var b strings.Builder
	const n = 50
	for i := 0; i < n; i++ {
		b.WriteString("<a>")
	}
	for i := 0; i < n; i++ {
		b.WriteString("</a>")
	}
	lim := DefaultLimits()
	lim.MaxDepth = 10
	if err := Check([]byte(b.String()), lim); !errors.Is(err, ErrDepthExceeded) {
		t.Errorf("attendait ErrDepthExceeded, obtenu %v", err)
	}
}

func TestSizeLimit(t *testing.T) {
	lim := DefaultLimits()
	lim.MaxBytes = 8
	if err := Check([]byte("<root>trop long</root>"), lim); !errors.Is(err, ErrTooLarge) {
		t.Errorf("attendait ErrTooLarge, obtenu %v", err)
	}
}

func TestUnmarshalSafeDocument(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?><Invoice><ID>F1</ID><Seller>Sté &amp; Cie</Seller></Invoice>`)
	var doc invoiceDoc
	if err := Decode(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.ID != "F1" || doc.Seller != "Sté & Cie" {
		t.Errorf("désérialisation incorrecte: %+v", doc)
	}
}

func TestLatin1Charset(t *testing.T) {
	// "é" en ISO-8859-1 = 0xE9
	data := append([]byte(`<?xml version="1.0" encoding="ISO-8859-1"?><Invoice><ID>`), 0xE9)
	data = append(data, []byte(`</ID><Seller>x</Seller></Invoice>`)...)
	var doc invoiceDoc
	if err := Decode(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.ID != "é" {
		t.Errorf("conversion Latin-1 = %q, veut é", doc.ID)
	}
}

func TestUnsafeCharsetRejected(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-16"?><root/>`)
	var v struct{}
	if err := Decode(data, &v); err == nil {
		t.Error("un encodage non supporté devrait être rejeté")
	}
}
