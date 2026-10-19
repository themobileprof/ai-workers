package admin

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/departments/files"
)

type FileFiling struct {
	ID             int64
	ProjectID      int64
	ProjectName    string
	Slug           string
	Folder         string
	Filename       string
	MIME           string
	TelegramFileID string
	Caption        string
	Path           string
	Rationale      string
	Status         string
	DriveFileID    string
	DriveURL       string
	PlaceError     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Store) migrateFiles(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS file_filings (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT REFERENCES projects(id) ON DELETE CASCADE,
  slug TEXT NOT NULL,
  folder TEXT NOT NULL,
  filename TEXT NOT NULL DEFAULT '',
  mime TEXT NOT NULL DEFAULT '',
  telegram_file_id TEXT NOT NULL DEFAULT '',
  caption TEXT NOT NULL DEFAULT '',
  path TEXT NOT NULL DEFAULT '',
  rationale TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'proposed',
  drive_file_id TEXT NOT NULL DEFAULT '',
  drive_url TEXT NOT NULL DEFAULT '',
  place_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS file_filings_status_idx ON file_filings (status, created_at DESC);
CREATE INDEX IF NOT EXISTS file_filings_project_idx ON file_filings (project_id, created_at DESC);
`)
	return err
}

func (d FileFiling) FolderLabel() string {
	company := d.Slug == files.CompanySlug
	if spec, ok := files.LookupFolder(d.Folder, company); ok {
		return spec.Label
	}
	return d.Folder
}

func (d FileFiling) ScopeLabel() string {
	if d.ProjectName != "" {
		return d.ProjectName
	}
	if d.Slug == files.CompanySlug {
		return "TheMobileProf"
	}
	return d.Slug
}

func (d FileFiling) StatusStamp() string {
	switch d.Status {
	case "accepted", "filed":
		return "ok"
	case "rejected", "failed":
		return "dark"
	default:
		return "brass"
	}
}

func (d FileFiling) Proposed() bool { return d.Status == "proposed" }
func (d FileFiling) Waiting() bool  { return d.Status == "accepted" }

func (s *Store) ListFileFilings(ctx context.Context, projectID int64, status string) ([]FileFiling, error) {
	q := `
SELECT d.id, COALESCE(d.project_id, 0), COALESCE(p.name, ''), d.slug, d.folder, d.filename, d.mime,
       d.telegram_file_id, d.caption, d.path, d.rationale, d.status, d.drive_file_id, d.drive_url, d.place_error,
       d.created_at, d.updated_at
FROM file_filings d
LEFT JOIN projects p ON p.id = d.project_id
WHERE 1=1
`
	args := []any{}
	n := 0
	if projectID > 0 {
		n++
		q += ` AND d.project_id = $` + strconv.Itoa(n)
		args = append(args, projectID)
	}
	if status != "" {
		n++
		q += ` AND d.status = $` + strconv.Itoa(n)
		args = append(args, status)
	}
	q += ` ORDER BY d.created_at DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileFiling
	for rows.Next() {
		d, err := scanFileFiling(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetFileFiling(ctx context.Context, id int64) (FileFiling, error) {
	row := s.pool.QueryRow(ctx, `
SELECT d.id, COALESCE(d.project_id, 0), COALESCE(p.name, ''), d.slug, d.folder, d.filename, d.mime,
       d.telegram_file_id, d.caption, d.path, d.rationale, d.status, d.drive_file_id, d.drive_url, d.place_error,
       d.created_at, d.updated_at
FROM file_filings d
LEFT JOIN projects p ON p.id = d.project_id
WHERE d.id = $1
`, id)
	return scanFileFiling(row)
}

func scanFileFiling(row projectScanner) (FileFiling, error) {
	var d FileFiling
	err := row.Scan(&d.ID, &d.ProjectID, &d.ProjectName, &d.Slug, &d.Folder, &d.Filename, &d.MIME,
		&d.TelegramFileID, &d.Caption, &d.Path, &d.Rationale, &d.Status, &d.DriveFileID, &d.DriveURL, &d.PlaceError,
		&d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return FileFiling{}, err
	}
	return d, nil
}

func (s *Store) InsertFileFiling(ctx context.Context, d FileFiling) (FileFiling, error) {
	d.Slug = strings.ToLower(strings.TrimSpace(d.Slug))
	d.Folder = strings.ToLower(strings.TrimSpace(d.Folder))
	d.Filename = strings.TrimSpace(d.Filename)
	if d.Filename == "" {
		return FileFiling{}, errors.New("filename is required")
	}
	company := d.Slug == files.CompanySlug
	if d.Slug == "" {
		return FileFiling{}, errors.New("project folder is required")
	}
	if _, ok := files.LookupFolder(d.Folder, company); !ok {
		return FileFiling{}, errors.New("folder is not in the catalog")
	}
	if !company && d.ProjectID <= 0 {
		return FileFiling{}, errors.New("project is required")
	}
	if company {
		d.ProjectID = 0
	}
	d.Status = "proposed"
	d.Path = files.Path(d.Slug, d.Folder, d.Filename)
	var id int64
	var project any
	if d.ProjectID > 0 {
		project = d.ProjectID
	}
	err := s.pool.QueryRow(ctx, `
INSERT INTO file_filings (project_id, slug, folder, filename, mime, telegram_file_id, caption, path, rationale, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id
`, project, d.Slug, d.Folder, d.Filename, d.MIME, d.TelegramFileID, d.Caption, d.Path, d.Rationale, d.Status).Scan(&id)
	if err != nil {
		return FileFiling{}, err
	}
	return s.GetFileFiling(ctx, id)
}

func (s *Store) SetFileFilingStatus(ctx context.Context, id int64, status string) error {
	status = strings.TrimSpace(status)
	d, err := s.GetFileFiling(ctx, id)
	if err != nil {
		return errors.New("filing not found")
	}
	switch status {
	case "accepted", "rejected":
		if d.Status != "proposed" {
			return errors.New("only a proposed filing can be stamped")
		}
	case "filed", "failed":
		if d.Status != "accepted" && d.Status != "failed" {
			return errors.New("only an accepted filing can be placed")
		}
	default:
		return errors.New("unknown status")
	}
	_, err = s.pool.Exec(ctx, `UPDATE file_filings SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func (s *Store) MarkFileFiled(ctx context.Context, id int64, driveID, driveURL, placeErr string) error {
	d, err := s.GetFileFiling(ctx, id)
	if err != nil {
		return errors.New("filing not found")
	}
	if d.Status != "accepted" && d.Status != "failed" {
		return errors.New("only an accepted filing can be placed")
	}
	status := "filed"
	if strings.TrimSpace(placeErr) != "" && driveID == "" {
		status = "failed"
	}
	_, err = s.pool.Exec(ctx, `
UPDATE file_filings SET status=$2, drive_file_id=$3, drive_url=$4, place_error=$5, updated_at=now() WHERE id=$1
`, id, status, strings.TrimSpace(driveID), strings.TrimSpace(driveURL), strings.TrimSpace(placeErr))
	return err
}
