package admin

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// DeskEpisode is one staff Ask → proposal → human act. Not Academy, not WhatsApp.
type DeskEpisode struct {
	ID          int64
	Desk        string
	ProjectSlug string
	TaskType    string
	Source      string
	Title       string
	PromptVer   string
	Model       string
	Input       map[string]any
	Proposal    map[string]any
	HumanAct    string
	Reason      string
	Outcome     string
	JobID       int64
	RefKind     string
	RefID       int64
	CreatedAt   time.Time
	StampedAt   time.Time
}

func (e DeskEpisode) Proposed() bool {
	return e.HumanAct == "" || e.HumanAct == "proposed"
}

func (e DeskEpisode) StatusStamp() string {
	switch e.HumanAct {
	case "accepted", "edited_accepted":
		return "ok"
	case "rejected":
		return "dark"
	default:
		return "brass"
	}
}

func (e DeskEpisode) ActLabel() string {
	switch e.HumanAct {
	case "accepted":
		return "accepted"
	case "edited_accepted":
		return "edited then accepted"
	case "rejected":
		return "turned away"
	default:
		return "proposed"
	}
}

func (s *Store) migrateEpisodes(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS desk_episodes (
  id BIGSERIAL PRIMARY KEY,
  desk TEXT NOT NULL,
  project_slug TEXT NOT NULL DEFAULT '',
  task_type TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  prompt_version TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  input JSONB NOT NULL DEFAULT '{}'::jsonb,
  proposal JSONB NOT NULL DEFAULT '{}'::jsonb,
  human_act TEXT NOT NULL DEFAULT 'proposed',
  reason TEXT NOT NULL DEFAULT '',
  outcome TEXT NOT NULL DEFAULT '',
  job_id BIGINT NOT NULL DEFAULT 0,
  ref_kind TEXT NOT NULL DEFAULT '',
  ref_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  stamped_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS desk_episodes_desk_idx ON desk_episodes (desk, project_slug, created_at DESC);
CREATE INDEX IF NOT EXISTS desk_episodes_job_idx ON desk_episodes (job_id) WHERE job_id > 0;
CREATE INDEX IF NOT EXISTS desk_episodes_ref_idx ON desk_episodes (ref_kind, ref_id) WHERE ref_id > 0;
`)
	if err != nil {
		return err
	}
	return s.purgeEpisodes(ctx)
}

func (s *Store) purgeEpisodes(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
DELETE FROM desk_episodes
WHERE (human_act = 'proposed' AND created_at < now() - interval '30 days')
   OR (human_act <> 'proposed' AND created_at < now() - interval '180 days')
`)
	return err
}

func (s *Store) InsertEpisode(ctx context.Context, e DeskEpisode) (DeskEpisode, error) {
	if s == nil || s.pool == nil {
		return DeskEpisode{}, nil
	}
	e.Desk = strings.TrimSpace(e.Desk)
	if e.Desk == "" {
		return DeskEpisode{}, nil
	}
	if e.HumanAct == "" {
		e.HumanAct = "proposed"
	}
	if e.Input == nil {
		e.Input = map[string]any{}
	}
	if e.Proposal == nil {
		e.Proposal = map[string]any{}
	}
	inRaw, err := json.Marshal(e.Input)
	if err != nil {
		return DeskEpisode{}, err
	}
	propRaw, err := json.Marshal(e.Proposal)
	if err != nil {
		return DeskEpisode{}, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO desk_episodes (
  desk, project_slug, task_type, source, title, prompt_version, model,
  input, proposal, human_act, reason, outcome, job_id, ref_kind, ref_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb,$10,$11,$12,$13,$14,$15)
RETURNING id
`, e.Desk, strings.TrimSpace(e.ProjectSlug), strings.TrimSpace(e.TaskType), strings.TrimSpace(e.Source),
		clipRunes(strings.TrimSpace(e.Title), 120), strings.TrimSpace(e.PromptVer), strings.TrimSpace(e.Model),
		inRaw, propRaw, e.HumanAct, clipRunes(e.Reason, 200), clipRunes(e.Outcome, 400),
		e.JobID, strings.TrimSpace(e.RefKind), e.RefID).Scan(&id)
	if err != nil {
		return DeskEpisode{}, err
	}
	e.ID = id
	e.CreatedAt = time.Now()
	return e, nil
}

func (s *Store) StampEpisodes(ctx context.Context, jobID int64, refKind string, refID int64, act, reason string) error {
	if s == nil || s.pool == nil {
		return nil
	}
	act = strings.TrimSpace(act)
	switch act {
	case "accepted", "rejected", "edited_accepted":
	default:
		return nil
	}
	reason = clipRunes(strings.TrimSpace(reason), 200)
	if jobID > 0 {
		_, err := s.pool.Exec(ctx, `
UPDATE desk_episodes
SET human_act=$2, reason=CASE WHEN $3 <> '' THEN $3 ELSE reason END, stamped_at=now()
WHERE job_id=$1 AND human_act='proposed'
`, jobID, act, reason)
		if err != nil {
			return err
		}
	}
	if refKind != "" && refID > 0 {
		_, err := s.pool.Exec(ctx, `
UPDATE desk_episodes
SET human_act=$3, reason=CASE WHEN $4 <> '' THEN $4 ELSE reason END, stamped_at=now()
WHERE ref_kind=$1 AND ref_id=$2 AND human_act='proposed'
`, refKind, refID, act, reason)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) NoteEpisodeOutcome(ctx context.Context, refKind string, refID int64, note string) error {
	if s == nil || s.pool == nil || refKind == "" || refID <= 0 {
		return nil
	}
	note = clipRunes(strings.TrimSpace(note), 400)
	if note == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
UPDATE desk_episodes
SET outcome=$3
WHERE ref_kind=$1 AND ref_id=$2 AND human_act IN ('accepted', 'edited_accepted')
`, refKind, refID, note)
	return err
}

func (s *Store) ListEpisodes(ctx context.Context, limit int) ([]DeskEpisode, error) {
	if s == nil || s.pool == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 400 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, desk, project_slug, task_type, source, title, prompt_version, model,
       input, proposal, human_act, reason, outcome, job_id, ref_kind, ref_id,
       created_at, COALESCE(stamped_at, TIMESTAMPTZ 'epoch')
FROM desk_episodes
ORDER BY created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeskEpisode
	for rows.Next() {
		e, err := scanEpisode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) ListEpisodeLessons(ctx context.Context, desk, slug string, acceptedN, rejectedN int) (accepted, rejected []DeskEpisode, err error) {
	if s == nil || s.pool == nil || strings.TrimSpace(desk) == "" {
		return nil, nil, nil
	}
	if acceptedN <= 0 {
		acceptedN = 3
	}
	if rejectedN <= 0 {
		rejectedN = 2
	}
	slug = strings.TrimSpace(slug)
	accepted, err = s.listEpisodeAct(ctx, desk, slug, []string{"accepted", "edited_accepted"}, acceptedN)
	if err != nil {
		return nil, nil, err
	}
	rejected, err = s.listEpisodeAct(ctx, desk, slug, []string{"rejected"}, rejectedN)
	return accepted, rejected, err
}

func (s *Store) listEpisodeAct(ctx context.Context, desk, slug string, acts []string, limit int) ([]DeskEpisode, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, desk, project_slug, task_type, source, title, prompt_version, model,
       input, proposal, human_act, reason, outcome, job_id, ref_kind, ref_id,
       created_at, COALESCE(stamped_at, TIMESTAMPTZ 'epoch')
FROM desk_episodes
WHERE desk=$1
  AND human_act = ANY($2)
  AND ($3 = '' OR project_slug = $3)
ORDER BY COALESCE(stamped_at, created_at) DESC
LIMIT $4
`, desk, acts, slug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeskEpisode
	for rows.Next() {
		e, err := scanEpisode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type episodeRow interface {
	Scan(dest ...any) error
}

func scanEpisode(row episodeRow) (DeskEpisode, error) {
	var e DeskEpisode
	var inRaw, propRaw []byte
	var stamped time.Time
	err := row.Scan(
		&e.ID, &e.Desk, &e.ProjectSlug, &e.TaskType, &e.Source, &e.Title, &e.PromptVer, &e.Model,
		&inRaw, &propRaw, &e.HumanAct, &e.Reason, &e.Outcome, &e.JobID, &e.RefKind, &e.RefID,
		&e.CreatedAt, &stamped,
	)
	if err != nil {
		return DeskEpisode{}, err
	}
	if !stamped.IsZero() && stamped.Year() > 1970 {
		e.StampedAt = stamped
	}
	e.Input = map[string]any{}
	e.Proposal = map[string]any{}
	_ = json.Unmarshal(inRaw, &e.Input)
	_ = json.Unmarshal(propRaw, &e.Proposal)
	return e, nil
}
