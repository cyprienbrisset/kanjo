package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

// entryHash calcule l'empreinte SHA-256 d'une entrée, champ Hash exclu. Le chaînage inclut Seq et
// PrevHash, si bien que toute modification, suppression ou réinsertion rompt la chaîne.
func entryHash(e Entry) string {
	e.Hash = ""
	data, _ := json.Marshal(e)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ChainIssue décrit une rupture de chaîne détectée à la vérification.
type ChainIssue struct {
	Index   int    `json:"index"`   // position (0-based) dans la liste d'entrées
	Seq     int64  `json:"seq"`     // numéro de séquence de l'entrée
	Problem string `json:"problem"` // nature du problème
}

// ChainReport est le bilan de vérification d'intégrité d'un journal.
type ChainReport struct {
	Total     int          `json:"total"`     // entrées lues
	Chained   int          `json:"chained"`   // entrées portant un chaînage
	Unchained int          `json:"unchained"` // entrées héritées sans chaînage (Hash vide)
	Issues    []ChainIssue `json:"issues"`    // ruptures détectées
	OK        bool         `json:"ok"`        // vrai si aucune rupture parmi les entrées chaînées
}

// VerifyChain recalcule et vérifie la chaîne d'intégrité d'une liste d'entrées ordonnée.
//
// Garanties (ce qui EST détecté) : toute modification d'une entrée chaînée (empreinte recalculée),
// toute suppression ou réinsertion au milieu (rupture de prevHash / de contiguïté de seq), et toute
// troncature en TÊTE de la portion chaînée — car la première entrée chaînée doit être la genèse
// (prevHash vide) ; un prevHash non vide en tête trahit des entrées antérieures disparues. Une entrée
// non chaînée intercalée APRÈS le début du chaînage est également signalée (un journal chaîné ne
// régresse pas). Les entrées héritées sans empreinte situées AVANT tout chaînage sont comptées à part.
//
// Limite (ce qui n'est PAS détecté, §17.7 — ne pas survendre) : le chaînage est *tamper-evident*, non
// *notarié*. Sans ancrage externe (horodatage signé, journal distant, WORM), la suppression de la
// TOTALITÉ de la portion chaînée — ou le remplacement du fichier par une chaîne cohérente plus courte
// forgée par un attaquant disposant du binaire — ne laisse pas de trace vérifiable localement.
func VerifyChain(entries []Entry) ChainReport {
	rep := ChainReport{Total: len(entries), OK: true}
	var prevHash string
	var prevSeq int64
	started := false
	for i, e := range entries {
		if e.Hash == "" {
			rep.Unchained++
			// Une entrée non chaînée après le début du chaînage est anormale (les entrées héritées
			// ne peuvent apparaître qu'en tête, avant la première entrée chaînée).
			if started {
				rep.Issues = append(rep.Issues, ChainIssue{Index: i, Seq: e.Seq, Problem: "entrée non chaînée intercalée après le début du chaînage (insertion probable)"})
				rep.OK = false
			}
			continue
		}
		rep.Chained++
		if got := entryHash(e); got != e.Hash {
			rep.Issues = append(rep.Issues, ChainIssue{Index: i, Seq: e.Seq, Problem: "empreinte invalide (entrée modifiée)"})
			rep.OK = false
		}
		if !started {
			// Ancrage initial : la première entrée chaînée est la genèse et porte un prevHash vide.
			// Un prevHash non vide révèle une troncature du début du journal.
			if e.PrevHash != "" {
				rep.Issues = append(rep.Issues, ChainIssue{Index: i, Seq: e.Seq, Problem: "ancrage initial manquant : la première entrée chaînée référence une entrée absente (troncature en tête)"})
				rep.OK = false
			}
		} else {
			if e.PrevHash != prevHash {
				rep.Issues = append(rep.Issues, ChainIssue{Index: i, Seq: e.Seq, Problem: "prevHash ne correspond pas à l'entrée précédente (suppression ou réinsertion)"})
				rep.OK = false
			}
			if e.Seq != prevSeq+1 {
				rep.Issues = append(rep.Issues, ChainIssue{Index: i, Seq: e.Seq, Problem: fmt.Sprintf("numéro de séquence non contigu (attendu %d)", prevSeq+1)})
				rep.OK = false
			}
		}
		prevHash = e.Hash
		prevSeq = e.Seq
		started = true
	}
	return rep
}

// FilterByPeriod ne garde que les entrées dont l'horodatage est dans [from, to] (bornes incluses).
// Une borne zéro est ignorée. Les horodatages illisibles sont conservés (prudence : ne rien perdre).
func FilterByPeriod(entries []Entry, from, to time.Time) []Entry {
	if from.IsZero() && to.IsZero() {
		return entries
	}
	var out []Entry
	for _, e := range entries {
		ts, err := time.Parse(time.RFC3339Nano, e.Ts)
		if err != nil {
			out = append(out, e)
			continue
		}
		if !from.IsZero() && ts.Before(from) {
			continue
		}
		if !to.IsZero() && ts.After(to) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// ExportHTML rend un rapport d'audit consolidé, imprimable, SANS aucune donnée personnelle
// (uniquement horodatages, actions, formats, empreintes techniques). Le bilan d'intégrité y figure.
func ExportHTML(entries []Entry, title string) []byte {
	rep := VerifyChain(entries)
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"fr\"><head><meta charset=\"utf-8\">\n")
	b.WriteString("<title>" + html.EscapeString(title) + "</title>\n")
	b.WriteString(`<style>body{font-family:system-ui,sans-serif;margin:2rem;color:#1a1a1a}` +
		`h1{font-size:1.4rem}table{border-collapse:collapse;width:100%;font-size:.85rem}` +
		`th,td{border:1px solid #ddd;padding:6px 8px;text-align:left}th{background:#f4f4f4}` +
		`.ok{color:#2e7d32;font-weight:bold}.ko{color:#b71c1c;font-weight:bold}` +
		`.meta{color:#555;font-size:.85rem;margin:.4rem 0 1.2rem}code{font-family:ui-monospace,monospace}</style>`)
	b.WriteString("</head><body>\n")
	b.WriteString("<h1>" + html.EscapeString(title) + "</h1>\n")

	status := `<span class="ok">intègre</span>`
	if !rep.OK {
		status = `<span class="ko">RUPTURE DÉTECTÉE</span>`
	}
	fmt.Fprintf(&b, `<p class="meta">%d entrées · %d chaînées · %d héritées (non chaînées) · intégrité : %s</p>`+"\n",
		rep.Total, rep.Chained, rep.Unchained, status)

	if !rep.OK {
		b.WriteString(`<p class="ko">Ruptures :</p><ul>`)
		for _, is := range rep.Issues {
			fmt.Fprintf(&b, "<li>entrée #%d (seq %d) : %s</li>", is.Index, is.Seq, html.EscapeString(is.Problem))
		}
		b.WriteString("</ul>\n")
	}

	b.WriteString("<table>\n<tr><th>Horodatage</th><th>Action</th><th>Acteur</th><th>Entrée→Sortie</th>" +
		"<th>Profil</th><th>Verdict</th><th>Pertes</th><th>Seq</th><th>Empreinte</th></tr>\n")
	for _, e := range entries {
		hashShort := e.Hash
		if len(hashShort) > 12 {
			hashShort = hashShort[:12] + "…"
		}
		fmt.Fprintf(&b, "<tr><td>%s</td><td>%s</td><td>%s</td><td>%s → %s</td><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td><code>%s</code></td></tr>\n",
			html.EscapeString(e.Ts), html.EscapeString(e.Action), html.EscapeString(e.Actor),
			html.EscapeString(e.InputFormat), html.EscapeString(e.OutputFormat),
			html.EscapeString(e.Profile), html.EscapeString(e.Verdict), e.LossCount, e.Seq,
			html.EscapeString(hashShort))
	}
	b.WriteString("</table>\n</body></html>\n")
	return []byte(b.String())
}
