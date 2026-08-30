package pdfa

import (
	"fmt"
	"time"

	pdfcpumodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// embedFacturX ajoute le XML de facture au contexte PDF en respectant l'association Factur-X
// exigée par PDF/A-3 : l'EmbeddedFile porte /Subtype text/xml et /Params, la spécification de
// fichier porte /AFRelationship /Data, et le catalogue référence la spécification via un tableau
// /AF (ISO 19005-3 §6.8, Factur-X §7). Ces éléments sont vérifiables par relecture du PDF produit.
//
// La conformité PDF/A-3 *globale* (polices, couleurs, XMP pdfaid, OutputIntent) dépend du PDF de
// base : cette fonction préserve la structure existante et n'invente aucun verdict (§17.7).
func embedFacturX(xt *pdfcpumodel.XRefTable, xml []byte, attachName, desc string, modTime time.Time) error {
	// 1) Flux du fichier embarqué (EmbeddedFile) : type/sous-type MIME + paramètres.
	sd, err := xt.NewStreamDictForBuf(xml)
	if err != nil {
		return fmt.Errorf("flux embarqué: %w", err)
	}
	sd.InsertName("Type", "EmbeddedFile")
	sd.InsertName("Subtype", "text#2Fxml") // « text/xml » (solidus échappé en notation PDF)
	params := types.NewDict()
	params.InsertInt("Size", len(xml))
	params.Insert("ModDate", types.StringLiteral(types.DateString(modTime)))
	sd.Insert("Params", params)
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encodage du flux embarqué: %w", err)
	}
	streamRef, err := xt.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("insertion du flux embarqué: %w", err)
	}

	// 2) Spécification de fichier avec la relation d'association (BT : la facture EST la donnée).
	fs, err := xt.NewFileSpecDict(attachName, attachName, desc, *streamRef)
	if err != nil {
		return fmt.Errorf("spécification de fichier: %w", err)
	}
	fs.InsertName("AFRelationship", "Data")
	fsRef, err := xt.IndRefForNewObject(fs)
	if err != nil {
		return fmt.Errorf("insertion de la spécification de fichier: %w", err)
	}

	// 3) Arbre de noms EmbeddedFiles (compatibilité lecteurs historiques).
	if err := xt.LocateNameTree("EmbeddedFiles", true); err != nil {
		return fmt.Errorf("arbre EmbeddedFiles: %w", err)
	}
	m := pdfcpumodel.NameMap{attachName: []types.Dict{fs}}
	if err := xt.Names["EmbeddedFiles"].Add(xt, attachName, *fsRef, m, []string{"F", "UF"}); err != nil {
		return fmt.Errorf("arbre EmbeddedFiles: ajout de %q: %w", attachName, err)
	}

	// 4) Tableau /AF du catalogue (Associated Files, PDF/A-3) : c'est ce qui rend le fichier
	// « associé » au document et non une simple pièce jointe.
	cat, err := xt.Catalog()
	if err != nil {
		return fmt.Errorf("catalogue: %w", err)
	}
	var af types.Array
	if existing, ok := cat["AF"].(types.Array); ok {
		af = existing
	}
	af = append(af, *fsRef)
	cat["AF"] = af
	return nil
}
