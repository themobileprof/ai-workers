package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/departments/seo"
)

type SeoBrief struct {
	ID          int64
	ProjectID   int64
	ProjectName string
	Kind        string
	URL         string
	Keyword     string
	Title       string
	Meta        string
	H1          string
	Body        string
	Flags       []string
	Status      string
	Rationale   string
	PackToken   string
	Inventory   []keywordPhrase
	PlanItems   []seoPlanItem
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type seoPlanItem struct {
	ProjectID  int64   `json:"project_id"`
	AssigneeID int64   `json:"assignee_id"`
	Title      string  `json:"title"`
	Hours      float64 `json:"hours"`
	Due        string  `json:"due"`
	Kind       string  `json:"kind"`
}

func (s *Store) migrateSEO(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS seo_briefs (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  kind TEXT NOT NULL DEFAULT 'page',
  url TEXT NOT NULL DEFAULT '',
  keyword TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  meta TEXT NOT NULL DEFAULT '',
  h1 TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  flags JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL DEFAULT 'proposed',
  rationale TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS seo_briefs_project_idx ON seo_briefs (project_id, created_at DESC);
ALTER TABLE seo_briefs ADD COLUMN IF NOT EXISTS pr_url TEXT NOT NULL DEFAULT '';
ALTER TABLE seo_briefs ADD COLUMN IF NOT EXISTS pack_token TEXT NOT NULL DEFAULT '';
ALTER TABLE seo_briefs ADD COLUMN IF NOT EXISTS inventory JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE seo_briefs ADD COLUMN IF NOT EXISTS plan_items JSONB NOT NULL DEFAULT '[]'::jsonb;
CREATE UNIQUE INDEX IF NOT EXISTS seo_briefs_pack_token_idx ON seo_briefs (pack_token) WHERE pack_token <> '';
CREATE TABLE IF NOT EXISTS seo_ga4 (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  property_id TEXT NOT NULL DEFAULT '',
  snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}

func (d SeoBrief) KindLabel() string {
	if spec, ok := seo.Lookup(d.Kind); ok {
		return spec.Label
	}
	if d.Kind == "" {
		return "Page brief"
	}
	return d.Kind
}

func (d SeoBrief) StatusStamp() string {
	switch d.Status {
	case "approved":
		return "ok"
	case "rejected":
		return "dark"
	default:
		return "brass"
	}
}

func (d SeoBrief) Proposed() bool { return d.Status == "proposed" }

func (s *Store) ListSeoBriefs(ctx context.Context, projectID int64) ([]SeoBrief, error) {
	q := `
SELECT d.id, d.project_id, p.name, d.kind, d.url, d.keyword, d.title, d.meta, d.h1, d.body, d.flags, d.status, d.rationale, COALESCE(d.pack_token, ''), COALESCE(d.inventory, '[]'::jsonb), COALESCE(d.plan_items, '[]'::jsonb), d.created_at, d.updated_at
FROM seo_briefs d
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
	var out []SeoBrief
	for rows.Next() {
		d, err := scanSeoBrief(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetSeoBrief(ctx context.Context, id int64) (SeoBrief, error) {
	row := s.pool.QueryRow(ctx, `
SELECT d.id, d.project_id, p.name, d.kind, d.url, d.keyword, d.title, d.meta, d.h1, d.body, d.flags, d.status, d.rationale, COALESCE(d.pack_token, ''), COALESCE(d.inventory, '[]'::jsonb), COALESCE(d.plan_items, '[]'::jsonb), d.created_at, d.updated_at
FROM seo_briefs d
JOIN projects p ON p.id = d.project_id
WHERE d.id = $1
`, id)
	return scanSeoBrief(row)
}

func (s *Store) GetSeoBriefByPackToken(ctx context.Context, token string) (SeoBrief, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return SeoBrief{}, errors.New("pack not found")
	}
	row := s.pool.QueryRow(ctx, `
SELECT d.id, d.project_id, p.name, d.kind, d.url, d.keyword, d.title, d.meta, d.h1, d.body, d.flags, d.status, d.rationale, COALESCE(d.pack_token, ''), COALESCE(d.inventory, '[]'::jsonb), COALESCE(d.plan_items, '[]'::jsonb), d.created_at, d.updated_at
FROM seo_briefs d
JOIN projects p ON p.id = d.project_id
WHERE d.pack_token = $1 AND d.status <> 'rejected'
`, token)
	return scanSeoBrief(row)
}

func scanSeoBrief(row projectScanner) (SeoBrief, error) {
	var d SeoBrief
	var raw []byte
	var inv []byte
	var plan []byte
	err := row.Scan(&d.ID, &d.ProjectID, &d.ProjectName, &d.Kind, &d.URL, &d.Keyword, &d.Title, &d.Meta, &d.H1, &d.Body, &raw, &d.Status, &d.Rationale, &d.PackToken, &inv, &plan, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return SeoBrief{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &d.Flags)
	}
	if len(inv) > 0 {
		_ = json.Unmarshal(inv, &d.Inventory)
	}
	if len(plan) > 0 {
		_ = json.Unmarshal(plan, &d.PlanItems)
	}
	return d, nil
}

func (s *Store) InsertSeoBrief(ctx context.Context, d SeoBrief) (SeoBrief, error) {
	if d.ProjectID <= 0 {
		return SeoBrief{}, errors.New("project is required")
	}
	d.Status = "proposed"
	if _, ok := seo.Lookup(d.Kind); !ok {
		d.Kind = "onsite"
	}
	if d.Title == "" {
		d.Title = d.KindLabel()
	}
	if d.Flags == nil {
		d.Flags = []string{}
	}
	flags, err := json.Marshal(d.Flags)
	if err != nil {
		return SeoBrief{}, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO seo_briefs (project_id, kind, url, keyword, title, meta, h1, body, flags, status, rationale)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11)
RETURNING id
`, d.ProjectID, d.Kind, strings.TrimSpace(d.URL), strings.TrimSpace(d.Keyword), d.Title, d.Meta, d.H1, d.Body, flags, d.Status, d.Rationale).Scan(&id)
	if err != nil {
		return SeoBrief{}, err
	}
	return s.GetSeoBrief(ctx, id)
}

func (s *Store) SetSeoStatus(ctx context.Context, id int64, status string) error {
	status = strings.TrimSpace(status)
	if status != "approved" && status != "rejected" {
		return errors.New("status must be approved or rejected")
	}
	d, err := s.GetSeoBrief(ctx, id)
	if err != nil {
		return errors.New("brief not found")
	}
	if d.Status != "proposed" {
		return errors.New("only a proposed brief can be stamped")
	}
	_, err = s.pool.Exec(ctx, `UPDATE seo_briefs SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func (s *Store) SetSeoInventory(ctx context.Context, id int64, inv []keywordPhrase) error {
	if inv == nil {
		inv = []keywordPhrase{}
	}
	raw, err := json.Marshal(inv)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE seo_briefs SET inventory = $2::jsonb, updated_at = now() WHERE id = $1`, id, raw)
	return err
}

func (s *Store) SetSeoPlanItems(ctx context.Context, id int64, items []seoPlanItem) error {
	if items == nil {
		items = []seoPlanItem{}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE seo_briefs SET plan_items = $2::jsonb, updated_at = now() WHERE id = $1`, id, raw)
	return err
}

func (s *Store) MintSeoPackToken(ctx context.Context, id int64) (string, error) {
	d, err := s.GetSeoBrief(ctx, id)
	if err != nil {
		return "", err
	}
	if d.PackToken != "" {
		return d.PackToken, nil
	}
	token, err := newPackToken()
	if err != nil {
		return "", err
	}
	_, err = s.pool.Exec(ctx, `UPDATE seo_briefs SET pack_token = $2, updated_at = now() WHERE id = $1 AND pack_token = ''`, id, token)
	if err != nil {
		return "", err
	}
	return token, nil
}

func newPackToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func seoBriefFromStructured(sd map[string]any, projectID int64) (SeoBrief, error) {
	kind := strVal(sd["action"])
	if kind == "" {
		kind = strVal(sd["kind"])
	}
	if _, ok := seo.Lookup(kind); !ok {
		kind = "onsite"
	}
	d := SeoBrief{
		ProjectID: projectID,
		Kind:      kind,
		URL:       strVal(sd["url"]),
		Keyword:   strVal(sd["keyword"]),
		Title:     strVal(sd["title"]),
		Meta:      firstNonEmpty(strVal(sd["meta_description"]), strVal(sd["meta"])),
		H1:        strVal(sd["h1"]),
		Body:      strVal(sd["body"]),
		Flags:     stringSlice(sd["flags"]),
		Rationale: strVal(sd["rationale"]),
	}
	if d.Rationale == "" {
		d.Rationale = strings.Join(d.Flags, "; ")
	}
	d.PlanItems = parseSeoPlanItems(sd["items"], projectID)
	if extra := walkBodyFromStructured(sd, ""); extra != "" {
		if d.Body == "" {
			d.Body = extra
		} else if !strings.Contains(d.Body, extra) {
			d.Body = strings.TrimSpace(d.Body) + "\n\n" + extra
		}
	}
	if d.Title == "" && d.Body == "" && len(d.Flags) == 0 && len(d.PlanItems) == 0 && !walkHasContent(sd) {
		return SeoBrief{}, errors.New("worker returned no title, outline, flags, or tasks")
	}
	if d.Title == "" {
		d.Title = d.KindLabel()
	}
	return d, nil
}
