package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/departments/growth"
)

type GrowthDeck struct {
	ID           int64
	ProjectID    int64
	ProjectName  string
	Kind         string
	Audience     string
	Title        string
	Body         string
	Status       string
	Rationale    string
	PackToken    string
	Slides       []growth.DeckSlide
	CurrentSlide int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *Store) migrateGrowth(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS growth_decks (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  kind TEXT NOT NULL DEFAULT 'deck',
  audience TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'proposed',
  rationale TEXT NOT NULL DEFAULT '',
  pack_token TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS growth_decks_project_idx ON growth_decks (project_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS growth_decks_pack_token_idx ON growth_decks (pack_token) WHERE pack_token <> '';
ALTER TABLE growth_decks ADD COLUMN IF NOT EXISTS slides JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE growth_decks ADD COLUMN IF NOT EXISTS current_slide INT NOT NULL DEFAULT 1;
`)
	return err
}

const growthDeckSelect = `
SELECT d.id, d.project_id, p.name, d.kind, d.audience, d.title, d.body, d.status, d.rationale, COALESCE(d.pack_token, ''), COALESCE(d.slides, '[]'::jsonb), COALESCE(d.current_slide, 1), d.created_at, d.updated_at
FROM growth_decks d
JOIN projects p ON p.id = d.project_id
`

func (d GrowthDeck) KindLabel() string {
	if spec, ok := growth.Lookup(d.Kind); ok {
		return spec.Label
	}
	if d.Kind == "" {
		return "Deck pack"
	}
	return d.Kind
}

func (d GrowthDeck) StatusStamp() string {
	switch d.Status {
	case "approved":
		return "ok"
	case "rejected":
		return "dark"
	default:
		return "brass"
	}
}

func (d GrowthDeck) Proposed() bool { return d.Status == "proposed" }

func (s *Store) ListGrowthDecks(ctx context.Context, projectID int64) ([]GrowthDeck, error) {
	q := growthDeckSelect
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
	var out []GrowthDeck
	for rows.Next() {
		d, err := scanGrowthDeck(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetGrowthDeck(ctx context.Context, id int64) (GrowthDeck, error) {
	row := s.pool.QueryRow(ctx, growthDeckSelect+` WHERE d.id = $1`, id)
	return scanGrowthDeck(row)
}

func (s *Store) GetGrowthDeckByPackToken(ctx context.Context, token string) (GrowthDeck, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return GrowthDeck{}, errors.New("pack not found")
	}
	row := s.pool.QueryRow(ctx, growthDeckSelect+` WHERE d.pack_token = $1 AND d.status <> 'rejected'`, token)
	return scanGrowthDeck(row)
}

func scanGrowthDeck(row projectScanner) (GrowthDeck, error) {
	var d GrowthDeck
	var slides []byte
	err := row.Scan(&d.ID, &d.ProjectID, &d.ProjectName, &d.Kind, &d.Audience, &d.Title, &d.Body, &d.Status, &d.Rationale, &d.PackToken, &slides, &d.CurrentSlide, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return GrowthDeck{}, err
	}
	if len(slides) > 0 {
		var raw any
		if json.Unmarshal(slides, &raw) == nil {
			d.Slides = growth.ParseSlides(raw)
		}
	}
	if d.CurrentSlide < 1 {
		d.CurrentSlide = 1
	}
	return d, nil
}

func (s *Store) InsertGrowthDeck(ctx context.Context, d GrowthDeck) (GrowthDeck, error) {
	if d.ProjectID <= 0 {
		return GrowthDeck{}, errors.New("project is required")
	}
	d.Status = "proposed"
	if _, ok := growth.Lookup(d.Kind); !ok {
		d.Kind = "session"
	}
	if d.Title == "" {
		d.Title = d.KindLabel()
	}
	if d.CurrentSlide < 1 {
		d.CurrentSlide = 1
	}
	if len(d.Slides) == 0 {
		d.Slides = growth.EmptySlides()
	}
	if strings.TrimSpace(d.Body) == "" {
		d.Body = growth.SlidesMarkdown(d.Slides)
	}
	raw, err := json.Marshal(d.Slides)
	if err != nil {
		return GrowthDeck{}, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO growth_decks (project_id, kind, audience, title, body, status, rationale, slides, current_slide)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
RETURNING id
`, d.ProjectID, d.Kind, strings.TrimSpace(d.Audience), d.Title, d.Body, d.Status, d.Rationale, raw, d.CurrentSlide).Scan(&id)
	if err != nil {
		return GrowthDeck{}, err
	}
	if _, err := s.MintGrowthPackToken(ctx, id); err != nil {
		return GrowthDeck{}, err
	}
	return s.GetGrowthDeck(ctx, id)
}

func (s *Store) SetGrowthDeckStatus(ctx context.Context, id int64, status string) error {
	status = strings.TrimSpace(status)
	if status != "approved" && status != "rejected" {
		return errors.New("status must be approved or rejected")
	}
	d, err := s.GetGrowthDeck(ctx, id)
	if err != nil {
		return errors.New("deck not found")
	}
	if d.Status != "proposed" {
		return errors.New("only a proposed deck can be stamped")
	}
	_, err = s.pool.Exec(ctx, `UPDATE growth_decks SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func (s *Store) MintGrowthPackToken(ctx context.Context, id int64) (string, error) {
	d, err := s.GetGrowthDeck(ctx, id)
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
	_, err = s.pool.Exec(ctx, `UPDATE growth_decks SET pack_token = $2, updated_at = now() WHERE id = $1 AND pack_token = ''`, id, token)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) UpdateGrowthDeckDraft(ctx context.Context, d GrowthDeck) (GrowthDeck, error) {
	if d.ID <= 0 {
		return GrowthDeck{}, errors.New("deck is required")
	}
	cur, err := s.GetGrowthDeck(ctx, d.ID)
	if err != nil {
		return GrowthDeck{}, err
	}
	if cur.Status != "proposed" {
		return GrowthDeck{}, errors.New("only a proposed deck can be edited")
	}
	if d.Audience == "" {
		d.Audience = cur.Audience
	}
	if d.Title == "" {
		d.Title = cur.Title
	}
	d.Slides = growth.MergeSlides(cur.Slides, d.Slides)
	if d.CurrentSlide < 1 {
		d.CurrentSlide = cur.CurrentSlide
	}
	d.CurrentSlide = growth.ClampSlide(d.CurrentSlide)
	d.Body = growth.SlidesMarkdown(d.Slides)
	raw, err := json.Marshal(d.Slides)
	if err != nil {
		return GrowthDeck{}, err
	}
	_, err = s.pool.Exec(ctx, `
UPDATE growth_decks SET audience=$2, title=$3, body=$4, rationale=$5, slides=$6::jsonb, current_slide=$7, updated_at=now()
WHERE id=$1 AND status='proposed'
`, d.ID, strings.TrimSpace(d.Audience), d.Title, d.Body, d.Rationale, raw, d.CurrentSlide)
	if err != nil {
		return GrowthDeck{}, err
	}
	return s.GetGrowthDeck(ctx, d.ID)
}

func (d GrowthDeck) SlideLabel() string {
	cur := d.Current()
	if cur.Title == "" {
		return "Slide " + strconv.Itoa(cur.N)
	}
	return "Slide " + strconv.Itoa(cur.N) + " · " + cur.Title
}

func (d GrowthDeck) Current() growth.DeckSlide {
	n := growth.ClampSlide(d.CurrentSlide)
	if n >= 1 && n <= len(d.Slides) {
		return d.Slides[n-1]
	}
	title := ""
	if n >= 1 && n <= 10 {
		title = growth.KawasakiSlides()[n-1].Title
	}
	return growth.DeckSlide{N: n, Title: title, Status: "draft"}
}
