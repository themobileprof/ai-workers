package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/departments/hr"
)

type HrRole struct {
	ID          int64
	ProjectID   int64
	ProjectName string
	Slug        string
	Title       string
	Kind        string
	Template    string
	JD          string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type HrApplication struct {
	ID             int64
	RoleID         int64
	RoleTitle      string
	RoleSlug       string
	Name           string
	Email          string
	Phone          string
	Source         string
	CVText         string
	Score          int
	Recommendation string
	Reasons        []string
	Status         string
	Rationale      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Store) migrateHR(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS hr_roles (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
  slug TEXT NOT NULL,
  title TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'contractor',
  template TEXT NOT NULL DEFAULT 'contractor',
  jd TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'proposed',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS hr_roles_slug_uq ON hr_roles (slug);
CREATE TABLE IF NOT EXISTS hr_applications (
  id BIGSERIAL PRIMARY KEY,
  role_id BIGINT REFERENCES hr_roles(id) ON DELETE SET NULL,
  name TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT 'desk',
  cv_text TEXT NOT NULL DEFAULT '',
  score INT NOT NULL DEFAULT 0,
  recommendation TEXT NOT NULL DEFAULT 'unclear',
  reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL DEFAULT 'proposed',
  rationale TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS hr_applications_role_idx ON hr_applications (role_id, created_at DESC);
`)
	return err
}

func (r HrRole) KindLabel() string {
	if spec, ok := hr.Lookup(r.Template); ok {
		return spec.Label
	}
	if r.Kind != "" {
		return r.Kind
	}
	return r.Template
}

func (r HrRole) StatusStamp() string {
	switch r.Status {
	case "open":
		return "ok"
	case "closed":
		return "dark"
	default:
		return "brass"
	}
}

func (r HrRole) Proposed() bool { return r.Status == "proposed" }
func (r HrRole) Open() bool     { return r.Status == "open" }

func (a HrApplication) StatusStamp() string {
	switch a.Status {
	case "shortlisted":
		return "ok"
	case "rejected":
		return "dark"
	default:
		return "brass"
	}
}

func (a HrApplication) Proposed() bool { return a.Status == "proposed" }

func (s *Store) ListHrRoles(ctx context.Context) ([]HrRole, error) {
	rows, err := s.pool.Query(ctx, `
SELECT r.id, COALESCE(r.project_id, 0), COALESCE(p.name, ''), r.slug, r.title, r.kind, r.template, r.jd, r.status, r.created_at, r.updated_at
FROM hr_roles r
LEFT JOIN projects p ON p.id = r.project_id
ORDER BY r.created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HrRole
	for rows.Next() {
		role, err := scanHrRole(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, rows.Err()
}

func (s *Store) ListOpenHrRoles(ctx context.Context) ([]HrRole, error) {
	all, err := s.ListHrRoles(ctx)
	if err != nil {
		return nil, err
	}
	var out []HrRole
	for _, r := range all {
		if r.Status == "open" {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *Store) GetHrRole(ctx context.Context, id int64) (HrRole, error) {
	row := s.pool.QueryRow(ctx, `
SELECT r.id, COALESCE(r.project_id, 0), COALESCE(p.name, ''), r.slug, r.title, r.kind, r.template, r.jd, r.status, r.created_at, r.updated_at
FROM hr_roles r
LEFT JOIN projects p ON p.id = r.project_id
WHERE r.id = $1
`, id)
	return scanHrRole(row)
}

func (s *Store) GetHrRoleBySlug(ctx context.Context, slug string) (HrRole, error) {
	slug = normalizeSlug(slug)
	row := s.pool.QueryRow(ctx, `
SELECT r.id, COALESCE(r.project_id, 0), COALESCE(p.name, ''), r.slug, r.title, r.kind, r.template, r.jd, r.status, r.created_at, r.updated_at
FROM hr_roles r
LEFT JOIN projects p ON p.id = r.project_id
WHERE r.slug = $1
`, slug)
	return scanHrRole(row)
}

func scanHrRole(row projectScanner) (HrRole, error) {
	var r HrRole
	err := row.Scan(&r.ID, &r.ProjectID, &r.ProjectName, &r.Slug, &r.Title, &r.Kind, &r.Template, &r.JD, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func (s *Store) InsertHrRole(ctx context.Context, r HrRole) (HrRole, error) {
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return HrRole{}, errors.New("title is required")
	}
	r.Slug = normalizeSlug(r.Slug)
	if r.Slug == "" {
		r.Slug = normalizeSlug(r.Title)
	}
	if r.Slug == "" {
		return HrRole{}, errors.New("slug is required")
	}
	if spec, ok := hr.Lookup(r.Template); ok {
		r.Template = spec.ID
		if r.Kind == "" {
			r.Kind = spec.Kind
		}
	} else {
		r.Template = "contractor"
		if r.Kind == "" {
			r.Kind = "contractor"
		}
	}
	r.Status = "proposed"
	var project any
	if r.ProjectID > 0 {
		project = r.ProjectID
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
INSERT INTO hr_roles (project_id, slug, title, kind, template, jd, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id
`, project, r.Slug, r.Title, r.Kind, r.Template, r.JD, r.Status).Scan(&id)
	if err != nil {
		return HrRole{}, friendlyDBErr(err)
	}
	return s.GetHrRole(ctx, id)
}

func (s *Store) SetHrRoleStatus(ctx context.Context, id int64, status string) error {
	status = strings.TrimSpace(status)
	if status != "open" && status != "closed" && status != "paused" {
		return errors.New("status must be open, paused, or closed")
	}
	r, err := s.GetHrRole(ctx, id)
	if err != nil {
		return errors.New("role not found")
	}
	if r.Status != "proposed" && status == "open" {
		return errors.New("only a proposed JD can be opened")
	}
	_, err = s.pool.Exec(ctx, `UPDATE hr_roles SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func (s *Store) ListHrApplications(ctx context.Context, roleID int64) ([]HrApplication, error) {
	q := `
SELECT a.id, COALESCE(a.role_id, 0), COALESCE(r.title, ''), COALESCE(r.slug, ''),
       a.name, a.email, a.phone, a.source, a.cv_text, a.score, a.recommendation, a.reasons, a.status, a.rationale, a.created_at, a.updated_at
FROM hr_applications a
LEFT JOIN hr_roles r ON r.id = a.role_id
`
	args := []any{}
	if roleID > 0 {
		q += ` WHERE a.role_id = $1`
		args = append(args, roleID)
	}
	q += ` ORDER BY a.created_at DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HrApplication
	for rows.Next() {
		a, err := scanHrApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetHrApplication(ctx context.Context, id int64) (HrApplication, error) {
	row := s.pool.QueryRow(ctx, `
SELECT a.id, COALESCE(a.role_id, 0), COALESCE(r.title, ''), COALESCE(r.slug, ''),
       a.name, a.email, a.phone, a.source, a.cv_text, a.score, a.recommendation, a.reasons, a.status, a.rationale, a.created_at, a.updated_at
FROM hr_applications a
LEFT JOIN hr_roles r ON r.id = a.role_id
WHERE a.id = $1
`, id)
	return scanHrApplication(row)
}

func scanHrApplication(row projectScanner) (HrApplication, error) {
	var a HrApplication
	var raw []byte
	err := row.Scan(&a.ID, &a.RoleID, &a.RoleTitle, &a.RoleSlug, &a.Name, &a.Email, &a.Phone, &a.Source,
		&a.CVText, &a.Score, &a.Recommendation, &raw, &a.Status, &a.Rationale, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return HrApplication{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &a.Reasons)
	}
	return a, nil
}

func (s *Store) InsertHrApplication(ctx context.Context, a HrApplication) (HrApplication, error) {
	a.Status = "proposed"
	a.Name = strings.TrimSpace(a.Name)
	a.Email = strings.TrimSpace(strings.ToLower(a.Email))
	if a.Name == "" && a.Email == "" && strings.TrimSpace(a.CVText) == "" {
		return HrApplication{}, errors.New("application needs a name, email, or CV")
	}
	if a.Recommendation == "" {
		a.Recommendation = "unclear"
	}
	switch a.Recommendation {
	case "shortlist", "reject", "unclear":
	default:
		a.Recommendation = "unclear"
	}
	if a.Source == "" {
		a.Source = "desk"
	}
	if a.Reasons == nil {
		a.Reasons = []string{}
	}
	if a.Score < 0 {
		a.Score = 0
	}
	if a.Score > 100 {
		a.Score = 100
	}
	reasons, err := json.Marshal(a.Reasons)
	if err != nil {
		return HrApplication{}, err
	}
	var role any
	if a.RoleID > 0 {
		role = a.RoleID
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO hr_applications (role_id, name, email, phone, source, cv_text, score, recommendation, reasons, status, rationale)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11)
RETURNING id
`, role, a.Name, a.Email, strings.TrimSpace(a.Phone), a.Source, a.CVText, a.Score, a.Recommendation, reasons, a.Status, a.Rationale).Scan(&id)
	if err != nil {
		return HrApplication{}, err
	}
	return s.GetHrApplication(ctx, id)
}

func (s *Store) SetHrApplicationStatus(ctx context.Context, id int64, status string) error {
	status = strings.TrimSpace(status)
	if status != "shortlisted" && status != "rejected" {
		return errors.New("status must be shortlisted or rejected")
	}
	a, err := s.GetHrApplication(ctx, id)
	if err != nil {
		return errors.New("application not found")
	}
	if a.Status != "proposed" {
		return errors.New("only a proposed application can be stamped")
	}
	_, err = s.pool.Exec(ctx, `UPDATE hr_applications SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func roleFromStructured(sd map[string]any, projectID int64) (HrRole, error) {
	tmpl := strVal(sd["template"])
	kind := strVal(sd["kind"])
	if spec, ok := hr.Lookup(tmpl); ok {
		tmpl = spec.ID
		if kind == "" {
			kind = spec.Kind
		}
	}
	jd := strVal(sd["jd"])
	if jd == "" {
		return HrRole{}, errors.New("JD came back empty — role not filed")
	}
	title := strVal(sd["role_title"])
	if title == "" {
		title = strVal(sd["title"])
	}
	if title == "" {
		if spec, ok := hr.Lookup(tmpl); ok {
			title = spec.Label
		} else {
			title = "Open role"
		}
	}
	return HrRole{
		ProjectID: projectID,
		Slug:      strVal(sd["role_slug"]),
		Title:     title,
		Kind:      kind,
		Template:  tmpl,
		JD:        jd,
	}, nil
}

func applicationFromStructured(sd map[string]any, roleID int64, source string) (HrApplication, error) {
	cv := strVal(sd["cv_text"])
	name := strVal(sd["applicant_name"])
	if name == "" {
		name = strVal(sd["name"])
	}
	email := strVal(sd["applicant_email"])
	if email == "" {
		email = strVal(sd["email"])
	}
	rec := strings.ToLower(strVal(sd["recommendation"]))
	if rec == "" {
		rec = "unclear"
	}
	a := HrApplication{
		RoleID:         roleID,
		Name:           name,
		Email:          email,
		Phone:          firstNonEmpty(strVal(sd["applicant_phone"]), strVal(sd["phone"])),
		Source:         source,
		CVText:         cv,
		Score:          intVal(sd["score"]),
		Recommendation: rec,
		Reasons:        stringSlice(sd["reasons"]),
		Rationale:      strVal(sd["rationale"]),
	}
	if a.Rationale == "" {
		a.Rationale = strings.Join(a.Reasons, "; ")
	}
	if a.Name == "" && a.Email == "" && a.CVText == "" {
		return HrApplication{}, errors.New("worker returned no applicant")
	}
	return a, nil
}
