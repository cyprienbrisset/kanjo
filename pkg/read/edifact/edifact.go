// Package edifact lit les messages UN/EDIFACT INVOIC (factures) et les convertit vers le modèle
// pivot. Contrairement aux autres formats, EDIFACT n'est pas du XML : le flux est tokenisé par un
// analyseur dédié (ISO 9735), conscient du segment de service UNA et du caractère d'échappement.
//
// Périmètre : lecture seule des messages INVOIC (types 380/381/386…). Les segments non porteurs de
// sens métier pour le pivot sont ignorés ; l'écriture EDIFACT et le rapport de perte détaillé
// relèvent d'un lot ultérieur (L4).
package edifact

import (
	"fmt"
	"strings"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/read"
)

func init() { read.Register(read.FormatEDIFACT, Read) }

// Read tokenise un interchange EDIFACT et convertit le premier message INVOIC en document pivot.
func Read(data []byte, sourceName string) (*model.Document, error) {
	segs, _, err := tokenize(data)
	if err != nil {
		return nil, fmt.Errorf("lecture EDIFACT %s: %w", sourceName, err)
	}

	profile, msg := messageProfile(segs)
	if msg != "INVOIC" {
		return nil, fmt.Errorf("lecture EDIFACT %s: message %q non pris en charge (INVOIC attendu)", sourceName, msg)
	}
	return mapToPivot(segs, profile, sourceName)
}

// messageProfile lit l'en-tête de message UNH pour identifier le type de message et son profil
// (ex. « INVOIC:D:97A:UN »). Renvoie le type de message et l'identifiant de spécification complet.
func messageProfile(segs []segment) (profile, msgType string) {
	for _, s := range segs {
		if s.tag == "UNH" {
			c := s.element(1) // S009 : type:version:release:agence[:code]
			msgType = strings.TrimSpace(s.comp(1, 0))
			profile = strings.Join(nonEmpty(c), ":")
			return profile, msgType
		}
	}
	return "", ""
}

func nonEmpty(comps []string) []string {
	var out []string
	for _, c := range comps {
		if v := strings.TrimSpace(c); v != "" {
			out = append(out, v)
		}
	}
	return out
}
