package admin

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Grant struct {
	ID          int64
	ProjectID   int64
	ProjectName string
	Title       string
	Funder      string
	Deadline    string
	Fit         string
	NextAction  string
	Status      string
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (g Grant) Open() bool    { return g.Status == "open" || g.Status == "" }
func (g Grant) DueSoon() bool { return g.Open() && g.Deadline != "" }
func (g Grant) StatusStamp() string {
	switch g.Status {
	case "done":
		return "ok"
	case "dropped":
		return "dark"
	default:
		return "brass"
	}
}

func (s *Store) migrateGrants(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS grants (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  funder TEXT NOT NULL DEFAULT '',
  deadline DATE,
  fit TEXT NOT NULL DEFAULT '',
  next_action TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS grants_open_idx ON grants (deadline) WHERE status = 'open';
`)
	return err
}

func (s *Store) ListGrants(ctx context.Context) ([]Grant, error) {
	rows, err := s.pool.Query(ctx, `
SELECT g.id, COALESCE(g.project_id, 0), COALESCE(p.name, ''), g.title, g.funder,
       COALESCE(to_char(g.deadline, 'YYYY-MM-DD'), ''), g.fit, g.next_action, g.status, g.notes,
       g.created_at, g.updated_at
FROM grants g
LEFT JOIN projects p ON p.id = g.project_id
ORDER BY g.status, g.deadline NULLS LAST, g.id DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Grant
	for rows.Next() {
		var g Grant
		if err := rows.Scan(&g.ID, &g.ProjectID, &g.ProjectName, &g.Title, &g.Funder, &g.Deadline,
			&g.Fit, &g.NextAction, &g.Status, &g.Notes, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) CountOpenGrants(ctx context.Context) int {
	var n int
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM grants WHERE status = 'open'`).Scan(&n)
	return n
}

func (s *Store) InsertGrant(ctx context.Context, g Grant) (Grant, error) {
	g.Title = strings.TrimSpace(g.Title)
	if g.Title == "" {
		return Grant{}, errors.New("title is required")
	}
	if g.Status == "" {
		g.Status = "open"
	}
	var due any
	if strings.TrimSpace(g.Deadline) != "" {
		due = g.Deadline
	}
	var project any
	if g.ProjectID > 0 {
		project = g.ProjectID
	}
	err := s.pool.QueryRow(ctx, `
INSERT INTO grants (project_id, title, funder, deadline, fit, next_action, status, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at, updated_at
`, project, g.Title, strings.TrimSpace(g.Funder), due, strings.TrimSpace(g.Fit),
		strings.TrimSpace(g.NextAction), g.Status, strings.TrimSpace(g.Notes)).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return Grant{}, friendlyDBErr(err)
	}
	return g, nil
}

func (s *Store) SetGrantStatus(ctx context.Context, id int64, status string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "open", "done", "dropped":
	default:
		return errors.New("unknown status")
	}
	tag, err := s.pool.Exec(ctx, `UPDATE grants SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("grant not found")
	}
	return nil
}
