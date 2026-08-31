package pdfa

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// createBasePDF génère un PDF d'une page via pdfcpu, pour servir de fond aux tests d'embed.
func createBasePDF(t *testing.T) []byte {
	t.Helper()
	const j = `{"pages":{"1":{"content":{"text":[{"value":"Facture F2026-0042","font":{"name":"Helvetica","size":12},"position":[100,700]}]}}}}`
	var buf bytes.Buffer
	if err := api.Create(nil, strings.NewReader(j), &buf, config()); err != nil {
		t.Fatalf("création du PDF de base: %v", err)
	}
	return buf.Bytes()
}

func TestEmbedThenExtractRoundTrip(t *testing.T) {
	base := createBasePDF(t)
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?><rsm:CrossIndustryInvoice><X>ok</X></rsm:CrossIndustryInvoice>`)

	res, err := EmbedXML(base, xml, "factur-x.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if res.PDFAChecked {
		t.Error("PDFAChecked doit rester false tant que veraPDF n'a pas validé (§17.7)")
	}

	got, name, warn, err := ExtractInvoiceXML(res.PDF)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if name != "factur-x.xml" {
		t.Errorf("nom = %q, veut factur-x.xml", name)
	}
	if warn != "" {
		t.Errorf("avertissement inattendu : %s", warn)
	}
	if !bytes.Equal(got, xml) {
		t.Errorf("XML extrait diffère de l'original.\noriginal: %s\nextrait:  %s", xml, got)
	}
}

// TestEmbedEstablishesFacturXAssociation vérifie, par relecture du PDF produit, que l'association
// PDF/A-3 est structurellement présente : /AFRelationship /Data et un tableau /AF au catalogue.
func TestEmbedEstablishesFacturXAssociation(t *testing.T) {
	base := createBasePDF(t)
	xml := []byte(`<?xml version="1.0"?><rsm:CrossIndustryInvoice/>`)

	res, err := EmbedXML(base, xml, "factur-x.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}

	ctx, err := api.ReadContext(bytes.NewReader(res.PDF), config())
	if err != nil {
		t.Fatalf("relecture du PDF produit: %v", err)
	}
	cat, err := ctx.XRefTable.Catalog()
	if err != nil {
		t.Fatalf("catalogue: %v", err)
	}
	// /AF présent et non vide.
	afObj, err := ctx.XRefTable.Dereference(cat["AF"])
	if err != nil {
		t.Fatalf("déréférencement de /AF: %v", err)
	}
	af, ok := afObj.(types.Array)
	if !ok || len(af) == 0 {
		t.Fatalf("tableau /AF absent ou vide au catalogue : %T", afObj)
	}
	// La spécification de fichier associée doit porter /Type /Filespec et /AFRelationship /Data.
	fs, err := ctx.XRefTable.DereferenceDict(af[0])
	if err != nil {
		t.Fatalf("déréférencement de la spécification de fichier: %v", err)
	}
	if rel := fs.NameEntry("AFRelationship"); rel == nil || *rel != "Data" {
		t.Errorf("/AFRelationship = %v, veut Data", rel)
	}
	if typ := fs.NameEntry("Type"); typ == nil || *typ != "Filespec" {
		t.Errorf("/Type de la spécification de fichier incorrect : %v", typ)
	}
}

// TestEmbedIncrementalObjectEOLBoundaries prouve, sans veraPDF (job CI dédié), que la section
// incrémentale ajoutée respecte la règle PDF/A 6.1.9-1 : dans les octets appended, chaque en-tête
// « N G obj » est précédé d'un EOL et suivi d'un EOL, et chaque « endobj » est précédé d'un EOL et
// suivi d'un EOL. Régression du FAIL 6.1.9-1 mesuré par veraPDF sur l'embed incrémental.
func TestEmbedIncrementalObjectEOLBoundaries(t *testing.T) {
	base, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus", "pdfa", "facturx_en16931.pdf"))
	if err != nil {
		t.Skipf("PDF de référence indisponible: %v", err)
	}
	res, err := EmbedXML(base, []byte(`<?xml version="1.0"?><rsm:CrossIndustryInvoice/>`), "factur-x.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(res.PDF) <= len(base) || string(res.PDF[:len(base)]) != string(base) {
		t.Fatal("les octets du PDF de base doivent être le préfixe exact du résultat")
	}
	app := res.PDF[len(base):] // section strictement ajoutée par la mise à jour incrémentale

	isEOL := func(b byte) bool { return b == '\n' || b == '\r' }

	// En-têtes d'objet « N G obj » : précédés et suivis d'un EOL.
	reObj := regexp.MustCompile(`(?m)(\d+)\s+(\d+)\s+obj`)
	locs := reObj.FindAllIndex(app, -1)
	if len(locs) == 0 {
		t.Fatal("aucun objet trouvé dans la section incrémentale")
	}
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		// Précédé d'un EOL : soit c'est le tout premier octet (précédé par le \n final du PDF de base),
		// soit l'octet juste avant est un EOL.
		if start > 0 && !isEOL(app[start-1]) {
			t.Errorf("6.1.9-1 : en-tête d'objet à l'offset %d non précédé d'un EOL (octet précédent %q)", start, app[start-1])
		}
		if start == 0 && !isEOL(base[len(base)-1]) {
			t.Errorf("6.1.9-1 : premier objet ajouté non précédé d'un EOL (dernier octet du PDF de base %q)", base[len(base)-1])
		}
		// « obj » suivi d'un EOL.
		if end < len(app) && !isEOL(app[end]) {
			t.Errorf("6.1.9-1 : mot-clé « obj » à l'offset %d non suivi d'un EOL (octet suivant %q)", end, app[end])
		}
	}

	// « endobj » : précédé et suivi d'un EOL.
	for _, idx := range allIndices(app, []byte("endobj")) {
		if idx > 0 && !isEOL(app[idx-1]) {
			t.Errorf("6.1.9-1 : « endobj » à l'offset %d non précédé d'un EOL (octet précédent %q)", idx, app[idx-1])
		}
		after := idx + len("endobj")
		if after < len(app) && !isEOL(app[after]) {
			t.Errorf("6.1.9-1 : « endobj » à l'offset %d non suivi d'un EOL (octet suivant %q)", idx, app[after])
		}
	}
}

// allIndices renvoie tous les offsets d'apparition de sub dans b (sans chevauchement).
func allIndices(b, sub []byte) []int {
	var out []int
	for off := 0; ; {
		i := bytes.Index(b[off:], sub)
		if i < 0 {
			break
		}
		out = append(out, off+i)
		off += i + len(sub)
	}
	return out
}

// TestEmbedIncrementalPreservesBytes vérifie que l'embed sur un vrai PDF/A (table xref classique)
// procède par mise à jour incrémentale : les octets du PDF de base sont le préfixe exact du
// résultat, la pièce jointe est extractible, et aucun doublon de nom n'est introduit.
func TestEmbedIncrementalPreservesBytes(t *testing.T) {
	base, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus", "pdfa", "facturx_en16931.pdf"))
	if err != nil {
		t.Skipf("PDF de référence indisponible: %v", err)
	}
	res, err := EmbedXML(base, []byte(`<?xml version="1.0"?><rsm:CrossIndustryInvoice/>`), "factur-x.xml")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(res.PDF) <= len(base) || string(res.PDF[:len(base)]) != string(base) {
		t.Fatal("les octets du PDF de base doivent être le préfixe exact du résultat")
	}
	// La sortie reste relisible et la pièce jointe extractible.
	atts, err := ExtractAttachments(res.PDF)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	var names []string
	seen := map[string]int{}
	for _, a := range atts {
		names = append(names, a.Name)
		seen[a.Name]++
	}
	for n, c := range seen {
		if c > 1 {
			t.Errorf("doublon de pièce jointe %q (%d)", n, c)
		}
	}
	// Le XML embarqué doit être retrouvable (sous factur-x.xml existant ou nom dédupliqué).
	found := false
	for _, n := range names {
		if strings.HasPrefix(n, "factur-x") {
			found = true
		}
	}
	if !found {
		t.Errorf("pièce jointe Factur-X introuvable parmi %v", names)
	}
}
