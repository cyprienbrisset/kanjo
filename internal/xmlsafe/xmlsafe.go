// Package xmlsafe est le SEUL point d'entrée XML de Kanjō (cahier des charges §17.1).
// Aucun reader ne doit appeler encoding/xml directement.
//
// Il applique les protections suivantes, toutes en pur Go (bibliothèque standard) :
//   - résolution d'entités externes désactivée (XXE) — encoding/xml ne résout jamais
//     d'entité SYSTEM/PUBLIC, et le mode Strict rejette toute entité non prédéfinie ;
//   - déclaration DOCTYPE / DTD refusée (bloque billion laughs et entités personnalisées) ;
//   - aucune résolution d'URI externe, aucun xi:include (traité comme de simples éléments) ;
//   - limite de profondeur d'imbrication (défaut 100) ;
//   - limite du nombre total de jetons (protection contre l'explosion de nœuds) ;
//   - limite de taille d'entrée avant parsing ;
//   - jeux de caractères restreints à un ensemble sûr (UTF-8, US-ASCII, Latin-1/CP1252).
package xmlsafe

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Limits borne les ressources consommées par le parsing XML.
type Limits struct {
	MaxBytes  int64 // taille maximale de l'entrée avant parsing
	MaxDepth  int   // profondeur d'imbrication maximale
	MaxTokens int64 // nombre total de jetons maximal
}

// DefaultLimits renvoie des bornes prudentes adaptées aux factures électroniques.
func DefaultLimits() Limits {
	return Limits{
		MaxBytes:  64 << 20,  // 64 Mo
		MaxDepth:  100,       // §17.1
		MaxTokens: 5_000_000, // borne large mais finie
	}
}

// Erreurs de sécurité XML.
var (
	ErrTooLarge       = errors.New("xmlsafe: entrée XML trop volumineuse")
	ErrDoctype        = errors.New("xmlsafe: déclaration DOCTYPE/DTD refusée")
	ErrDepthExceeded  = errors.New("xmlsafe: profondeur d'imbrication excessive")
	ErrTokensExceeded = errors.New("xmlsafe: nombre de nœuds excessif")
	ErrExternalEntity = errors.New("xmlsafe: entité externe refusée")
	ErrUnsafeCharset  = errors.New("xmlsafe: jeu de caractères non supporté")
)

// Check inspecte le flux XML et applique toutes les bornes de sécurité, sans le désérialiser.
// À appeler avant toute désérialisation.
func Check(data []byte, lim Limits) error {
	if lim.MaxBytes > 0 && int64(len(data)) > lim.MaxBytes {
		return fmt.Errorf("%w: %d octets", ErrTooLarge, len(data))
	}
	dec := newHardenedDecoder(bytes.NewReader(data))

	depth := 0
	var count int64
	for {
		tok, err := dec.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Une entité non prédéfinie (ex. XXE via DTD) provoque une erreur ici.
			if strings.Contains(err.Error(), "entity") {
				return fmt.Errorf("%w: %v", ErrExternalEntity, err)
			}
			return fmt.Errorf("xmlsafe: %w", err)
		}
		count++
		if lim.MaxTokens > 0 && count > lim.MaxTokens {
			return ErrTokensExceeded
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if lim.MaxDepth > 0 && depth > lim.MaxDepth {
				return fmt.Errorf("%w: > %d", ErrDepthExceeded, lim.MaxDepth)
			}
		case xml.EndElement:
			depth--
		case xml.Directive:
			if isDoctype(t) {
				return ErrDoctype
			}
		}
	}
	return nil
}

// isDoctype détecte une directive DOCTYPE (donc une DTD interne ou externe).
func isDoctype(d xml.Directive) bool {
	trimmed := bytes.TrimSpace([]byte(d))
	return bytes.HasPrefix(bytes.ToUpper(trimmed), []byte("DOCTYPE"))
}

// Unmarshal vérifie puis désérialise le XML dans v. C'est la fonction à utiliser partout.
func Unmarshal(data []byte, v any, lim Limits) error {
	if err := Check(data, lim); err != nil {
		return err
	}
	dec := newHardenedDecoder(bytes.NewReader(data))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("xmlsafe: désérialisation: %w", err)
	}
	return nil
}

// Decode est un raccourci d'Unmarshal avec les bornes par défaut.
func Decode(data []byte, v any) error { return Unmarshal(data, v, DefaultLimits()) }

// newHardenedDecoder construit un décodeur encoding/xml configuré de façon sûre.
func newHardenedDecoder(r io.Reader) *xml.Decoder {
	dec := xml.NewDecoder(r)
	dec.Strict = true
	// Entity laissé à nil : seules les entités XML prédéfinies (&lt; &gt; &amp; &apos; &quot;)
	// sont acceptées ; toute autre entité (donc toute entité définie par DTD) est une erreur.
	dec.Entity = nil
	dec.CharsetReader = safeCharsetReader
	return dec
}
