package pdfa

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// ErrIncrementalUnsupported : la structure du PDF (flux xref, trailer introuvable) n'est pas prise
// en charge par la mise à jour incrémentale ; l'appelant peut se rabattre sur la réécriture.
var ErrIncrementalUnsupported = fmt.Errorf("pdfa: mise à jour incrémentale non applicable à ce PDF")

var (
	reStartxref = regexp.MustCompile(`(?s)startxref\s+(\d+)\s+%%EOF\s*$`)
	reRoot      = regexp.MustCompile(`/Root\s+(\d+)\s+\d+\s+R`)
	reInfo      = regexp.MustCompile(`/Info\s+(\d+)\s+\d+\s+R`)
	reSize      = regexp.MustCompile(`/Size\s+(\d+)`)
	reID        = regexp.MustCompile(`(?s)/ID\s*\[(.*?)\]`)
)

// embedIncremental ajoute le XML de facture à un PDF SANS réécrire ses octets : le PDF d'origine
// devient le préfixe exact du résultat, suivi des nouveaux objets et d'une section xref
// incrémentale. C'est la clé de la préservation PDF/A-3b : polices, couleurs, XMP et OutputIntent
// du PDF de base restent bit-à-bit intacts (§9.2). Ne gère que les tables xref classiques.
func embedIncremental(orig, xml []byte, name, desc string, modTime time.Time) ([]byte, error) {
	// Localiser le dernier startxref et le trailer classique.
	m := reStartxref.FindSubmatch(orig)
	if m == nil {
		return nil, ErrIncrementalUnsupported
	}
	prevStartxref := string(m[1])

	trailerStart := bytes.LastIndex(orig, []byte("trailer"))
	if trailerStart < 0 {
		return nil, ErrIncrementalUnsupported // probablement un flux xref
	}
	trailer := orig[trailerStart:]
	rootM := reRoot.FindSubmatch(trailer)
	sizeM := reSize.FindSubmatch(trailer)
	if rootM == nil || sizeM == nil {
		return nil, ErrIncrementalUnsupported
	}
	rootNum, _ := strconv.Atoi(string(rootM[1]))
	size, _ := strconv.Atoi(string(sizeM[1]))
	idInner := ""
	if idM := reID.FindSubmatch(trailer); idM != nil {
		idInner = string(bytes.TrimSpace(idM[1]))
	}
	infoRef := ""
	if infoM := reInfo.FindSubmatch(trailer); infoM != nil {
		infoRef = " /Info " + string(infoM[1]) + " 0 R"
	}

	// Extraire l'objet catalogue pour le compléter.
	catBody, err := objectBody(orig, rootNum)
	if err != nil {
		return nil, err
	}
	// Éviter un doublon de nom dans l'arbre des pièces jointes (les noms du PDF sont échappés en
	// octal : on compare via l'extraction réelle, pas par recherche d'octets).
	name = uniqueAttachmentName(orig, name)

	streamNum := size // nouvel objet flux du fichier embarqué
	filespecNum := size + 1
	newSize := size + 2

	// Objets modifiés/nouveaux à réécrire (numéro → corps complet « N 0 obj … endobj »).
	type obj struct {
		num  int
		body []byte
	}
	var objs []obj

	// 1) Catalogue mis à jour : /AF et /Names/EmbeddedFiles complétés du nouveau fichier.
	newCat, afArrayNum, afArrayBody, err := updateCatalog(orig, catBody, rootNum, filespecNum, name)
	if err != nil {
		return nil, err
	}
	// Si /AF pointe un tableau indirect, on le réécrit en y ajoutant le nouveau filespec.
	if afArrayNum > 0 {
		objs = append(objs, obj{afArrayNum, afArrayBody})
	}
	objs = append(objs, obj{rootNum, newCat})

	// 2) Flux du fichier embarqué (EmbeddedFile).
	stream := buildStreamObj(streamNum, xml, modTime)
	// 3) Spécification de fichier (association Factur-X).
	filespec := buildFilespecObj(filespecNum, streamNum, name, desc)
	objs = append(objs, obj{streamNum, stream}, obj{filespecNum, filespec})

	// Assembler la section ajoutée en suivant les offsets absolus.
	var app bytes.Buffer
	if len(orig) > 0 && orig[len(orig)-1] != '\n' {
		app.WriteByte('\n')
	}
	base := len(orig)
	offsets := map[int]int{}
	for _, o := range objs {
		// PDF/A 6.1.9-1 : le numéro d'objet et « endobj » doivent chacun être PRÉCÉDÉS d'un EOL, et
		// « obj »/« endobj » chacun SUIVIS d'un EOL. Les corps issus d'objectBody() se terminent par
		// « endobj » sans saut de ligne ; concaténés bruts, ils produiraient « endobjN 0 obj ». On
		// force donc un unique EOL avant chaque objet et après chaque « endobj ».
		if app.Len() > 0 && app.Bytes()[app.Len()-1] != '\n' {
			app.WriteByte('\n')
		}
		offsets[o.num] = base + app.Len()
		app.Write(bytes.TrimRight(o.body, "\r\n"))
		app.WriteByte('\n')
	}
	xrefOffset := base + app.Len()

	// Table xref incrémentale : regrouper les numéros contigus en sous-sections triées.
	app.WriteString(buildXref(offsets))
	fmt.Fprintf(&app, "trailer\n<< /Size %d /Root %d 0 R%s", newSize, rootNum, infoRef)
	if idInner != "" {
		fmt.Fprintf(&app, " /ID [%s]", idInner)
	}
	fmt.Fprintf(&app, " /Prev %s >>\nstartxref\n%d\n%%%%EOF\n", prevStartxref, xrefOffset)

	out := make([]byte, 0, len(orig)+app.Len())
	out = append(out, orig...)
	out = append(out, app.Bytes()...)
	return out, nil
}

// uniqueAttachmentName renvoie un nom de pièce jointe absent du PDF ; si `name` existe déjà, on
// insère un suffixe « -N » avant l'extension.
func uniqueAttachmentName(orig []byte, name string) string {
	existing := map[string]bool{}
	if atts, err := ExtractAttachments(orig); err == nil {
		for _, a := range atts {
			existing[a.Name] = true
		}
	}
	if !existing[name] {
		return name
	}
	stem, ext := name, ""
	if dot := bytes.LastIndexByte([]byte(name), '.'); dot > 0 {
		stem, ext = name[:dot], name[dot:]
	}
	for i := 1; ; i++ {
		cand := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if !existing[cand] {
			return cand
		}
	}
}

// objectBody renvoie le corps « N 0 obj … endobj » de l'objet demandé.
func objectBody(data []byte, num int) ([]byte, error) {
	marker := []byte(fmt.Sprintf("\n%d 0 obj", num))
	i := bytes.Index(data, marker)
	if i < 0 {
		return nil, fmt.Errorf("pdfa: objet %d introuvable", num)
	}
	i++ // sauter le \n initial
	j := bytes.Index(data[i:], []byte("endobj"))
	if j < 0 {
		return nil, fmt.Errorf("pdfa: fin de l'objet %d introuvable", num)
	}
	return data[i : i+j+len("endobj")], nil
}

// updateCatalog produit un nouveau corps de catalogue incluant le filespec dans /AF et
// /Names/EmbeddedFiles. Renvoie aussi, le cas échéant, le tableau /AF indirect réécrit.
func updateCatalog(orig, cat []byte, rootNum, filespecNum int, name string) (newCat []byte, afArrayNum int, afArrayBody []byte, err error) {
	body := append([]byte(nil), cat...)
	ref := fmt.Sprintf("%d 0 R", filespecNum)
	pdfName := pdfString(name)

	// --- /AF ---
	if loc := regexp.MustCompile(`/AF\s+(\d+)\s+\d+\s+R`).FindSubmatchIndex(body); loc != nil {
		afArrayNum, _ = strconv.Atoi(string(body[loc[2]:loc[3]]))
		arr, e := objectBody(orig, afArrayNum)
		if e != nil {
			return nil, 0, nil, e
		}
		afArrayBody = insertBeforeLast(arr, ']', []byte(" "+ref+" "))
	} else if loc := bytes.Index(body, []byte("/AF")); loc >= 0 {
		// /AF [ … ] inline
		if lb := bytes.IndexByte(body[loc:], '['); lb >= 0 {
			abs := loc + lb
			rb := bytes.IndexByte(body[abs:], ']')
			if rb < 0 {
				return nil, 0, nil, ErrIncrementalUnsupported
			}
			body = insertAt(body, abs+rb, []byte(" "+ref+" "))
		}
	} else {
		body = insertBeforeLast(body, '>', []byte("/AF [ "+ref+" ]\n"))
		// insertBeforeLast('>') insère avant le dernier '>' ; le catalogue se termine par « >> ».
	}

	// --- /Names/EmbeddedFiles/Names ---
	if ef := bytes.Index(body, []byte("/EmbeddedFiles")); ef >= 0 {
		if nm := bytes.Index(body[ef:], []byte("/Names")); nm >= 0 {
			abs := ef + nm
			if lb := bytes.IndexByte(body[abs:], '['); lb >= 0 {
				arrStart := abs + lb
				rb := bytes.IndexByte(body[arrStart:], ']')
				if rb < 0 {
					return nil, 0, nil, ErrIncrementalUnsupported
				}
				// Insertion TRIÉE : l'arbre de noms PDF doit rester ordonné (exigence PDF/A).
				rebuilt := insertSortedNames(body[arrStart:arrStart+rb+1], pdfName, ref)
				out := append([]byte(nil), body[:arrStart]...)
				out = append(out, rebuilt...)
				out = append(out, body[arrStart+rb+1:]...)
				body = out
			}
		}
	} else {
		names := fmt.Sprintf("/Names << /EmbeddedFiles << /Names [ %s %s ] >> >>\n", pdfName, ref)
		body = insertBeforeLast(body, '>', []byte(names))
	}
	return body, afArrayNum, afArrayBody, nil
}

// insertSortedNames reconstruit un tableau d'arbre de noms « [ (nom) réf … ] » en y insérant la
// paire (newNameLiteral, newRef) à sa place triée (par nom décodé), ordre exigé par PDF/A.
func insertSortedNames(arr []byte, newNameLiteral, newRef string) []byte {
	type pair struct {
		key string // nom décodé (clé de tri)
		lit []byte // littéral d'origine préservé tel quel
		ref []byte
	}
	var pairs []pair
	i := 0
	for i < len(arr) {
		// Chercher le prochain littéral « (…) ».
		lp := bytes.IndexByte(arr[i:], '(')
		if lp < 0 {
			break
		}
		start := i + lp
		end := scanLiteral(arr, start)
		if end < 0 {
			break
		}
		lit := arr[start:end]
		// Référence « N G R » qui suit.
		rest := arr[end:]
		rm := regexp.MustCompile(`^\s*(\d+\s+\d+\s+R)`).FindSubmatchIndex(rest)
		if rm == nil {
			break
		}
		ref := rest[rm[2]:rm[3]]
		pairs = append(pairs, pair{decodePDFLiteral(lit), lit, ref})
		i = end + rm[1]
	}
	pairs = append(pairs, pair{decodePDFLiteral([]byte(newNameLiteral)), []byte(newNameLiteral), []byte(newRef)})
	// Tri stable par clé.
	for a := 1; a < len(pairs); a++ {
		for b := a; b > 0 && pairs[b-1].key > pairs[b].key; b-- {
			pairs[b-1], pairs[b] = pairs[b], pairs[b-1]
		}
	}
	var out bytes.Buffer
	out.WriteString("[ ")
	for _, p := range pairs {
		out.Write(p.lit)
		out.WriteByte(' ')
		out.Write(p.ref)
		out.WriteByte(' ')
	}
	out.WriteByte(']')
	return out.Bytes()
}

// scanLiteral renvoie l'index juste après le « ) » fermant d'un littéral PDF débutant à start ('(').
func scanLiteral(b []byte, start int) int {
	depth := 0
	for i := start; i < len(b); i++ {
		switch b[i] {
		case '\\':
			i++ // échappement : sauter le caractère suivant
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

// decodePDFLiteral décode un littéral PDF « (…) » (échappements \ddd octal et \c) en chaîne logique.
func decodePDFLiteral(lit []byte) string {
	if len(lit) >= 2 && lit[0] == '(' {
		lit = lit[1 : len(lit)-1]
	}
	var out []byte
	for i := 0; i < len(lit); i++ {
		if lit[i] != '\\' {
			out = append(out, lit[i])
			continue
		}
		i++
		if i >= len(lit) {
			break
		}
		c := lit[i]
		if c >= '0' && c <= '7' {
			// jusqu'à 3 chiffres octaux
			val := int(c - '0')
			for k := 0; k < 2 && i+1 < len(lit) && lit[i+1] >= '0' && lit[i+1] <= '7'; k++ {
				i++
				val = val*8 + int(lit[i]-'0')
			}
			out = append(out, byte(val))
			continue
		}
		switch c {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		default:
			out = append(out, c)
		}
	}
	return string(out)
}

func buildStreamObj(num int, xml []byte, modTime time.Time) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /EmbeddedFile /Subtype /text#2Fxml /Length %d /Params << /Size %d /ModDate %s >> >>\nstream\n",
		num, len(xml), len(xml), pdfDate(modTime))
	b.Write(xml)
	b.WriteString("\nendstream\nendobj\n")
	return b.Bytes()
}

func buildFilespecObj(num, streamNum int, name, desc string) []byte {
	n := pdfString(name)
	var b bytes.Buffer
	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /Filespec /F %s /UF %s /AFRelationship /Data",
		num, n, n)
	if desc != "" {
		fmt.Fprintf(&b, " /Desc %s", pdfString(desc))
	}
	fmt.Fprintf(&b, " /EF << /F %d 0 R /UF %d 0 R >> >>\nendobj\n", streamNum, streamNum)
	return b.Bytes()
}

// buildXref écrit une table xref classique pour les objets donnés, en sous-sections contiguës triées.
func buildXref(offsets map[int]int) string {
	nums := make([]int, 0, len(offsets)+1)
	for n := range offsets {
		nums = append(nums, n)
	}
	sortInts(nums)

	var b bytes.Buffer
	b.WriteString("xref\n")
	// En-tête de la liste des objets libres (objet 0), toujours présent.
	b.WriteString("0 1\n0000000000 65535 f \n")
	i := 0
	for i < len(nums) {
		start := nums[i]
		j := i
		for j+1 < len(nums) && nums[j+1] == nums[j]+1 {
			j++
		}
		fmt.Fprintf(&b, "%d %d\n", start, j-i+1)
		for k := i; k <= j; k++ {
			fmt.Fprintf(&b, "%010d %05d n \n", offsets[nums[k]], 0)
		}
		i = j + 1
	}
	return b.String()
}

// --- petites aides ---

func pdfString(s string) string {
	// Chaîne PDF littérale, échappement des parenthèses et du backslash.
	r := make([]byte, 0, len(s)+2)
	r = append(r, '(')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', ')', '\\':
			r = append(r, '\\', s[i])
		default:
			r = append(r, s[i])
		}
	}
	return string(append(r, ')'))
}

func pdfDate(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf("(D:%04d%02d%02d%02d%02d%02d+00'00')",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func insertAt(b []byte, pos int, ins []byte) []byte {
	out := make([]byte, 0, len(b)+len(ins))
	out = append(out, b[:pos]...)
	out = append(out, ins...)
	out = append(out, b[pos:]...)
	return out
}

func insertBeforeLast(b []byte, ch byte, ins []byte) []byte {
	pos := bytes.LastIndexByte(b, ch)
	if pos < 0 {
		return append(append([]byte(nil), b...), ins...)
	}
	// Reculer devant un éventuel « > » précédent pour insérer avant « >> ».
	if ch == '>' && pos > 0 && b[pos-1] == '>' {
		pos--
	}
	return insertAt(b, pos, ins)
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
