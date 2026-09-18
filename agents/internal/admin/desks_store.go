package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type DeskJob struct {
	ID        int64
	Desk      string
	Title     string
	Status    string
	Source    string
	Summary   string
	Payload   map[string]any
	RefKind   string
	RefID     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type JobMessage struct {
	ID         int64
	JobID      int64
	AuthorKind string
	UserID     int64
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

func (j DeskJob) Proposed() bool {
	return j.Status == "proposed" || j.Status == "open"
}

func (j DeskJob) StatusStamp() string {
	switch j.Status {
	case "accepted":
		return "ok"
	case "rejected", "closed":
		return "dark"
	default:
		return "brass"
	}
}

func (j DeskJob) DeskTitle() string {
	if d, ok := LookupDesk(j.Desk); ok {
		return d.Title
	}
	return j.Desk
}

func (m JobMessage) Handler() bool { return m.AuthorKind == "handler" }
func (m JobMessage) Worker() bool  { return m.AuthorKind == "worker" }

func (s *Store) migrateDesks(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS user_desks (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  desk TEXT NOT NULL,
  PRIMARY KEY (user_id, desk)
);
CREATE TABLE IF NOT EXISTS desk_jobs (
  id BIGSERIAL PRIMARY KEY,
  desk TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'proposed',
  source TEXT NOT NULL DEFAULT 'desk',
  summary TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  ref_kind TEXT NOT NULL DEFAULT '',
  ref_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS desk_jobs_desk_idx ON desk_jobs (desk, created_at DESC);
CREATE TABLE IF NOT EXISTS desk_job_messages (
  id BIGSERIAL PRIMARY KEY,
  job_id BIGINT NOT NULL REFERENCES desk_jobs(id) ON DELETE CASCADE,
  author_kind TEXT NOT NULL,
  user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  body TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS desk_job_messages_job_idx ON desk_job_messages (job_id, created_at);
`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO user_desks (user_id, desk)
SELECT id, 'accounts' FROM users WHERE role = 'accounts'
ON CONFLICT DO NOTHING
`)
	return err
}

func (s *Store) desksFor(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT desk FROM user_desks WHERE user_id = $1 ORDER BY desk`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) replaceDesks(ctx context.Context, userID int64, desks []string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM user_desks WHERE user_id = $1`, userID); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, slug := range desks {
		slug = strings.TrimSpace(slug)
		if _, ok := LookupDesk(slug); !ok || seen[slug] {
			continue
		}
		seen[slug] = true
		if _, err := s.pool.Exec(ctx, `INSERT INTO user_desks (user_id, desk) VALUES ($1, $2)`, userID, slug); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListDeskJobs(ctx context.Context, desk string, limit int) ([]DeskJob, error) {
	if limit <= 0 {
		limit = 40
	}
	q := `
SELECT id, desk, title, status, source, summary, payload, ref_kind, ref_id, created_at, updated_at
FROM desk_jobs
`
	args := []any{}
	if desk != "" {
		q += ` WHERE desk = $1 ORDER BY created_at DESC LIMIT $2`
		args = append(args, desk, limit)
	} else {
		q += ` ORDER BY created_at DESC LIMIT $1`
		args = append(args, limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeskJob
	for rows.Next() {
		j, err := scanDeskJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) GetDeskJob(ctx context.Context, id int64) (DeskJob, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, desk, title, status, source, summary, payload, ref_kind, ref_id, created_at, updated_at
FROM desk_jobs WHERE id = $1
`, id)
	return scanDeskJob(row)
}

func scanDeskJob(row projectScanner) (DeskJob, error) {
	var j DeskJob
	var raw []byte
	err := row.Scan(&j.ID, &j.Desk, &j.Title, &j.Status, &j.Source, &j.Summary, &raw, &j.RefKind, &j.RefID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return DeskJob{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &j.Payload)
	}
	if j.Payload == nil {
		j.Payload = map[string]any{}
	}
	return j, nil
}

func (s *Store) InsertDeskJob(ctx context.Context, j DeskJob) (DeskJob, error) {
	if _, ok := LookupDesk(j.Desk); !ok {
		return DeskJob{}, errors.New("unknown desk")
	}
	j.Title = strings.TrimSpace(j.Title)
	if j.Title == "" {
		j.Title = j.Desk + " job"
	}
	if j.Status == "" {
		j.Status = "proposed"
	}
	if j.Source == "" {
		j.Source = "desk"
	}
	if j.Payload == nil {
		j.Payload = map[string]any{}
	}
	payload, err := json.Marshal(j.Payload)
	if err != nil {
		return DeskJob{}, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO desk_jobs (desk, title, status, source, summary, payload, ref_kind, ref_id)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
RETURNING id
`, j.Desk, j.Title, j.Status, j.Source, strings.TrimSpace(j.Summary), payload, strings.TrimSpace(j.RefKind), j.RefID).Scan(&id)
	if err != nil {
		return DeskJob{}, err
	}
	return s.GetDeskJob(ctx, id)
}

func (s *Store) SetDeskJobStatus(ctx context.Context, id int64, status string) error {
	status = strings.TrimSpace(status)
	switch status {
	case "proposed", "open", "accepted", "rejected", "closed":
	default:
		return errors.New("unknown job status")
	}
	j, err := s.GetDeskJob(ctx, id)
	if err != nil {
		return errors.New("job not found")
	}
	if status == "accepted" || status == "rejected" {
		if j.Status != "proposed" && j.Status != "open" {
			return errors.New("only an open or proposed job can be stamped")
		}
	}
	_, err = s.pool.Exec(ctx, `UPDATE desk_jobs SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func (s *Store) StampJobsByRef(ctx context.Context, kind string, refID int64, status string) error {
	if kind == "" || refID <= 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
UPDATE desk_jobs SET status=$3, updated_at=now()
WHERE ref_kind=$1 AND ref_id=$2 AND status IN ('proposed', 'open')
`, kind, refID, status)
	return err
}

func (s *Store) ListJobMessages(ctx context.Context, jobID int64) ([]JobMessage, error) {
	rows, err := s.pool.Query(ctx, `
SELECT m.id, m.job_id, m.author_kind, COALESCE(m.user_id, 0), COALESCE(u.name, ''), m.body, m.created_at
FROM desk_job_messages m
LEFT JOIN users u ON u.id = m.user_id
WHERE m.job_id = $1
ORDER BY m.created_at ASC, m.id ASC
`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobMessage
	for rows.Next() {
		var m JobMessage
		if err := rows.Scan(&m.ID, &m.JobID, &m.AuthorKind, &m.UserID, &m.AuthorName, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		if m.AuthorKind == "worker" && m.AuthorName == "" {
			m.AuthorName = "worker"
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) InsertJobMessage(ctx context.Context, m JobMessage) (JobMessage, error) {
	m.Body = strings.TrimSpace(m.Body)
	if m.Body == "" {
		return JobMessage{}, errors.New("message is empty")
	}
	if m.AuthorKind != "handler" && m.AuthorKind != "worker" && m.AuthorKind != "system" {
		return JobMessage{}, errors.New("unknown author")
	}
	var uid any
	if m.UserID > 0 {
		uid = m.UserID
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
INSERT INTO desk_job_messages (job_id, author_kind, user_id, body)
VALUES ($1, $2, $3, $4)
RETURNING id
`, m.JobID, m.AuthorKind, uid, m.Body).Scan(&id)
	if err != nil {
		return JobMessage{}, err
	}
	_, _ = s.pool.Exec(ctx, `UPDATE desk_jobs SET updated_at=now() WHERE id=$1`, m.JobID)
	msgs, err := s.ListJobMessages(ctx, m.JobID)
	if err != nil {
		return JobMessage{}, err
	}
	for _, got := range msgs {
		if got.ID == id {
			return got, nil
		}
	}
	return JobMessage{ID: id, JobID: m.JobID, AuthorKind: m.AuthorKind, UserID: m.UserID, Body: m.Body}, nil
}

func jobFromWorker(desk, source, task string, sd map[string]any, output string) DeskJob {
	title := firstNonEmpty(strVal(sd["role_title"]), strVal(sd["title"]))
	title = firstNonEmpty(title, strVal(sd["task_type"]))
	title = firstNonEmpty(title, strings.TrimSpace(task))
	if len(title) > 120 {
		title = title[:117] + "…"
	}
	summary := strings.TrimSpace(output)
	if len(summary) > 800 {
		summary = summary[:797] + "…"
	}
	refKind, refID := "", int64(0)
	if id := int64Val(sd["legal_draft_id"]); id > 0 {
		refKind, refID = "legal_draft", id
	}
	if id := int64Val(sd["hr_role_id"]); id > 0 {
		refKind, refID = "hr_role", id
	}
	if id := int64Val(sd["hr_application_id"]); id > 0 {
		refKind, refID = "hr_application", id
	}
	if id := int64Val(sd["project_id"]); id > 0 && refKind == "" {
		refKind, refID = "project", id
	}
	return DeskJob{
		Desk:    desk,
		Title:   title,
		Status:  "proposed",
		Source:  source,
		Summary: summary,
		Payload: sd,
		RefKind: refKind,
		RefID:   refID,
	}
}

func shouldFileInboundJob(dept string, data map[string]any, sd map[string]any) bool {
	if strings.EqualFold(strVal(data["channel"]), "desk") {
		return false
	}
	if strings.EqualFold(strVal(data["action"]), "watch") {
		return false
	}
	task := strings.ToLower(strVal(sd["task_type"]))
	if task == "access_denied" {
		return false
	}
	if boolVal(sd["needs_human"]) || boolVal(sd["escalate_to_founder"]) {
		return true
	}
	switch dept {
	case "legal":
		return true
	case "hr":
		if strings.EqualFold(strVal(data["channel"]), "email") {
			return false
		}
		return true
	case "product-dev":
		return strVal(sd["suggested_journey"]) != "" || strVal(data["action"]) == "place_on_journey"
	default:
		return false
	}
}
