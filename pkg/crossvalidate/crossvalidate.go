// Package crossvalidate confronte le verdict de conformité de Kanjō à celui de validateurs
// externes de référence (Mustangproject, validateur KoSIT). Il sert aux dossiers d'agrément et aux
// bancs d'essai : Kanjō ne remplace pas ces outils, il se compare à eux.
//
// Principe d'honnêteté (§17.7) : aucun verdict externe n'est simulé. Si l'outil n'est pas
// configuré/disponible, la confrontation est marquée « non exécutée », jamais « conforme ».
//
// Configuration (aucun appel réseau, tout est local — §17.2) :
//   - KANJO_MUSTANG_JAR : chemin du JAR Mustang-CLI (validation Factur-X/ZUGFeRD).
//   - KANJO_KOSIT_JAR + KANJO_KOSIT_SCENARIOS : validateur KoSIT (XRechnung) et son scénario.
//
// L'exécution requiert « java » dans le PATH.
package crossvalidate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Verdict est le résultat d'une confrontation à un validateur externe.
type Verdict struct {
	Tool      string `json:"tool"`             // "mustangproject" | "kosit"
	Ran       bool   `json:"ran"`              // l'outil a réellement été exécuté
	Compliant bool   `json:"compliant"`        // conforme selon l'outil externe
	Detail    string `json:"detail,omitempty"` // extrait du rapport / raison de non-exécution
}

// Available renvoie la liste des validateurs externes configurés ET exécutables (java présent).
func Available() []string {
	var out []string
	if _, err := exec.LookPath("java"); err != nil {
		return nil
	}
	if os.Getenv("KANJO_MUSTANG_JAR") != "" {
		out = append(out, "mustangproject")
	}
	if os.Getenv("KANJO_KOSIT_JAR") != "" && os.Getenv("KANJO_KOSIT_SCENARIOS") != "" {
		out = append(out, "kosit")
	}
	return out
}

// Run confronte le fichier donné à tous les validateurs externes configurés. Renvoie un Verdict par
// outil (avec Ran=false si l'outil n'est pas configuré). Ne renvoie jamais d'erreur pour un simple
// « outil absent » : c'est un Verdict non exécuté, pas un échec.
func Run(path string) []Verdict {
	var verdicts []Verdict
	hasJava := false
	if _, err := exec.LookPath("java"); err == nil {
		hasJava = true
	}

	// Mustangproject.
	if jar := os.Getenv("KANJO_MUSTANG_JAR"); jar != "" {
		if !hasJava {
			verdicts = append(verdicts, Verdict{Tool: "mustangproject", Detail: "java introuvable dans le PATH"})
		} else {
			verdicts = append(verdicts, runMustang(jar, path))
		}
	}
	// KoSIT.
	if jar := os.Getenv("KANJO_KOSIT_JAR"); jar != "" {
		scen := os.Getenv("KANJO_KOSIT_SCENARIOS")
		switch {
		case !hasJava:
			verdicts = append(verdicts, Verdict{Tool: "kosit", Detail: "java introuvable dans le PATH"})
		case scen == "":
			verdicts = append(verdicts, Verdict{Tool: "kosit", Detail: "KANJO_KOSIT_SCENARIOS non défini"})
		default:
			verdicts = append(verdicts, runKoSIT(jar, scen, path))
		}
	}
	return verdicts
}

func runMustang(jar, path string) Verdict {
	out, _ := runJava(jar, "--action", "validate", "--source", path)
	return Verdict{Tool: "mustangproject", Ran: true,
		Compliant: parseMustang(out), Detail: firstLine(out)}
}

func runKoSIT(jar, scenarios, path string) Verdict {
	out, _ := runJava(jar, "-s", scenarios, "-r", ".", path)
	return Verdict{Tool: "kosit", Ran: true,
		Compliant: parseKoSIT(out), Detail: firstLine(out)}
}

// runJava exécute java -jar <jar> <args...> avec un délai maximal, sans aucun accès réseau initié
// par Kanjō (l'outil externe est local).
func runJava(jar string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	full := append([]string{"-jar", jar}, args...)
	out, err := exec.CommandContext(ctx, "java", full...).CombinedOutput()
	return string(out), err
}

// parseMustang lit le verdict d'un rapport Mustangproject (résumé « status="valid" »).
func parseMustang(out string) bool {
	s := strings.ToLower(out)
	if strings.Contains(s, `status="invalid"`) || strings.Contains(s, "is not valid") {
		return false
	}
	return strings.Contains(s, `status="valid"`) || strings.Contains(s, "is valid")
}

// parseKoSIT lit le verdict d'un rapport KoSIT (recommandation « accept »).
func parseKoSIT(out string) bool {
	s := strings.ToLower(out)
	if strings.Contains(s, "reject") {
		return false
	}
	return strings.Contains(s, "accept")
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	if len(s) > 200 {
		return s[:200]
	}
	return s
}

// Compare confronte le verdict de conformité de Kanjō (kanjoCompliant) aux verdicts externes et
// renvoie, pour chaque outil exécuté, s'il y a accord. Les outils non exécutés sont ignorés.
func Compare(kanjoCompliant bool, verdicts []Verdict) (agreements, disagreements int, lines []string) {
	for _, v := range verdicts {
		if !v.Ran {
			lines = append(lines, fmt.Sprintf("%-15s : non exécuté (%s)", v.Tool, v.Detail))
			continue
		}
		if v.Compliant == kanjoCompliant {
			agreements++
			lines = append(lines, fmt.Sprintf("%-15s : ACCORD (%s)", v.Tool, verdictWord(v.Compliant)))
		} else {
			disagreements++
			lines = append(lines, fmt.Sprintf("%-15s : DÉSACCORD (externe=%s, Kanjō=%s)",
				v.Tool, verdictWord(v.Compliant), verdictWord(kanjoCompliant)))
		}
	}
	return agreements, disagreements, lines
}

func verdictWord(b bool) string {
	if b {
		return "conforme"
	}
	return "non conforme"
}
