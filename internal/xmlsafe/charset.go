package xmlsafe

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// safeCharsetReader n'autorise qu'un ensemble restreint et sûr de jeux de caractères,
// tous convertibles en pur Go sans dépendance externe. Toute autre déclaration d'encodage
// est refusée plutôt que traitée de façon hasardeuse.
func safeCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch normalizeCharset(charset) {
	case "utf-8", "us-ascii", "ascii", "":
		return input, nil
	case "iso-8859-1", "latin1", "windows-1252", "cp1252":
		return newLatinReader(input), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsafeCharset, charset)
	}
}

func normalizeCharset(c string) string {
	return strings.ToLower(strings.TrimSpace(c))
}

// newLatinReader convertit un flux Latin-1 / Windows-1252 en UTF-8. Les octets 0x80–0x9F
// suivent la table Windows-1252 (surensemble de Latin-1 courant dans les factures).
func newLatinReader(r io.Reader) io.Reader {
	raw, err := io.ReadAll(r)
	if err != nil {
		return &errReader{err: err}
	}
	var b bytes.Buffer
	b.Grow(len(raw) * 2)
	for _, c := range raw {
		if c < 0x80 {
			b.WriteByte(c)
			continue
		}
		if ru := cp1252High[c-0x80]; ru != 0 {
			b.WriteRune(ru)
		} else {
			b.WriteRune(rune(c)) // repli Latin-1 pour les positions non définies
		}
	}
	return &b
}

type errReader struct{ err error }

func (e *errReader) Read([]byte) (int, error) { return 0, e.err }

// cp1252High mappe les octets 0x80–0xFF de Windows-1252 vers Unicode. Un 0 signifie
// « non défini » → repli sur la valeur Latin-1 (rune(octet)).
var cp1252High = [128]rune{
	0x20AC, 0, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, // 80-87
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0, 0x017D, 0, // 88-8F
	0, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, // 90-97
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0, 0x017E, 0x0178, // 98-9F
	// 0xA0–0xFF identiques à Latin-1 : on laisse 0 pour déclencher le repli rune(octet).
}
