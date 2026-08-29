// Package tabular écrit un document pivot au format CSV dénormalisé (« aplati ») :
// une ligne CSV par ligne de facture, l'en-tête de facture étant répété sur chaque ligne.
// Le CSV est produit en UTF-8 sans BOM via encoding/csv (séparateur virgule).
package tabular

import (
	"bytes"
	"encoding/csv"

	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func init() { write.Register("csv", Write) }

// header est l'en-tête de colonnes du CSV, dans l'ordre exact attendu.
var header = []string{
	"invoiceId",
	"issueDate",
	"typeCode",
	"currency",
	"sellerName",
	"sellerVat",
	"buyerName",
	"lineId",
	"lineName",
	"quantity",
	"unitCode",
	"netPrice",
	"taxCategory",
	"taxRate",
	"lineNetAmount",
	"totalHT",
	"totalVAT",
	"totalTTC",
	"duePayable",
}

// Write sérialise le document pivot en CSV dénormalisé.
//
// Chaque ligne de facture (doc.Lines) produit une ligne CSV, avec les champs d'en-tête
// de facture répétés. Si la facture ne comporte aucune ligne, une unique ligne CSV est
// tout de même émise avec les champs d'en-tête remplis et les champs de ligne vides.
func Write(doc *model.Document, opts write.Options) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write(header); err != nil {
		return nil, err
	}

	if len(doc.Lines) == 0 {
		if err := w.Write(row(doc, nil)); err != nil {
			return nil, err
		}
	} else {
		for i := range doc.Lines {
			if err := w.Write(row(doc, &doc.Lines[i])); err != nil {
				return nil, err
			}
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// row construit une ligne CSV à partir de l'en-tête de facture et, si présente, d'une ligne.
func row(doc *model.Document, l *model.Line) []string {
	t := doc.Totals
	rec := []string{
		doc.ID,
		doc.IssueDate.ISO(),
		string(doc.TypeCode),
		doc.CurrencyCode,
		doc.Seller.Name,
		doc.Seller.VATID,
		doc.Buyer.Name,
	}

	if l == nil {
		// Champs de ligne vides : lineId, lineName, quantity, unitCode, netPrice,
		// taxCategory, taxRate, lineNetAmount.
		rec = append(rec, "", "", "", "", "", "", "", "")
	} else {
		taxRate := ""
		if l.TaxRate != nil {
			taxRate = l.TaxRate.String()
		}
		rec = append(rec,
			l.ID,
			l.Name,
			l.Quantity.String(),
			string(l.UnitCode),
			l.NetPrice.String(),
			string(l.TaxCategory),
			taxRate,
			l.NetAmount.String(),
		)
	}

	rec = append(rec,
		t.TaxExclusiveAmount.String(),
		t.TaxAmount.String(),
		t.TaxInclusiveAmount.String(),
		t.DuePayableAmount.String(),
	)
	return rec
}
