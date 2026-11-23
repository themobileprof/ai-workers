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
	ProjectSlug  string
	Kind         string
	Audience     string
	Title        string
	Body         string
	Status       string
	Rationale    string
	PackToken    string
	Slides       []growth.DeckSlide
	Ads          growth.AdsPack
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
ALTER TABLE growth_decks ADD COLUMN IF NOT EXISTS ads JSONB NOT NULL DEFAULT '{}'::jsonb;
`)
	return err
}

const growthDeckSelect = `
SELECT d.id, d.project_id, p.name, p.slug, d.kind, d.audience, d.title, d.body, d.status, d.rationale, COALESCE(d.pack_token, ''), COALESCE(d.slides, '[]'::jsonb), COALESCE(d.current_slide, 1), COALESCE(d.ads, '{}'::jsonb), d.created_at, d.updated_at
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
	var ads []byte
	err := row.Scan(&d.ID, &d.ProjectID, &d.ProjectName, &d.ProjectSlug, &d.Kind, &d.Audience, &d.Title, &d.Body, &d.Status, &d.Rationale, &d.PackToken, &slides, &d.CurrentSlide, &ads, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return GrowthDeck{}, err
	}
	if len(slides) > 0 {
		var raw any
		if json.Unmarshal(slides, &raw) == nil {
			d.Slides = growth.ParseSlides(raw)
		}
	}
	if len(ads) > 0 {
		var raw any
		if json.Unmarshal(ads, &raw) == nil {
			d.Ads = growth.ParseAds(raw)
		}
	}
	if d.Kind == "ads" {
		d.CurrentSlide = growth.ClampAdsStep(d.Ads.Current)
		if d.Ads.Current < 1 {
			d.Ads.Current = d.CurrentSlide
		}
	} else if d.CurrentSlide < 1 {
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
	if d.Kind == "ads" {
		d.Ads = growth.ClipAds(d.Ads)
		if d.Ads.Current < 1 {
			d.Ads.Current = 1
		}
		d.CurrentSlide = d.Ads.Current
		d.Slides = nil
		if strings.TrimSpace(d.Body) == "" {
			d.Body = growth.AdsMarkdown(d.Ads, d.ProjectName, d.ProjectSlug)
		}
	} else {
		if d.CurrentSlide < 1 {
			d.CurrentSlide = 1
		}
		if len(d.Slides) == 0 {
			d.Slides = growth.EmptySlides()
		}
		if strings.TrimSpace(d.Body) == "" {
			d.Body = growth.SlidesMarkdown(d.Slides)
		}
	}
	raw, err := json.Marshal(d.Slides)
	if err != nil {
		return GrowthDeck{}, err
	}
	adsRaw, err := json.Marshal(d.Ads)
	if err != nil {
		return GrowthDeck{}, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO growth_decks (project_id, kind, audience, title, body, status, rationale, slides, current_slide, ads)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10::jsonb)
RETURNING id
`, d.ProjectID, d.Kind, strings.TrimSpace(d.Audience), d.Title, d.Body, d.Status, d.Rationale, raw, d.CurrentSlide, adsRaw).Scan(&id)
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
	if cur.Kind == "ads" || d.Kind == "ads" {
		d.Kind = "ads"
		d.Ads = growth.MergeAds(cur.Ads, d.Ads)
		d.CurrentSlide = d.Ads.Current
		d.Slides = cur.Slides
		d.Body = growth.AdsMarkdown(d.Ads, firstNonEmpty(d.ProjectName, cur.ProjectName), firstNonEmpty(d.ProjectSlug, cur.ProjectSlug))
	} else {
		d.Slides = growth.MergeSlides(cur.Slides, d.Slides)
		if d.CurrentSlide < 1 {
			d.CurrentSlide = cur.CurrentSlide
		}
		d.CurrentSlide = growth.ClampSlide(d.CurrentSlide)
		d.Body = growth.SlidesMarkdown(d.Slides)
	}
	raw, err := json.Marshal(d.Slides)
	if err != nil {
		return GrowthDeck{}, err
	}
	adsRaw, err := json.Marshal(d.Ads)
	if err != nil {
		return GrowthDeck{}, err
	}
	_, err = s.pool.Exec(ctx, `
UPDATE growth_decks SET audience=$2, title=$3, body=$4, rationale=$5, slides=$6::jsonb, current_slide=$7, ads=$8::jsonb, updated_at=now()
WHERE id=$1 AND status='proposed'
`, d.ID, strings.TrimSpace(d.Audience), d.Title, d.Body, d.Rationale, raw, d.CurrentSlide, adsRaw)
	if err != nil {
		return GrowthDeck{}, err
	}
	return s.GetGrowthDeck(ctx, d.ID)
}

type AdsPlacementRow struct {
	DeckID     int64    `json:"deck_id"`
	Channel    string   `json:"channel"`
	ID         string   `json:"id"`
	CustomerID string   `json:"customer_id,omitempty"`
	CampaignID string   `json:"campaign_id"`
	At         string   `json:"at,omitempty"`
	AgeDays    float64  `json:"age_days"`
	Dropped    []string `json:"dropped,omitempty"`
}

func (s *Store) ListAdsPlacements(ctx context.Context) ([]AdsPlacementRow, error) {
	decks, err := s.ListGrowthDecks(ctx, 0)
	if err != nil {
		return nil, err
	}
	var out []AdsPlacementRow
	for _, d := range decks {
		if d.Kind != "ads" || !d.Approved() {
			continue
		}
		for _, pl := range d.Ads.Placements {
			if strings.TrimSpace(pl.ID) == "" {
				continue
			}
			cust, camp := growth.AdsCampaignIDs(pl.Channel, pl.ID)
			if camp == "" {
				continue
			}
			out = append(out, AdsPlacementRow{
				DeckID: d.ID, Channel: pl.Channel, ID: pl.ID,
				CustomerID: cust, CampaignID: camp, At: pl.At,
				AgeDays: growth.AdsAgeDays(pl.At), Dropped: pl.Dropped,
			})
		}
	}
	return out, nil
}

func (s *Store) RecordAdsPrune(ctx context.Context, channel, id, note string, dropped []string) error {
	channel = strings.ToLower(strings.TrimSpace(channel))
	id = strings.TrimSpace(id)
	note = strings.TrimSpace(note)
	if channel == "" || id == "" || note == "" {
		return nil
	}
	decks, err := s.ListGrowthDecks(ctx, 0)
	if err != nil {
		return err
	}
	for _, d := range decks {
		if d.Kind != "ads" {
			continue
		}
		found := false
		for _, pl := range d.Ads.Placements {
			if pl.Channel == channel && pl.ID == id {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		d.Ads = d.Ads.WithPrune(channel, note, dropped)
		if _, err := s.RecordAdsPlacement(ctx, d); err != nil {
			return err
		}
		_ = s.NoteEpisodeOutcome(ctx, "growth_deck", d.ID, note)
		return nil
	}
	return nil
}

func (s *Store) RecordAdsPlacement(ctx context.Context, d GrowthDeck) (GrowthDeck, error) {
	if d.ID <= 0 || d.Kind != "ads" {
		return GrowthDeck{}, errors.New("ads pack is required")
	}
	d.Ads = growth.ClipAds(d.Ads)
	slug := strings.TrimSpace(d.ProjectSlug)
	name := strings.TrimSpace(d.ProjectName)
	if p, err := s.GetProject(ctx, d.ProjectID); err == nil {
		if slug == "" {
			slug = p.Slug
		}
		if name == "" {
			name = p.Name
		}
	}
	d.Body = growth.AdsMarkdown(d.Ads, name, slug)
	adsRaw, err := json.Marshal(d.Ads)
	if err != nil {
		return GrowthDeck{}, err
	}
	_, err = s.pool.Exec(ctx, `
UPDATE growth_decks SET ads=$2::jsonb, body=$3, updated_at=now()
WHERE id=$1 AND kind='ads'
`, d.ID, adsRaw, d.Body)
	if err != nil {
		return GrowthDeck{}, err
	}
	return s.GetGrowthDeck(ctx, d.ID)
}

func (d GrowthDeck) SlideLabel() string {
	if d.Kind == "ads" {
		return d.AdsStepLabel()
	}
	cur := d.Current()
	if cur.Title == "" {
		return "Slide " + strconv.Itoa(cur.N)
	}
	return "Slide " + strconv.Itoa(cur.N) + " · " + cur.Title
}

func (d GrowthDeck) AdsStepLabel() string {
	n := growth.ClampAdsStep(d.Ads.Current)
	for _, s := range growth.AdsSteps() {
		if s.N == n {
			return "Step " + strconv.Itoa(n) + " · " + s.Title
		}
	}
	return "Step " + strconv.Itoa(n)
}

func (d GrowthDeck) IsAds() bool { return d.Kind == "ads" }

func (d GrowthDeck) Approved() bool { return d.Status == "approved" }

func (d GrowthDeck) CanPlaceAds() bool {
	return d.IsAds() && d.Approved() && d.Ads.PlaceReady() == ""
}

func (d GrowthDeck) AdsWalk() []growth.AdsWalkStep { return d.Ads.Walk() }

func (d GrowthDeck) AdsManualGoogle() []growth.AdsManualStep {
	return growth.AdsManualGoogle(d.Ads, d.ProjectName, d.ProjectSlug)
}

func (d GrowthDeck) AdsManualMeta() []growth.AdsManualStep {
	return growth.AdsManualMeta(d.Ads, d.ProjectName, d.ProjectSlug)
}

func (d GrowthDeck) AdsReturnPath() string {
	if d.ProjectID <= 0 {
		return "/admin/growth"
	}
	return "/admin/projects/" + strconv.FormatInt(d.ProjectID, 10) + "?tab=ads"
}

func (d GrowthDeck) AdsBookedHow(channel string) string { return d.Ads.BookedHow(channel) }

func (d GrowthDeck) AdsManualCampaignID(channel string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	for _, pl := range d.Ads.Placements {
		if pl.Channel != channel || pl.Mode != "manual" {
			continue
		}
		if pl.ID == "" || pl.ID == "manual" {
			return ""
		}
		return pl.ID
	}
	return ""
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
