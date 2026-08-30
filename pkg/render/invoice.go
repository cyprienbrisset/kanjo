package render

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/cyprienbrisset/kanjo/pkg/model"
)

// designCSS reprend les jetons du système de design 大福帳 (§12.2 thème 昼 Hiru) : encres
// indigo, papier, filets, aucune ombre, aucun rayon (hors sceau). Autonome (aucun réseau).
const designCSS = `
:root{
 --ink-900:#14243D;--ink-700:#24405E;--ink-500:#5A6B7D;--ink-300:#8E9AA6;
 --paper-0:#FBFAF6;--paper-1:#F3F1EA;--paper-2:#E8E5DB;
 --rule-hair:#D5D0C4;--rule-firm:#B9B2A2;--seal:#C8402E;--koke:#5E7A4A;--beni:#9E2B32;--kohaku:#B8862F;
}
*{box-sizing:border-box}
body{margin:0;background:var(--paper-0);color:var(--ink-900);
 font-family:"Zen Kaku Gothic New","Helvetica Neue",Arial,sans-serif;font-size:15px;line-height:1.55}
.sheet{max-width:820px;margin:24px auto;background:var(--paper-1);border-left:3px solid var(--ink-700);
 padding:32px 32px 32px 48px}
h1{font-family:"Shippori Mincho B1",Georgia,serif;font-size:30px;font-weight:400;margin:0 0 4px;color:var(--ink-700)}
.doc-id{font-family:"IBM Plex Mono",monospace;color:var(--ink-500)}
.meta{color:var(--ink-500);font-size:13px;margin-bottom:24px}
.parties{display:flex;gap:48px;margin-bottom:24px}
.party{flex:1}
.party .role{font-size:11px;letter-spacing:.08em;color:var(--ink-500);margin-bottom:4px}
.party .name{font-weight:400;color:var(--ink-900)}
.party .id{font-family:"IBM Plex Mono",monospace;font-size:12px;color:var(--ink-500)}
table{width:100%;border-collapse:collapse;margin:16px 0}
th{font-size:11px;letter-spacing:.08em;color:var(--ink-500);text-align:left;font-weight:400;
 border-bottom:1px solid var(--rule-firm);padding:6px 8px}
td{padding:6px 8px;border-bottom:1px solid var(--rule-hair)}
td.num,th.num{text-align:right;font-family:"IBM Plex Mono",monospace;font-variant-numeric:tabular-nums}
.totals{margin-left:auto;width:320px;margin-top:16px}
.totals td{border:none;padding:3px 8px}
.totals .grand td{border-top:1px solid var(--rule-firm);font-weight:400;font-size:19px;color:var(--ink-700)}
.notes{margin-top:24px;font-size:13px;color:var(--ink-500)}
.seal{display:inline-block;width:44px;height:44px;line-height:40px;text-align:center;border-radius:50%;
 border:2px solid var(--koke);color:var(--koke);font-family:"Shippori Mincho B1",serif;
 transform:rotate(-7deg);float:right}
.seal.err{border-color:var(--beni);color:var(--beni)}
.seal.warn{border-color:var(--kohaku);color:var(--kohaku)}
.foot{margin-top:32px;padding-top:8px;border-top:1px solid var(--rule-hair);font-size:11px;color:var(--ink-300)}
`

type invoiceView struct {
	Doc       *model.Document
	Lang      Lang
	Title     string
	Seal      string // ✓/▲/✕ (décoratif)
	SealLabel string // libellé français du sceau (§12.10)
	SealErr   bool
	SealWarn  bool
	CSS       template.CSS
}

var invoiceFuncs = template.FuncMap{
	"money": FormatMoney,
	"pct":   FormatPercent,
	"catLabel": func(c model.TaxCategoryCode, l Lang) string {
		return c.Label(l)
	},
}

var invoiceTmpl = template.Must(template.New("invoice").Funcs(invoiceFuncs).Funcs(assetFuncs).Parse(`<!doctype html>
<html lang="{{.Lang}}"><head><meta charset="utf-8">
<title>{{.Title}} {{.Doc.ID}}</title>
<link rel="icon" type="image/png" href="{{faviconURI}}"><style>{{.CSS}}</style></head>
<body><div class="sheet">
 {{if .Seal}}<div class="seal{{if .SealErr}} err{{else if .SealWarn}} warn{{end}}" role="img" aria-label="{{.SealLabel}}" title="{{.SealLabel}}">{{.Seal}}</div>{{end}}
 <img src="{{logoURI}}" alt="Kanjō" width="34" height="34" style="border-radius:7px;margin-bottom:6px">
 <h1>{{.Title}}</h1>
 <div class="doc-id">{{.Doc.ID}}</div>
 <div class="meta">Émise le {{.Doc.IssueDate.ISO}}{{if .Doc.DueDate}} · échéance {{.Doc.DueDate.ISO}}{{end}}</div>
 <div class="parties">
  <div class="party"><div class="role">Émetteur</div>
   <div class="name">{{.Doc.Seller.Name}}</div>
   {{if .Doc.Seller.VATID}}<div class="id">TVA {{.Doc.Seller.VATID}}</div>{{end}}
   <div class="id">{{.Doc.Seller.Address.City}} {{.Doc.Seller.Address.PostalCode}}</div>
  </div>
  <div class="party"><div class="role">Destinataire</div>
   <div class="name">{{.Doc.Buyer.Name}}</div>
   {{if .Doc.Buyer.VATID}}<div class="id">TVA {{.Doc.Buyer.VATID}}</div>{{end}}
   <div class="id">{{.Doc.Buyer.Address.City}} {{.Doc.Buyer.Address.PostalCode}}</div>
  </div>
 </div>
 <table><thead><tr>
  <th>Désignation</th><th class="num">Qté</th><th class="num">PU HT</th>
  <th class="num">TVA</th><th class="num">Montant HT</th></tr></thead>
  <tbody>{{$l := .Lang}}{{range .Doc.Lines}}<tr>
   <td>{{.Name}}</td>
   <td class="num">{{.Quantity.String}}</td>
   <td class="num">{{money .NetPrice $l}}</td>
   <td class="num">{{if .TaxRate}}{{pct .TaxRate}}{{else}}—{{end}}</td>
   <td class="num">{{money .NetAmount $l}}</td>
  </tr>{{end}}</tbody>
 </table>
 <table class="totals">
  <tr><td>Total HT</td><td class="num">{{money .Doc.Totals.TaxExclusiveAmount $l}}</td></tr>
  <tr><td>Total TVA</td><td class="num">{{money .Doc.Totals.TaxAmount $l}}</td></tr>
  <tr class="grand"><td>Total TTC</td><td class="num">{{money .Doc.Totals.TaxInclusiveAmount $l}}</td></tr>
  <tr><td>Net à payer</td><td class="num">{{money .Doc.Totals.DuePayableAmount $l}}</td></tr>
 </table>
 {{range .Doc.TaxBreakdown}}{{if .ExemptionReason}}<div class="notes">{{catLabel .Category $l}} : {{.ExemptionReason}}</div>{{end}}{{end}}
 {{range .Doc.Notes}}<div class="notes">{{.Content}}</div>{{end}}
 <div class="foot">Document lisible généré par Kanjō. Ne remplace pas le fichier structuré.</div>
</div></body></html>`))

// RenderInvoiceHTML produit la face lisible HTML d'une facture. Le paramètre verdict
// (facultatif : "ok" | "warning" | "error" | "") reflète un verdict de validation déjà
// calculé et détermine le badge de statut (symbole ✓ / ▲ / ✕ + libellé français, §12.10).
func RenderInvoiceHTML(doc *model.Document, lang Lang, verdict string) ([]byte, error) {
	if lang == "" {
		lang = model.LangFR
	}
	title := "Facture"
	if doc.IsCreditNote() {
		title = "Avoir"
	}
	sym := map[string]string{"ok": "✓", "warning": "▲", "error": "✕"}[verdict]
	v := invoiceView{
		Doc: doc, Lang: lang, Title: title, Seal: sym,
		SealLabel: sealLabel(verdict),
		SealErr:   verdict == "error",
		SealWarn:  verdict == "warning",
		CSS:       template.CSS(designCSS),
	}
	var buf bytes.Buffer
	if err := invoiceTmpl.Execute(&buf, v); err != nil {
		return nil, fmt.Errorf("rendu HTML: %w", err)
	}
	return buf.Bytes(), nil
}

// sealLabel renvoie le libellé français d'un verdict (§12.10 : symbole toujours doublé).
func sealLabel(verdict string) string {
	switch verdict {
	case "ok":
		return "conforme"
	case "warning":
		return "sous réserve"
	case "error":
		return "non conforme"
	default:
		return "non validé"
	}
}
