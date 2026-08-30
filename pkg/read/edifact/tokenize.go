package edifact

import (
	"errors"
	"strings"
)

// delimiters porte les séparateurs de service EDIFACT. Les valeurs par défaut correspondent au
// jeu UNOB/UNOA usuel ; un segment UNA en tête peut les redéfinir (ISO 9735 §4).
type delimiters struct {
	component byte // séparateur de composant (défaut « : »)
	data      byte // séparateur d'élément de données (défaut « + »)
	decimal   byte // signe décimal (défaut « . »)
	release   byte // caractère d'échappement (défaut « ? »)
	segment   byte // terminateur de segment (défaut « ' »)
}

func defaultDelimiters() delimiters {
	return delimiters{component: ':', data: '+', decimal: '.', release: '?', segment: '\''}
}

// segment est un segment EDIFACT décodé : son étiquette (LIN, MOA…) et ses éléments de données,
// chacun éclaté en composants. elements[i][j] = j-ᵉ composant du i-ᵉ élément de données.
type segment struct {
	tag      string
	elements [][]string
}

// element renvoie le i-ᵉ élément de données (0-based) ou une tranche vide s'il est absent.
func (s segment) element(i int) []string {
	if i < 0 || i >= len(s.elements) {
		return nil
	}
	return s.elements[i]
}

// comp renvoie le composant (i, j) ou "" s'il est absent.
func (s segment) comp(i, j int) string {
	el := s.element(i)
	if j < 0 || j >= len(el) {
		return ""
	}
	return el[j]
}

var errEmpty = errors.New("interchange EDIFACT vide")

// tokenize découpe un interchange EDIFACT en segments, en respectant l'éventuel segment UNA et
// le caractère d'échappement. Les blancs entre segments (CR/LF, espaces d'indentation) sont
// ignorés. La fonction est linéaire et bornée par la taille de l'entrée (pas de DoS).
func tokenize(data []byte) ([]segment, delimiters, error) {
	del := defaultDelimiters()
	s := string(data)

	// Segment UNA optionnel : « UNA » suivi de 6 caractères de service.
	if i := strings.Index(s, "UNA"); i >= 0 && i+9 <= len(s) {
		// N'accepter UNA qu'en tête réelle (précédé uniquement de blancs).
		if strings.TrimSpace(s[:i]) == "" {
			svc := s[i+3 : i+9]
			del.component = svc[0]
			del.data = svc[1]
			del.decimal = svc[2]
			del.release = svc[3]
			// svc[4] = séparateur de répétition/réservé (espace le plus souvent) — ignoré ici.
			del.segment = svc[5]
			s = s[i+9:]
		}
	}

	var segs []segment
	var raw strings.Builder
	escaped := false
	flush := func() {
		seg := strings.TrimLeft(raw.String(), " \t\r\n")
		raw.Reset()
		if seg == "" {
			return
		}
		segs = append(segs, splitSegment(seg, del))
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			raw.WriteByte(c)
			escaped = false
			continue
		}
		switch c {
		case del.release:
			raw.WriteByte(c) // conservé : splitSegment ré-interprète l'échappement
			escaped = true
		case del.segment:
			flush()
		default:
			raw.WriteByte(c)
		}
	}
	flush() // segment final éventuellement non terminé

	if len(segs) == 0 {
		return nil, del, errEmpty
	}
	return segs, del, nil
}

// splitSegment éclate un segment brut (sans son terminateur) en étiquette + éléments/composants,
// en respectant le caractère d'échappement.
func splitSegment(raw string, del delimiters) segment {
	fields := splitEscaped(raw, del.data, del.release)
	seg := segment{}
	if len(fields) > 0 {
		seg.tag = strings.TrimSpace(fields[0])
		fields = fields[1:]
	}
	for _, f := range fields {
		seg.elements = append(seg.elements, splitEscaped(f, del.component, del.release))
	}
	return seg
}

// splitEscaped découpe s sur sep, sauf lorsque sep est précédé du caractère d'échappement rel.
// L'échappement est retiré du résultat.
func splitEscaped(s string, sep, rel byte) []string {
	var out []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == rel && i+1 < len(s) {
			cur.WriteByte(s[i+1]) // caractère littéral échappé
			i++
			continue
		}
		if c == sep {
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	out = append(out, cur.String())
	return out
}
