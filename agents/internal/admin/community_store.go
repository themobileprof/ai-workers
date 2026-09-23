package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var errCommunityTitle = errors.New("title is required")

// CommunityMandate is the brief for one WhatsApp/Telegram room. The worker reads it; it does not invent a catalog.
type CommunityMandate struct {
	ID              int64
	Slug            string
	Title           string
	SiteURL         string
	Brief           string
	Catalog         string
	WhatsAppGroupID string
	TelegramChatID  string
	Active          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (m CommunityMandate) JSON() map[string]any {
	return map[string]any{
		"id":                m.ID,
		"slug":              m.Slug,
		"title":             m.Title,
		"site_url":          m.SiteURL,
		"brief":             m.Brief,
		"catalog":           m.Catalog,
		"whatsapp_group_id": m.WhatsAppGroupID,
		"telegram_chat_id":  m.TelegramChatID,
		"active":            m.Active,
	}
}

func (s *Store) migrateCommunity(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS community_mandates (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  site_url TEXT NOT NULL DEFAULT '',
  brief TEXT NOT NULL DEFAULT '',
  catalog TEXT NOT NULL DEFAULT '',
  whatsapp_group_id TEXT NOT NULL DEFAULT '',
  telegram_chat_id TEXT NOT NULL DEFAULT '',
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS community_mandates_wa_idx ON community_mandates (whatsapp_group_id) WHERE whatsapp_group_id <> '';
`)
	if err != nil {
		return err
	}
	if err := s.seedLMSMandate(ctx); err != nil {
		return err
	}
	return s.refreshLMSSeed(ctx)
}

func (s *Store) seedLMSMandate(ctx context.Context) error {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM community_mandates WHERE slug = 'lms'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO community_mandates (slug, title, site_url, brief, catalog, active)
VALUES ('lms', $1, $2, $3, $4, true)
`, lmsMandateTitle, lmsMandateSite, lmsMandateBrief, lmsMandateCatalog)
	return err
}

const lmsMandateTitle = "Academy LMS"
const lmsMandateSite = "https://lms.themobileprof.com"

func (s *Store) refreshLMSSeed(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
UPDATE community_mandates
SET brief = $1, catalog = $2, updated_at = now()
WHERE slug = 'lms' AND catalog LIKE '%Three tiers (do not invent a fourth)%'
`, lmsMandateBrief, lmsMandateCatalog)
	return err
}

const lmsMandateBrief = `You host the TheMobileProf Academy student WhatsApp group for https://lms.themobileprof.com.

Your job in this room:
- Welcome people. Ask which course they are on (Micro / Mini path / Professional — use the catalog titles) and which lesson.
- Once a week (n8n Monday 09:00 Lagos) introduce yourself: who you are, that unprefixed group chat already reaches you, /cm or /community to be sure, then ask what each person is working on.
- Encourage them to say one thing they learned, not a vibe. Recap and ask a follow-up that keeps the thread going.
- Answer questions you can from the catalog and from general study practice. You have course titles, descriptions, and (where listed) syllabus lines. You do not have lesson video or quiz text until a human wires an LMS token.
- If you do not know — login lock, missing payment, certificate not showing, a course not in the catalog, a complaint — say a human admin will pick it up. Set escalate_to_founder true. Do not invent a reset, a refund, or a course name.
- Do not pitch other TheMobileProf products. Do not take books, legal, or HR in this room.
- Stay quiet when members are just chatting (ok, lol, stickers, side talk) unless they ask a question, share progress, greet as new, or this is the weekly intro.`

const lmsMandateCatalog = `TheMobileProf Academy — mobile-first tech education. Login: https://lms.themobileprof.com
Public story: https://themobileprof.com
Course list is public at https://api.themobileprof.com/api/courses (and /api/mini-courses). Lesson bodies are not public.

Three product lines (students may say Mini when they mean a Micro):
1. Micro — afternoon courses, API format=micro.
   Termux / Linux: Termux Essentials: Your First Steps in Mobile Linux; Starting Termux; Linux Navigation; Navigating Linux; Package Management; Working in a Linux shell; Linux File Permissions; Advanced Shell Commands; Networking on Linux; Bash Scripts; Git Essentials; Git Remote & SSH; SSH & Remote Servers; tmux & Long Sessions; Troubleshoot Like a Dev; Clio: Your Terminal Assistant.
2. Mini (Path) — bundles of Micros, API /api/mini-courses:
   Introduction to Linux Terminal (Starting Termux, Linux Navigation, Package Management);
   Working in the Shell (Working in a Linux shell, Linux File Permissions, Advanced Shell Commands);
   Networking and Automation (Networking on Linux, Bash Scripts, Git Essentials, Git Remote & SSH).
3. Professional — Linux on Mobile; Web Dev Skills, All on Your Phone!; Security: Master Your Digital Fortress on the Go!

Facts you may use:
- Built for the phone you already carry. The desk is optional.
- HD video, assignments, certificates on the LMS.
- School: 3 Thorborn Avenue, Sabo, Yaba, Lagos.
- info@themobileprof.com · +234 803 395 4301 · WhatsApp business +234 707 524 0400.
- Free 2-day live training exists for selected youth NGOs; that is not a student discount in this group — escalate if someone asks to enrol an NGO.

Amend this catalog on the Community desk when the LMS adds or retires a course. Until a title is written here, you do not have it.`

func (s *Store) ListCommunityMandates(ctx context.Context) ([]CommunityMandate, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, slug, title, site_url, brief, catalog, whatsapp_group_id, telegram_chat_id, active, created_at, updated_at
FROM community_mandates ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommunityMandate
	for rows.Next() {
		m, err := scanCommunityMandate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetCommunityMandate(ctx context.Context, id int64) (CommunityMandate, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, slug, title, site_url, brief, catalog, whatsapp_group_id, telegram_chat_id, active, created_at, updated_at
FROM community_mandates WHERE id = $1`, id)
	return scanCommunityMandate(row)
}

func (s *Store) LookupCommunityMandate(ctx context.Context, groupID, telegramID string) (CommunityMandate, bool, error) {
	groupID = strings.TrimSpace(groupID)
	telegramID = strings.TrimSpace(telegramID)
	if groupID == "" && telegramID == "" {
		return CommunityMandate{}, false, nil
	}
	row := s.pool.QueryRow(ctx, `
SELECT id, slug, title, site_url, brief, catalog, whatsapp_group_id, telegram_chat_id, active, created_at, updated_at
FROM community_mandates
WHERE active AND (
  ($1 <> '' AND whatsapp_group_id = $1) OR
  ($2 <> '' AND telegram_chat_id = $2)
)
ORDER BY updated_at DESC
LIMIT 1`, groupID, telegramID)
	m, err := scanCommunityMandate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CommunityMandate{}, false, nil
		}
		return CommunityMandate{}, false, err
	}
	return m, true, nil
}

func (s *Store) UpsertCommunityMandate(ctx context.Context, m CommunityMandate) (CommunityMandate, error) {
	m.Slug = strings.TrimSpace(strings.ToLower(m.Slug))
	m.Title = strings.TrimSpace(m.Title)
	m.SiteURL = strings.TrimSpace(m.SiteURL)
	m.WhatsAppGroupID = strings.TrimSpace(m.WhatsAppGroupID)
	m.TelegramChatID = strings.TrimSpace(m.TelegramChatID)
	if m.Title == "" {
		return CommunityMandate{}, errCommunityTitle
	}
	if m.Slug == "" {
		m.Slug = communitySlug(m.Title)
	}
	if m.ID > 0 {
		_, err := s.pool.Exec(ctx, `
UPDATE community_mandates
SET slug=$2, title=$3, site_url=$4, brief=$5, catalog=$6, whatsapp_group_id=$7, telegram_chat_id=$8, active=$9, updated_at=now()
WHERE id=$1`, m.ID, m.Slug, m.Title, m.SiteURL, m.Brief, m.Catalog, m.WhatsAppGroupID, m.TelegramChatID, m.Active)
		if err != nil {
			return CommunityMandate{}, err
		}
		return s.GetCommunityMandate(ctx, m.ID)
	}
	row := s.pool.QueryRow(ctx, `
INSERT INTO community_mandates (slug, title, site_url, brief, catalog, whatsapp_group_id, telegram_chat_id, active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id`, m.Slug, m.Title, m.SiteURL, m.Brief, m.Catalog, m.WhatsAppGroupID, m.TelegramChatID, m.Active)
	if err := row.Scan(&m.ID); err != nil {
		return CommunityMandate{}, err
	}
	return s.GetCommunityMandate(ctx, m.ID)
}

func (s *Store) DeleteCommunityMandate(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM community_mandates WHERE id = $1`, id)
	return err
}

type mandateRow interface {
	Scan(dest ...any) error
}

func scanCommunityMandate(row mandateRow) (CommunityMandate, error) {
	var m CommunityMandate
	err := row.Scan(&m.ID, &m.Slug, &m.Title, &m.SiteURL, &m.Brief, &m.Catalog, &m.WhatsAppGroupID, &m.TelegramChatID, &m.Active, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func communitySlug(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '_' || r == '-' {
			return '-'
		}
		return -1
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
