package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/departments/legal"
)

type LegalDraft struct {
	ID                 int64
	ProjectID          int64
	ProjectName        string
	CounterpartyUserID int64
	CounterpartyName   string
	Template           string
	Title              string
	Body               string
	Flags              []string
	NeedsLawyer        bool
	Status             string
	Rationale          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (s *Store) migrateLegal(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS legal_drafts (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  counterparty_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  counterparty_name TEXT NOT NULL DEFAULT '',
  template TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  flags JSONB NOT NULL DEFAULT '[]'::jsonb,
  needs_lawyer BOOLEAN NOT NULL DEFAULT false,
  status TEXT NOT NULL DEFAULT 'proposed',
  rationale TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS legal_drafts_project_idx ON legal_drafts (project_id, created_at DESC);
`)
	return err
}

func (d LegalDraft) TemplateLabel() string {
	if spec, ok := legal.Lookup(d.Template); ok {
		return spec.Label
	}
	if d.Template == "" || d.Template == "none" {
		return "review"
	}
	return d.Template
}

func (d LegalDraft) StatusStamp() string {
	switch d.Status {
	case "approved":
		return "ok"
	case "rejected":
		return "dark"
	default:
		return "brass"
	}
}

func (d LegalDraft) Proposed() bool { return d.Status == "proposed" }

func legalLabel(id string) string {
	if spec, ok := legal.Lookup(id); ok {
		return spec.Label
	}
	return "Review"
}

func (s *Store) ListLegalDrafts(ctx context.Context, projectID int64) ([]LegalDraft, error) {
	q := `
SELECT d.id, d.project_id, p.name, COALESCE(d.counterparty_user_id, 0), d.counterparty_name,
       d.template, d.title, d.body, d.flags, d.needs_lawyer, d.status, d.rationale, d.created_at, d.updated_at
FROM legal_drafts d
JOIN projects p ON p.id = d.project_id
`
	args := []any{}
	if projectID > 0 {
		q += ` WHERE d.project_id = $1`
		args = append(args, projectID)
	}
	q += ` ORDER BY d.created_at DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LegalDraft
	for rows.Next() {
		d, err := scanLegalDraft(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetLegalDraft(ctx context.Context, id int64) (LegalDraft, error) {
	row := s.pool.QueryRow(ctx, `
SELECT d.id, d.project_id, p.name, COALESCE(d.counterparty_user_id, 0), d.counterparty_name,
       d.template, d.title, d.body, d.flags, d.needs_lawyer, d.status, d.rationale, d.created_at, d.updated_at
FROM legal_drafts d
JOIN projects p ON p.id = d.project_id
WHERE d.id = $1
`, id)
	return scanLegalDraft(row)
}

func scanLegalDraft(row projectScanner) (LegalDraft, error) {
	var d LegalDraft
	var raw []byte
	err := row.Scan(&d.ID, &d.ProjectID, &d.ProjectName, &d.CounterpartyUserID, &d.CounterpartyName,
		&d.Template, &d.Title, &d.Body, &raw, &d.NeedsLawyer, &d.Status, &d.Rationale, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return LegalDraft{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &d.Flags)
	}
	return d, nil
}

func (s *Store) InsertLegalDraft(ctx context.Context, d LegalDraft) (LegalDraft, error) {
	if d.ProjectID <= 0 {
		return LegalDraft{}, errors.New("project is required")
	}
	d.Status = "proposed"
	if d.Template == "" {
		d.Template = "none"
	}
	if d.Title == "" {
		d.Title = legalLabel(d.Template)
	}
	if d.Flags == nil {
		d.Flags = []string{}
	}
	flags, err := json.Marshal(d.Flags)
	if err != nil {
		return LegalDraft{}, err
	}
	var cp any
	if d.CounterpartyUserID > 0 {
		cp = d.CounterpartyUserID
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO legal_drafts (project_id, counterparty_user_id, counterparty_name, template, title, body, flags, needs_lawyer, status, rationale)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)
RETURNING id
`, d.ProjectID, cp, strings.TrimSpace(d.CounterpartyName), d.Template, d.Title, d.Body, flags, d.NeedsLawyer, d.Status, d.Rationale).Scan(&id)
	if err != nil {
		return LegalDraft{}, err
	}
	return s.GetLegalDraft(ctx, id)
}

func (s *Store) SetLegalStatus(ctx context.Context, id int64, status string) error {
	status = strings.TrimSpace(status)
	if status != "approved" && status != "rejected" {
		return errors.New("status must be approved or rejected")
	}
	d, err := s.GetLegalDraft(ctx, id)
	if err != nil {
		return errors.New("draft not found")
	}
	if d.Status != "proposed" {
		return errors.New("only a proposed draft can be stamped")
	}
	_, err = s.pool.Exec(ctx, `UPDATE legal_drafts SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func draftFromStructured(sd map[string]any, projectID int64, counterparty User, extraName string) (LegalDraft, error) {
	tmpl := strVal(sd["template"])
	if tmpl == "" {
		tmpl = "none"
	}
	if tmpl != "none" {
		if _, ok := legal.Lookup(tmpl); !ok {
			return LegalDraft{}, errors.New("unknown legal template")
		}
	}
	name := strings.TrimSpace(extraName)
	if name == "" {
		name = strVal(sd["counterparty_name"])
	}
	if name == "" && counterparty.Name != "" {
		name = counterparty.Name
	}
	d := LegalDraft{
		ProjectID:          projectID,
		CounterpartyUserID: counterparty.ID,
		CounterpartyName:   name,
		Template:           tmpl,
		Body:               strVal(sd["body"]),
		Flags:              stringSlice(sd["flags"]),
		NeedsLawyer:        boolVal(sd["needs_human_lawyer"]),
		Rationale:          strVal(sd["rationale"]),
		Title:              legalLabel(tmpl),
	}
	if d.Rationale == "" {
		d.Rationale = strings.Join(d.Flags, "; ")
	}
	if d.Body == "" && strVal(sd["action"]) == "draft" {
		return LegalDraft{}, errors.New("draft came back with no body — stamp not written")
	}
	if d.Body == "" && len(d.Flags) == 0 && d.Rationale == "" {
		return LegalDraft{}, errors.New("worker returned no draft and no flags")
	}
	return d, nil
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func boolVal(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}
