package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type CRMContact struct {
	ID           int64
	ProjectID    int64
	ProjectName  string
	Name         string
	Company      string
	Email        string
	Phone        string
	Stage        string
	Offer        string
	NextAction   string
	NextActionBy string
	Notes        string
	Source       string
	LastChannel  string
	LastAt       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CRMEvent struct {
	ID         int64
	ContactID  int64
	Channel    string
	Direction  string
	Summary    string
	Body       string
	HappenedAt time.Time
}

func (e CRMEvent) WhenLabel() string {
	if e.HappenedAt.IsZero() {
		return ""
	}
	return e.HappenedAt.In(lagosLocation()).Format("2 Jan 2006 15:04")
}

func (e CRMEvent) KindLabel() string {
	ch := e.Channel
	if e.Direction != "" && e.Direction != "note" {
		return ch + " · " + e.Direction
	}
	return ch
}

func (c CRMContact) Open() bool {
	switch c.Stage {
	case "won", "lost":
		return false
	default:
		return true
	}
}

func (c CRMContact) StatusStamp() string {
	switch c.Stage {
	case "won":
		return "ok"
	case "lost":
		return "dark"
	case "proposal", "qualified":
		return "brass"
	default:
		return "brass"
	}
}

func crmStage(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "qualified":
		return "qualified"
	case "proposal":
		return "proposal"
	case "won":
		return "won"
	case "lost":
		return "lost"
	case "nurture":
		return "nurture"
	default:
		return "new"
	}
}

func crmChannel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "email", "mail":
		return "email"
	case "whatsapp", "wa":
		return "whatsapp"
	case "call", "phone", "voice":
		return "call"
	case "telegram", "ops":
		return "telegram"
	case "stage":
		return "stage"
	default:
		return "desk"
	}
}

func crmDirection(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "in", "inbound":
		return "in"
	case "out", "outbound":
		return "out"
	default:
		return "note"
	}
}

func clipCRM(s string, n int) string {
	s = strings.TrimSpace(s)
	if n > 0 && len(s) > n {
		return s[:n]
	}
	return s
}

func (s *Store) migrateCRM(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS crm_contacts (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  company TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  stage TEXT NOT NULL DEFAULT 'new',
  next_action TEXT NOT NULL DEFAULT '',
  next_action_by DATE,
  notes TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS crm_contacts_email_idx ON crm_contacts (email) WHERE email <> '';
CREATE INDEX IF NOT EXISTS crm_contacts_phone_idx ON crm_contacts (phone) WHERE phone <> '';
CREATE INDEX IF NOT EXISTS crm_contacts_open_idx ON crm_contacts (next_action_by) WHERE stage NOT IN ('won', 'lost');
ALTER TABLE crm_contacts ADD COLUMN IF NOT EXISTS offer TEXT NOT NULL DEFAULT '';
ALTER TABLE crm_contacts ADD COLUMN IF NOT EXISTS last_channel TEXT NOT NULL DEFAULT '';
ALTER TABLE crm_contacts ADD COLUMN IF NOT EXISTS last_at TIMESTAMPTZ;
CREATE TABLE IF NOT EXISTS crm_events (
  id BIGSERIAL PRIMARY KEY,
  contact_id BIGINT NOT NULL REFERENCES crm_contacts(id) ON DELETE CASCADE,
  channel TEXT NOT NULL DEFAULT 'desk',
  direction TEXT NOT NULL DEFAULT 'note',
  summary TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  happened_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS crm_events_contact_idx ON crm_events (contact_id, happened_at DESC);
`)
	return err
}

func scanCRMContact(scan func(...any) error) (CRMContact, error) {
	var c CRMContact
	err := scan(&c.ID, &c.ProjectID, &c.ProjectName, &c.Name, &c.Company, &c.Email, &c.Phone,
		&c.Stage, &c.Offer, &c.NextAction, &c.NextActionBy, &c.Notes, &c.Source, &c.LastChannel, &c.LastAt,
		&c.CreatedAt, &c.UpdatedAt)
	return c, err
}

const crmSelect = `
SELECT c.id, COALESCE(c.project_id, 0), COALESCE(p.name, ''), c.name, c.company, c.email, c.phone,
       c.stage, c.offer, c.next_action, COALESCE(to_char(c.next_action_by, 'YYYY-MM-DD'), ''), c.notes, c.source,
       c.last_channel, COALESCE(to_char(c.last_at AT TIME ZONE 'Africa/Lagos', 'YYYY-MM-DD HH24:MI'), ''),
       c.created_at, c.updated_at
FROM crm_contacts c
LEFT JOIN projects p ON p.id = c.project_id
`

func (s *Store) ListCRMContacts(ctx context.Context) ([]CRMContact, error) {
	rows, err := s.pool.Query(ctx, crmSelect+`
ORDER BY CASE WHEN c.stage IN ('won','lost') THEN 1 ELSE 0 END,
         c.next_action_by NULLS LAST, c.updated_at DESC, c.id DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CRMContact
	for rows.Next() {
		c, err := scanCRMContact(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CountOpenCRM(ctx context.Context) int {
	var n int
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM crm_contacts WHERE stage NOT IN ('won','lost')`).Scan(&n)
	return n
}

func (s *Store) LookupCRMContact(ctx context.Context, email, phone, name string) (CRMContact, bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	phone = normalizePhone(phone)
	name = strings.TrimSpace(name)
	var c CRMContact
	var err error
	if email != "" {
		c, err = scanCRMContact(s.pool.QueryRow(ctx, crmSelect+`WHERE lower(c.email) = $1 LIMIT 1`, email).Scan)
		if err == nil {
			return c, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return CRMContact{}, false, err
		}
	}
	if phone != "" {
		c, err = scanCRMContact(s.pool.QueryRow(ctx, crmSelect+`WHERE c.phone = $1 LIMIT 1`, phone).Scan)
		if err == nil {
			return c, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return CRMContact{}, false, err
		}
	}
	if name != "" {
		c, err = scanCRMContact(s.pool.QueryRow(ctx, crmSelect+`WHERE lower(c.name) = lower($1) LIMIT 1`, name).Scan)
		if err == nil {
			return c, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return CRMContact{}, false, err
		}
		c, err = scanCRMContact(s.pool.QueryRow(ctx, crmSelect+`WHERE lower(c.company) = lower($1) LIMIT 1`, name).Scan)
		if err == nil {
			return c, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return CRMContact{}, false, err
		}
		if len(name) >= 2 {
			pat := likeContains(name)
			c, err = scanCRMContact(s.pool.QueryRow(ctx, crmSelect+`
WHERE (lower(c.name) LIKE lower($1) ESCAPE '\' OR lower(c.company) LIKE lower($1) ESCAPE '\')
ORDER BY CASE WHEN c.stage IN ('won','lost') THEN 1 ELSE 0 END, c.updated_at DESC, c.id DESC
LIMIT 1`, pat).Scan)
			if err == nil {
				return c, true, nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return CRMContact{}, false, err
			}
		}
	}
	return CRMContact{}, false, nil
}

func (s *Store) ListDueCRMContacts(ctx context.Context, today time.Time) ([]CRMContact, error) {
	day := today.In(lagosLocation()).Format("2006-01-02")
	rows, err := s.pool.Query(ctx, crmSelect+`
WHERE c.stage NOT IN ('won','lost')
  AND c.next_action_by IS NOT NULL
  AND c.next_action_by <= $1::date
ORDER BY c.next_action_by ASC, c.id ASC
`, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CRMContact
	for rows.Next() {
		c, err := scanCRMContact(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCRMContact(ctx context.Context, id int64) (CRMContact, error) {
	c, err := scanCRMContact(s.pool.QueryRow(ctx, crmSelect+`WHERE c.id = $1`, id).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CRMContact{}, errors.New("contact not found")
		}
		return CRMContact{}, err
	}
	return c, nil
}

func (s *Store) UpsertCRMContact(ctx context.Context, in CRMContact) (CRMContact, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Company = strings.TrimSpace(in.Company)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = normalizePhone(in.Phone)
	in.Notes = strings.TrimSpace(in.Notes)
	in.NextAction = strings.TrimSpace(in.NextAction)
	in.NextActionBy = strings.TrimSpace(in.NextActionBy)
	in.Source = strings.TrimSpace(in.Source)
	in.Offer = clipCRM(in.Offer, 200)
	in.Stage = crmStage(in.Stage)
	if in.Name == "" {
		if in.Email != "" {
			in.Name = strings.Split(in.Email, "@")[0]
		} else if in.Phone != "" {
			in.Name = "WhatsApp " + in.Phone
		}
	}
	if in.Name == "" {
		return CRMContact{}, errors.New("name, email, or phone is required")
	}
	hit, ok, err := s.LookupCRMContact(ctx, in.Email, in.Phone, in.Name)
	if err != nil {
		return CRMContact{}, err
	}
	var due any
	if in.NextActionBy != "" {
		due = in.NextActionBy
	}
	var project any
	if in.ProjectID > 0 {
		project = in.ProjectID
	}
	if ok {
		notes := hit.Notes
		if in.Notes != "" && in.Notes != hit.Notes {
			if notes != "" {
				notes = strings.TrimSpace(notes + "\n" + in.Notes)
			} else {
				notes = in.Notes
			}
			if len(notes) > 4000 {
				notes = notes[len(notes)-4000:]
			}
		}
		stage := hit.Stage
		if in.Stage != "" && in.Stage != "new" {
			stage = in.Stage
		}
		next := hit.NextAction
		if in.NextAction != "" {
			next = in.NextAction
		}
		nextBy := hit.NextActionBy
		if in.NextActionBy != "" {
			nextBy = in.NextActionBy
			due = nextBy
		} else if nextBy != "" {
			due = nextBy
		} else {
			due = nil
		}
		name := hit.Name
		if in.Name != "" && !strings.HasPrefix(in.Name, "WhatsApp ") {
			name = in.Name
		}
		company := hit.Company
		if in.Company != "" {
			company = in.Company
		}
		email := hit.Email
		if in.Email != "" {
			email = in.Email
		}
		phone := hit.Phone
		if in.Phone != "" {
			phone = in.Phone
		}
		offer := hit.Offer
		if in.Offer != "" {
			offer = in.Offer
		}
		if in.ProjectID > 0 {
			project = in.ProjectID
		} else if hit.ProjectID > 0 {
			project = hit.ProjectID
		}
		_, err = s.pool.Exec(ctx, `
UPDATE crm_contacts SET
  project_id = $2, name = $3, company = $4, email = $5, phone = $6,
  stage = $7, offer = $8, next_action = $9, next_action_by = $10, notes = $11, source = $12, updated_at = now()
WHERE id = $1
`, hit.ID, project, name, company, email, phone, stage, offer, next, due, notes, firstNonEmpty(in.Source, hit.Source))
		if err != nil {
			return CRMContact{}, friendlyDBErr(err)
		}
		hit.ProjectID = in.ProjectID
		if hit.ProjectID == 0 {
			hit.ProjectID = int64Val(project)
		}
		hit.Name, hit.Company, hit.Email, hit.Phone = name, company, email, phone
		hit.Stage, hit.Offer, hit.NextAction, hit.NextActionBy, hit.Notes = stage, offer, next, nextBy, notes
		return hit, nil
	}
	err = s.pool.QueryRow(ctx, `
INSERT INTO crm_contacts (project_id, name, company, email, phone, stage, offer, next_action, next_action_by, notes, source)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id, created_at, updated_at
`, project, in.Name, in.Company, in.Email, in.Phone, in.Stage, in.Offer, in.NextAction, due, in.Notes, in.Source).Scan(&in.ID, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return CRMContact{}, friendlyDBErr(err)
	}
	return in, nil
}

func (s *Store) SetCRMStage(ctx context.Context, id int64, stage string) error {
	stage = crmStage(stage)
	tag, err := s.pool.Exec(ctx, `UPDATE crm_contacts SET stage = $2, updated_at = now() WHERE id = $1`, id, stage)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("contact not found")
	}
	_ = s.InsertCRMEvent(ctx, CRMEvent{ContactID: id, Channel: "stage", Direction: "note", Summary: "Marked " + stage})
	return nil
}

func (s *Store) UpdateCRMContact(ctx context.Context, in CRMContact) (CRMContact, error) {
	if in.ID == 0 {
		return CRMContact{}, errors.New("contact not found")
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Company = strings.TrimSpace(in.Company)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = normalizePhone(in.Phone)
	in.Offer = clipCRM(in.Offer, 200)
	in.NextAction = strings.TrimSpace(in.NextAction)
	in.NextActionBy = strings.TrimSpace(in.NextActionBy)
	in.Notes = strings.TrimSpace(in.Notes)
	in.Stage = crmStage(in.Stage)
	if in.Name == "" {
		return CRMContact{}, errors.New("name is required")
	}
	var due any
	if in.NextActionBy != "" {
		due = in.NextActionBy
	}
	var project any
	if in.ProjectID > 0 {
		project = in.ProjectID
	}
	tag, err := s.pool.Exec(ctx, `
UPDATE crm_contacts SET
  project_id = $2, name = $3, company = $4, email = $5, phone = $6,
  stage = $7, offer = $8, next_action = $9, next_action_by = $10, notes = $11, updated_at = now()
WHERE id = $1
`, in.ID, project, in.Name, in.Company, in.Email, in.Phone, in.Stage, in.Offer, in.NextAction, due, in.Notes)
	if err != nil {
		return CRMContact{}, friendlyDBErr(err)
	}
	if tag.RowsAffected() == 0 {
		return CRMContact{}, errors.New("contact not found")
	}
	return s.GetCRMContact(ctx, in.ID)
}

func (s *Store) ListCRMEvents(ctx context.Context, contactID int64, limit int) ([]CRMEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, contact_id, channel, direction, summary, body, happened_at
FROM crm_events
WHERE contact_id = $1
ORDER BY happened_at DESC, id DESC
LIMIT $2
`, contactID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CRMEvent
	for rows.Next() {
		var e CRMEvent
		if err := rows.Scan(&e.ID, &e.ContactID, &e.Channel, &e.Direction, &e.Summary, &e.Body, &e.HappenedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) InsertCRMEvent(ctx context.Context, e CRMEvent) error {
	if e.ContactID == 0 {
		return errors.New("contact not found")
	}
	e.Channel = crmChannel(e.Channel)
	e.Direction = crmDirection(e.Direction)
	e.Summary = clipCRM(e.Summary, 400)
	e.Body = clipCRM(e.Body, 8000)
	if e.Summary == "" && e.Body == "" {
		return nil
	}
	if e.Summary == "" {
		e.Summary = e.Channel
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO crm_events (contact_id, channel, direction, summary, body)
VALUES ($1,$2,$3,$4,$5)
`, e.ContactID, e.Channel, e.Direction, e.Summary, e.Body)
	if err != nil {
		return friendlyDBErr(err)
	}
	_, _ = s.pool.Exec(ctx, `UPDATE crm_contacts SET last_channel = $2, last_at = now(), updated_at = now() WHERE id = $1`, e.ContactID, e.Channel)
	return nil
}
