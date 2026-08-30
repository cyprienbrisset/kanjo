package render

import (
	_ "embed"
	"encoding/base64"
	"html/template"
)

// Le logo est embarqué et injecté en data-URI : les sorties HTML (rapport de validation, face
// lisible) restent autonomes et imprimables, sans aucune dépendance réseau (§17.2).

//go:embed assets/logo-64.png
var logoPNG []byte

//go:embed assets/favicon-32.png
var faviconPNG []byte

var (
	logoDataURI    = "data:image/png;base64," + base64.StdEncoding.EncodeToString(logoPNG)
	faviconDataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(faviconPNG)
)

// assetFuncs expose le logo et le favicon aux gabarits. template.URL contourne l'assainisseur
// d'URL de html/template, qui bloquerait sinon le schéma data:.
var assetFuncs = template.FuncMap{
	"logoURI":    func() template.URL { return template.URL(logoDataURI) },
	"faviconURI": func() template.URL { return template.URL(faviconDataURI) },
}
