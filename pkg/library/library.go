// Package library est la bibliothèque locale (蔵 Kura, §G8) : un index SQLite des documents
// traités par Kanjō, pour les retrouver et les rejouer. Stockage 100 % local, pur Go
// (modernc.org/sqlite, sans cgo — ADR-002 préservé).
//
// Conformité (§17.4) : l'index stocke des métadonnées et des empreintes, PAS le contenu des
// factures ni les fichiers eux-mêmes. Il est purgeable (droit à l'effacement) et soumis à une
// rétention configurable (défaut 13 mois, à définir selon la politique de conservation retenue).
package library

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Record est une entrée d'index (métadonnées d'un document traité).
type Record struct {
	ID          string `json:"id"`
	IssueDate   string `json:"issueDate"`
	SellerName  string `json:"sellerName"`
	BuyerName   string `json:"buyerName"`
	TotalTTC    string `json:"totalTtc"`
	Currency    string `json:"currency"`
	Format      string `json:"format"`
	Profile     string `json:"profile"`
	Verdict     string `json:"verdict"`
	InputSha256 string `json:"inputSha256"`
	InputPath   string `json:"inputPath"`
	OutputPath  string `json:"outputPath,omitempty"`
	Batch       string `json:"batch,omitempty"`
	ProcessedAt string `json:"processedAt"`
}

// Query filtre une recherche à facettes.
type Query struct {
	Text    string // recherche sur id / vendeur / acheteur
	Verdict string
	Format  string
	From    string // date d'émission min (AAAA-MM-JJ)
	To      string // date d'émission max
	Limit   int
}

// Library est l'index local.
type Library struct{ db *sql.DB }

// DefaultPath renvoie le chemin standard de la base (répertoire de données, §15.1).
func DefaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "kanjo", "library.db"), nil
}

// Open ouvre (ou crée) la bibliothèque et son schéma.
func Open(path string) (*Library, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la bibliothèque %q: %w", path, err)
	}
	l := &Library{db: db}
	if err := l.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return l, nil
}

func (l *Library) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS documents (
 id TEXT, issue_date TEXT, seller_name TEXT, buyer_name TEXT,
 total_ttc TEXT, currency TEXT, format TEXT, profile TEXT, verdict TEXT,
 input_sha256 TEXT PRIMARY KEY, input_path TEXT, output_path TEXT,
 batch TEXT, processed_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_issue ON documents(issue_date);
CREATE INDEX IF NOT EXISTS idx_seller ON documents(seller_name);
CREATE INDEX IF NOT EXISTS idx_verdict ON documents(verdict);
`
	_, err := l.db.Exec(schema)
	return err
}

// Index insère ou met à jour un document (clé : empreinte d'entrée).
func (l *Library) Index(r Record) error {
	if r.ProcessedAt == "" {
		r.ProcessedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := l.db.Exec(`
INSERT INTO documents (id,issue_date,seller_name,buyer_name,total_ttc,currency,format,profile,verdict,input_sha256,input_path,output_path,batch,processed_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(input_sha256) DO UPDATE SET
 id=excluded.id,issue_date=excluded.issue_date,seller_name=excluded.seller_name,buyer_name=excluded.buyer_name,
 total_ttc=excluded.total_ttc,currency=excluded.currency,format=excluded.format,profile=excluded.profile,
 verdict=excluded.verdict,input_path=excluded.input_path,output_path=excluded.output_path,batch=excluded.batch,
 processed_at=excluded.processed_at`,
		r.ID, r.IssueDate, r.SellerName, r.BuyerName, r.TotalTTC, r.Currency, r.Format, r.Profile,
		r.Verdict, r.InputSha256, r.InputPath, r.OutputPath, r.Batch, r.ProcessedAt)
	return err
}

// Search interroge l'index avec des filtres à facettes.
func (l *Library) Search(q Query) ([]Record, error) {
	var where []string
	var args []any
	if q.Text != "" {
		where = append(where, "(id LIKE ? OR seller_name LIKE ? OR buyer_name LIKE ?)")
		like := "%" + q.Text + "%"
		args = append(args, like, like, like)
	}
	if q.Verdict != "" {
		where = append(where, "verdict = ?")
		args = append(args, q.Verdict)
	}
	if q.Format != "" {
		where = append(where, "format = ?")
		args = append(args, q.Format)
	}
	if q.From != "" {
		where = append(where, "issue_date >= ?")
		args = append(args, q.From)
	}
	if q.To != "" {
		where = append(where, "issue_date <= ?")
		args = append(args, q.To)
	}
	sqlStr := "SELECT id,issue_date,seller_name,buyer_name,total_ttc,currency,format,profile,verdict,input_sha256,input_path,output_path,batch,processed_at FROM documents"
	if len(where) > 0 {
		sqlStr += " WHERE " + strings.Join(where, " AND ")
	}
	sqlStr += " ORDER BY issue_date DESC, id"
	if q.Limit > 0 {
		sqlStr += fmt.Sprintf(" LIMIT %d", q.Limit)
	}
	rows, err := l.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.ID, &r.IssueDate, &r.SellerName, &r.BuyerName, &r.TotalTTC, &r.Currency,
			&r.Format, &r.Profile, &r.Verdict, &r.InputSha256, &r.InputPath, &r.OutputPath, &r.Batch, &r.ProcessedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Count renvoie le nombre de documents indexés.
func (l *Library) Count() (int, error) {
	var n int
	err := l.db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&n)
	return n, err
}

// Forget supprime les documents correspondant au critère texte (droit à l'effacement RGPD,
// §17.4). Renvoie le nombre de lignes supprimées.
func (l *Library) Forget(text string) (int64, error) {
	like := "%" + text + "%"
	res, err := l.db.Exec("DELETE FROM documents WHERE id LIKE ? OR seller_name LIKE ? OR buyer_name LIKE ? OR input_sha256 = ?",
		like, like, like, text)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PurgeBefore supprime les documents traités avant la date donnée (rétention, §17.4).
func (l *Library) PurgeBefore(cutoff time.Time) (int64, error) {
	res, err := l.db.Exec("DELETE FROM documents WHERE processed_at < ?", cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Close ferme la base.
func (l *Library) Close() error { return l.db.Close() }
