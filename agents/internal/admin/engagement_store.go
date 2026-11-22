package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	engageCommunity = "community"
	engageAcademy   = "academy"
	engageReferral  = "referral"
)

type EngageScore struct {
	TelegramID string
	Name       string
	Username   string
	Community  int
	Academy    int
	Referral   int
	Total      int
}

type RewardDay struct {
	Day          time.Time
	WinnerID     string
	WinnerName   string
	GroupChatID  string
	Stage        string
	Amount       int
	Phone        string
	Network      string
	RequestID    string
	OrderID      string
	ClubStatus   string
	CommunityPts int
	AcademyPts   int
	ReferralPts  int
	CheckedInAt  *time.Time
	AnnouncedAt  *time.Time
}

func (s *Store) migrateEngagement(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS community_engagement (
  id BIGSERIAL PRIMARY KEY,
  telegram_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  weight INT NOT NULL DEFAULT 1,
  day DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS community_engagement_day_idx ON community_engagement (day, telegram_id);
CREATE TABLE IF NOT EXISTS community_referrals (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL,
  referrer_telegram_id TEXT NOT NULL,
  referred_telegram_id TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS community_reward_days (
  day DATE PRIMARY KEY,
  winner_telegram_id TEXT NOT NULL DEFAULT '',
  winner_name TEXT NOT NULL DEFAULT '',
  group_chat_id TEXT NOT NULL DEFAULT '',
  stage TEXT NOT NULL DEFAULT 'none',
  amount INT NOT NULL DEFAULT 0,
  phone TEXT NOT NULL DEFAULT '',
  network TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  order_id TEXT NOT NULL DEFAULT '',
  club_status TEXT NOT NULL DEFAULT '',
  community_pts INT NOT NULL DEFAULT 0,
  academy_pts INT NOT NULL DEFAULT 0,
  referral_pts INT NOT NULL DEFAULT 0,
  announced_at TIMESTAMPTZ,
  checked_in_at TIMESTAMPTZ,
  sent_at TIMESTAMPTZ,
  verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS community_join_hits (
  id BIGSERIAL PRIMARY KEY,
  code TEXT NOT NULL DEFAULT '',
  landing TEXT NOT NULL DEFAULT '',
  referer TEXT NOT NULL DEFAULT '',
  utm_source TEXT NOT NULL DEFAULT '',
  utm_medium TEXT NOT NULL DEFAULT '',
  utm_campaign TEXT NOT NULL DEFAULT '',
  utm_content TEXT NOT NULL DEFAULT '',
  utm_term TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE community_reward_days ADD COLUMN IF NOT EXISTS checked_in_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS community_join_hits_code_idx ON community_join_hits (code, created_at DESC);
CREATE TABLE IF NOT EXISTS community_lms_signups (
  email TEXT PRIMARY KEY,
  code TEXT NOT NULL,
  referrer_telegram_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS community_referral_aliases (
  code TEXT PRIMARY KEY,
  telegram_id TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS community_referral_email_codes (
  email TEXT PRIMARY KEY,
  code TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}

type JoinHit struct {
	Code        string
	Landing     string
	Referer     string
	UTMSource   string
	UTMMedium   string
	UTMCampaign string
	UTMContent  string
	UTMTerm     string
	UserAgent   string
}

func clipText(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

func (s *Store) RecordJoinHit(ctx context.Context, h JoinHit) {
	if s == nil || s.pool == nil {
		return
	}
	code := strings.ToLower(strings.TrimSpace(h.Code))
	_, _ = s.pool.Exec(ctx, `
INSERT INTO community_join_hits (code, landing, referer, utm_source, utm_medium, utm_campaign, utm_content, utm_term, user_agent)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
`, code, clipText(h.Landing, 2000), clipText(h.Referer, 2000), clipText(h.UTMSource, 120), clipText(h.UTMMedium, 120), clipText(h.UTMCampaign, 120), clipText(h.UTMContent, 120), clipText(h.UTMTerm, 120), clipText(h.UserAgent, 400))
}

func (s *Store) ConfirmLMSReferral(ctx context.Context, code, email string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	email = strings.ToLower(strings.TrimSpace(email))
	if code == "" || email == "" {
		return false
	}
	ref, err := s.LookupCommunityStudentByCode(ctx, code)
	if err != nil || ref.TelegramID == "" {
		return false
	}
	tag, err := s.pool.Exec(ctx, `
INSERT INTO community_lms_signups (email, code, referrer_telegram_id)
VALUES ($1,$2,$3)
ON CONFLICT (email) DO NOTHING
`, email, code, ref.TelegramID)
	if err != nil || tag.RowsAffected() == 0 {
		return false
	}
	s.RecordEngagement(ctx, ref.TelegramID, engageReferral, 5, time.Now())
	return true
}

func (s *Store) RecordEngagement(ctx context.Context, telegramID, kind string, weight int, when time.Time) {
	telegramID = strings.TrimSpace(telegramID)
	kind = strings.ToLower(strings.TrimSpace(kind))
	if telegramID == "" || weight <= 0 {
		return
	}
	if kind != engageCommunity && kind != engageAcademy && kind != engageReferral {
		return
	}
	if when.IsZero() {
		when = time.Now()
	}
	day := when.In(lagosLocation()).Format("2006-01-02")
	_, _ = s.pool.Exec(ctx, `
INSERT INTO community_engagement (telegram_id, kind, weight, day)
VALUES ($1,$2,$3,$4::date)
`, telegramID, kind, weight, day)
}

func (s *Store) ConfirmReferral(ctx context.Context, code, referredID, referredName string, when time.Time) (CommunityStudent, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	referredID = strings.TrimSpace(referredID)
	if code == "" || referredID == "" {
		return CommunityStudent{}, false
	}
	ref, err := s.LookupCommunityStudentByCode(ctx, code)
	if err != nil || ref.TelegramID == "" || ref.TelegramID == referredID {
		return CommunityStudent{}, false
	}
	tag, err := s.pool.Exec(ctx, `
INSERT INTO community_referrals (code, referrer_telegram_id, referred_telegram_id)
VALUES ($1,$2,$3)
ON CONFLICT (referred_telegram_id) DO NOTHING
`, code, ref.TelegramID, referredID)
	if err != nil || tag.RowsAffected() == 0 {
		return CommunityStudent{}, false
	}
	s.RecordEngagement(ctx, ref.TelegramID, engageReferral, 5, when)
	return ref, true
}

func capInt(n, capN int) int {
	if n > capN {
		return capN
	}
	if n < 0 {
		return 0
	}
	return n
}

func (s *Store) RankEngagement(ctx context.Context, day time.Time) ([]EngageScore, error) {
	d := day.In(lagosLocation()).Format("2006-01-02")
	rows, err := s.pool.Query(ctx, `
SELECT e.telegram_id,
       COALESCE(MAX(s.name), ''),
       COALESCE(MAX(s.username), ''),
       COALESCE(SUM(CASE WHEN e.kind = 'community' THEN e.weight ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN e.kind = 'academy' THEN e.weight ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN e.kind = 'referral' THEN e.weight ELSE 0 END), 0)
FROM community_engagement e
LEFT JOIN community_students s ON s.telegram_id = e.telegram_id
WHERE e.day = $1::date
GROUP BY e.telegram_id
`, d)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EngageScore
	for rows.Next() {
		var sc EngageScore
		if err := rows.Scan(&sc.TelegramID, &sc.Name, &sc.Username, &sc.Community, &sc.Academy, &sc.Referral); err != nil {
			return nil, err
		}
		sc.Community = capInt(sc.Community, 8)
		sc.Academy = capInt(sc.Academy, 8)
		sc.Referral = capInt(sc.Referral, 15)
		sc.Total = sc.Community + sc.Academy + sc.Referral
		if sc.Total > 0 {
			out = append(out, sc)
		}
	}
	return out, rows.Err()
}

func scoreBeats(a, b EngageScore) bool {
	if a.Total != b.Total {
		return a.Total > b.Total
	}
	if a.Referral != b.Referral {
		return a.Referral > b.Referral
	}
	if a.Academy != b.Academy {
		return a.Academy > b.Academy
	}
	return a.TelegramID < b.TelegramID
}

func effectiveTotal(sc EngageScore, recent map[string]bool) int {
	if recent[sc.TelegramID] {
		return sc.Total / 2
	}
	return sc.Total
}

func pickWinner(scores []EngageScore, staff map[string]bool, recent map[string]bool) (EngageScore, bool) {
	var best EngageScore
	bestEff := -1
	ok := false
	for _, sc := range scores {
		if staff[sc.TelegramID] {
			continue
		}
		eff := effectiveTotal(sc, recent)
		if !ok || eff > bestEff || (eff == bestEff && scoreBeats(sc, best)) {
			best = sc
			bestEff = eff
			ok = true
		}
	}
	return best, ok
}

func (s *Store) RecentAirtimeWinners(ctx context.Context, since time.Time) (map[string]bool, error) {
	out := map[string]bool{}
	if s == nil || s.pool == nil {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT winner_telegram_id
FROM community_reward_days
WHERE winner_telegram_id <> ''
  AND stage IN ('sent', 'verified')
  AND COALESCE(sent_at, day::timestamptz) >= $1
`, since)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return out, err
		}
		if id = strings.TrimSpace(id); id != "" {
			out[id] = true
		}
	}
	return out, rows.Err()
}

func (s *Store) AirtimeSentOnLagosDay(ctx context.Context, telegramID string, day time.Time) (bool, error) {
	telegramID = strings.TrimSpace(telegramID)
	if telegramID == "" {
		return false, nil
	}
	d := day.In(lagosLocation()).Format("2006-01-02")
	var n int
	err := s.pool.QueryRow(ctx, `
SELECT COUNT(*) FROM community_reward_days
WHERE winner_telegram_id = $1
  AND stage IN ('sent', 'verified')
  AND COALESCE((sent_at AT TIME ZONE 'Africa/Lagos')::date, day) = $2::date
`, telegramID, d).Scan(&n)
	return n > 0, err
}

func (s *Store) GetRewardDay(ctx context.Context, day time.Time) (RewardDay, error) {
	d := day.In(lagosLocation()).Format("2006-01-02")
	row := s.pool.QueryRow(ctx, `
SELECT day, winner_telegram_id, winner_name, group_chat_id, stage, amount, phone, network,
       request_id, order_id, club_status, community_pts, academy_pts, referral_pts, announced_at, checked_in_at
FROM community_reward_days WHERE day = $1::date`, d)
	rd, err := scanRewardDay(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return RewardDay{}, nil
	}
	return rd, err
}

func (s *Store) OpenRewardFor(ctx context.Context, telegramID string) (RewardDay, error) {
	telegramID = strings.TrimSpace(telegramID)
	if telegramID == "" {
		return RewardDay{}, nil
	}
	row := s.pool.QueryRow(ctx, `
SELECT day, winner_telegram_id, winner_name, group_chat_id, stage, amount, phone, network,
       request_id, order_id, club_status, community_pts, academy_pts, referral_pts, announced_at, checked_in_at
FROM community_reward_days
WHERE winner_telegram_id = $1 AND stage IN ('await_phone','await_confirm','sent')
ORDER BY day DESC
LIMIT 1`, telegramID)
	rd, err := scanRewardDay(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return RewardDay{}, nil
	}
	return rd, err
}

func (s *Store) UpsertRewardDay(ctx context.Context, rd RewardDay) error {
	d := rd.Day.In(lagosLocation()).Format("2006-01-02")
	_, err := s.pool.Exec(ctx, `
INSERT INTO community_reward_days (
  day, winner_telegram_id, winner_name, group_chat_id, stage, amount, phone, network,
  request_id, order_id, club_status, community_pts, academy_pts, referral_pts, updated_at
) VALUES (
  $1::date,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14, now()
)
ON CONFLICT (day) DO UPDATE SET
  winner_telegram_id = EXCLUDED.winner_telegram_id,
  winner_name = EXCLUDED.winner_name,
  group_chat_id = EXCLUDED.group_chat_id,
  stage = EXCLUDED.stage,
  amount = EXCLUDED.amount,
  phone = EXCLUDED.phone,
  network = EXCLUDED.network,
  request_id = EXCLUDED.request_id,
  order_id = EXCLUDED.order_id,
  club_status = EXCLUDED.club_status,
  community_pts = EXCLUDED.community_pts,
  academy_pts = EXCLUDED.academy_pts,
  referral_pts = EXCLUDED.referral_pts,
  announced_at = CASE WHEN EXCLUDED.stage IN ('await_phone','await_confirm','sent','verified') AND community_reward_days.announced_at IS NULL THEN now() ELSE community_reward_days.announced_at END,
  sent_at = CASE WHEN EXCLUDED.stage IN ('sent','verified') AND community_reward_days.sent_at IS NULL THEN now() ELSE community_reward_days.sent_at END,
  verified_at = CASE WHEN EXCLUDED.stage = 'verified' THEN now() ELSE community_reward_days.verified_at END,
  updated_at = now()
`, d, rd.WinnerID, rd.WinnerName, rd.GroupChatID, rd.Stage, rd.Amount, rd.Phone, rd.Network, rd.RequestID, rd.OrderID, rd.ClubStatus, rd.CommunityPts, rd.AcademyPts, rd.ReferralPts)
	return err
}

func (s *Store) MarkRewardAnnounced(ctx context.Context, day time.Time) error {
	d := day.In(lagosLocation()).Format("2006-01-02")
	_, err := s.pool.Exec(ctx, `
UPDATE community_reward_days
SET announced_at = COALESCE(announced_at, now()), updated_at = now()
WHERE day = $1::date
`, d)
	return err
}

func (s *Store) MarkRewardCheckedIn(ctx context.Context, day time.Time) error {
	d := day.In(lagosLocation()).Format("2006-01-02")
	_, err := s.pool.Exec(ctx, `
UPDATE community_reward_days
SET checked_in_at = COALESCE(checked_in_at, now()), updated_at = now()
WHERE day = $1::date
`, d)
	return err
}

func scanRewardDay(scan func(dest ...any) error) (RewardDay, error) {
	var rd RewardDay
	err := scan(&rd.Day, &rd.WinnerID, &rd.WinnerName, &rd.GroupChatID, &rd.Stage, &rd.Amount, &rd.Phone, &rd.Network, &rd.RequestID, &rd.OrderID, &rd.ClubStatus, &rd.CommunityPts, &rd.AcademyPts, &rd.ReferralPts, &rd.AnnouncedAt, &rd.CheckedInAt)
	return rd, err
}
