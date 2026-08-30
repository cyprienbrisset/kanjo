package xmlsafe

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

// FuzzDecode s'assure que le décodeur XML durci ne panique JAMAIS, quelle que soit l'entrée
// (aléatoire, malformée ou malveillante). Les entrées inattendues doivent devenir des erreurs,
// jamais un crash ni une boucle non bornée.
func FuzzDecode(f *testing.F) {
	// Graines : XML valides, cas limites et charges d'attaque connues.
	f.Add([]byte(`<a><b>x</b></a>`))
	f.Add([]byte(`<?xml version="1.0"?><Invoice><ID>1</ID></Invoice>`))
	f.Add([]byte(``))
	f.Add([]byte(`<`))
	f.Add([]byte("\xEF\xBB\xBF<a/>"))
	f.Add([]byte(`<!DOCTYPE a [<!ENTITY x "y">]><a>&x;</a>`))
	if dir := filepath.Join("..", "..", "testdata", "fuzz", "xxe"); dir != "" {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if b, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
					f.Add(b)
				}
			}
		}
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var v struct {
			XMLName xml.Name
			Inner   string `xml:",innerxml"`
		}
		// Le contrat : pas de panic. Une erreur est un résultat acceptable.
		_ = Decode(data, &v)
	})
}
