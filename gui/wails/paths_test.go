package main

import "testing"

func TestFilterInvoicePaths(t *testing.T) {
	in := []string{"/usr/bin/Kanjo", "-flag", "/a/facture.xml", "/b/scan.PDF", "/c/note.txt"}
	got := filterInvoicePaths(in)
	if len(got) != 2 || got[0] != "/a/facture.xml" || got[1] != "/b/scan.PDF" {
		t.Fatalf("filtrage inattendu: %v", got)
	}
}
