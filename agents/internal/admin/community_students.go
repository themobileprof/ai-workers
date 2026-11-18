package admin

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// CommunityStudent is one Academy Telegram person. Email is how we know their LMS account.
type CommunityStudent struct {
	ID           int64
	TelegramID   string
	Username     string
	Name         string
	Email        string
	CourseTitle  string
	LessonNote   string
	LastAsk      string
	ChatID       string
	MandateID    int64
	ReferralCode string
	Phone        string
	Network      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *Store) migrateCommunityStudents(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS community_students (
  id BIGSERIAL PRIMARY KEY,
  telegram_id TEXT NOT NULL UNIQUE,
  username TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  course_title TEXT NOT NULL DEFAULT '',
  lesson_note TEXT NOT NULL DEFAULT '',
  last_ask TEXT NOT NULL DEFAULT '',
  chat_id TEXT NOT NULL DEFAULT '',
  mandate_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS community_students_email_idx ON community_students (email) WHERE email <> '';
ALTER TABLE community_students ADD COLUMN IF NOT EXISTS last_ask TEXT NOT NULL DEFAULT '';
ALTER TABLE community_students ADD COLUMN IF NOT EXISTS referral_code TEXT NOT NULL DEFAULT '';
ALTER TABLE community_students ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '';
ALTER TABLE community_students ADD COLUMN IF NOT EXISTS network TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS community_students_ref_idx ON community_students (referral_code) WHERE referral_code <> '';
`)
	return err
}

func (s *Store) UpsertCommunityStudent(ctx context.Context, st CommunityStudent) (CommunityStudent, error) {
	st.TelegramID = strings.TrimSpace(st.TelegramID)
	st.Email = strings.ToLower(strings.TrimSpace(st.Email))
	st.Username = strings.TrimSpace(st.Username)
	st.Name = strings.TrimSpace(st.Name)
	st.CourseTitle = strings.TrimSpace(st.CourseTitle)
	st.LessonNote = strings.TrimSpace(st.LessonNote)
	st.LastAsk = strings.ToLower(strings.TrimSpace(st.LastAsk))
	st.ChatID = strings.TrimSpace(st.ChatID)
	if st.TelegramID == "" {
		return CommunityStudent{}, nil
	}
	st.Phone = ngPhoneLocal(st.Phone)
	st.Network = strings.ToLower(strings.TrimSpace(st.Network))
	st.ReferralCode = strings.ToLower(strings.TrimSpace(st.ReferralCode))
	_, err := s.pool.Exec(ctx, `
INSERT INTO community_students (telegram_id, username, name, email, course_title, lesson_note, last_ask, chat_id, mandate_id, referral_code, phone, network, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, now())
ON CONFLICT (telegram_id) DO UPDATE SET
  username = CASE WHEN EXCLUDED.username <> '' THEN EXCLUDED.username ELSE community_students.username END,
  name = CASE WHEN EXCLUDED.name <> '' THEN EXCLUDED.name ELSE community_students.name END,
  email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE community_students.email END,
  course_title = CASE WHEN EXCLUDED.course_title <> '' THEN EXCLUDED.course_title ELSE community_students.course_title END,
  lesson_note = CASE WHEN EXCLUDED.lesson_note <> '' THEN EXCLUDED.lesson_note ELSE community_students.lesson_note END,
  last_ask = CASE WHEN EXCLUDED.last_ask <> '' THEN EXCLUDED.last_ask ELSE community_students.last_ask END,
  chat_id = CASE WHEN EXCLUDED.chat_id <> '' THEN EXCLUDED.chat_id ELSE community_students.chat_id END,
  mandate_id = CASE WHEN EXCLUDED.mandate_id <> 0 THEN EXCLUDED.mandate_id ELSE community_students.mandate_id END,
  phone = CASE WHEN EXCLUDED.phone <> '' THEN EXCLUDED.phone ELSE community_students.phone END,
  network = CASE WHEN EXCLUDED.network <> '' THEN EXCLUDED.network ELSE community_students.network END,
  updated_at = now()
`, st.TelegramID, st.Username, st.Name, st.Email, st.CourseTitle, st.LessonNote, st.LastAsk, st.ChatID, st.MandateID, st.ReferralCode, st.Phone, st.Network)
	if err != nil {
		return CommunityStudent{}, err
	}
	got, err := s.LookupCommunityStudent(ctx, st.TelegramID)
	if err != nil || got.ID == 0 {
		return got, err
	}
	got, err = s.ensureReferralCode(ctx, got)
	if err != nil {
		return got, err
	}
	if email := strings.ToLower(strings.TrimSpace(got.Email)); email != "" {
		var pending string
		if qerr := s.pool.QueryRow(ctx, `SELECT code FROM community_referral_email_codes WHERE email = $1`, email).Scan(&pending); qerr == nil && strings.TrimSpace(pending) != "" {
			_ = s.bindReferralCode(ctx, got, pending)
			got, err = s.LookupCommunityStudent(ctx, got.TelegramID)
		}
	}
	return got, err
}

func (s *Store) LookupCommunityStudent(ctx context.Context, telegramID string) (CommunityStudent, error) {
	telegramID = strings.TrimSpace(telegramID)
	if telegramID == "" {
		return CommunityStudent{}, nil
	}
	row := s.pool.QueryRow(ctx, communityStudentSelect+` WHERE telegram_id = $1`, telegramID)
	st, err := scanCommunityStudent(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CommunityStudent{}, nil
		}
		return CommunityStudent{}, err
	}
	return st, nil
}

func (s *Store) LookupCommunityStudentByCode(ctx context.Context, code string) (CommunityStudent, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return CommunityStudent{}, nil
	}
	row := s.pool.QueryRow(ctx, communityStudentSelect+` WHERE referral_code = $1`, code)
	st, err := scanCommunityStudent(row.Scan)
	if err == nil {
		return st, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return CommunityStudent{}, err
	}
	var telegramID string
	err = s.pool.QueryRow(ctx, `SELECT telegram_id FROM community_referral_aliases WHERE code = $1`, code).Scan(&telegramID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CommunityStudent{}, nil
		}
		return CommunityStudent{}, err
	}
	return s.LookupCommunityStudent(ctx, telegramID)
}

func (s *Store) LookupCommunityStudentByEmail(ctx context.Context, email string) (CommunityStudent, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return CommunityStudent{}, nil
	}
	row := s.pool.QueryRow(ctx, communityStudentSelect+` WHERE lower(email) = $1 ORDER BY updated_at DESC LIMIT 1`, email)
	st, err := scanCommunityStudent(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CommunityStudent{}, nil
		}
		return CommunityStudent{}, err
	}
	return st, nil
}

func (s *Store) BindReferralCodeByEmail(ctx context.Context, email, code string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.ToLower(strings.TrimSpace(code))
	if email == "" || code == "" {
		return nil
	}
	_, _ = s.pool.Exec(ctx, `
INSERT INTO community_referral_email_codes (email, code)
VALUES ($1,$2)
ON CONFLICT (email) DO UPDATE SET code = EXCLUDED.code
`, email, code)
	st, err := s.LookupCommunityStudentByEmail(ctx, email)
	if err != nil || st.TelegramID == "" {
		return err
	}
	return s.bindReferralCode(ctx, st, code)
}

func (s *Store) bindReferralCode(ctx context.Context, st CommunityStudent, code string) error {
	code = strings.ToLower(strings.TrimSpace(code))
	if st.TelegramID == "" || code == "" {
		return nil
	}
	old := strings.ToLower(strings.TrimSpace(st.ReferralCode))
	if old != "" && old != code {
		_, _ = s.pool.Exec(ctx, `
INSERT INTO community_referral_aliases (code, telegram_id)
VALUES ($1,$2)
ON CONFLICT (code) DO UPDATE SET telegram_id = EXCLUDED.telegram_id
`, old, st.TelegramID)
	}
	_, err := s.pool.Exec(ctx, `UPDATE community_students SET referral_code = $2, updated_at = now() WHERE telegram_id = $1`, st.TelegramID, code)
	if err != nil {
		return err
	}
	_, _ = s.pool.Exec(ctx, `
INSERT INTO community_referral_aliases (code, telegram_id)
VALUES ($1,$2)
ON CONFLICT (code) DO UPDATE SET telegram_id = EXCLUDED.telegram_id
`, code, st.TelegramID)
	return nil
}

func (s *Store) ListCommunityStudents(ctx context.Context, limit int) ([]CommunityStudent, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, communityStudentSelect+` ORDER BY updated_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommunityStudent
	for rows.Next() {
		st, err := scanCommunityStudent(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

const communityStudentSelect = `
SELECT id, telegram_id, username, name, email, course_title, lesson_note, last_ask, chat_id, mandate_id,
       referral_code, phone, network, created_at, updated_at
FROM community_students`

func scanCommunityStudent(scan func(dest ...any) error) (CommunityStudent, error) {
	var st CommunityStudent
	err := scan(&st.ID, &st.TelegramID, &st.Username, &st.Name, &st.Email, &st.CourseTitle, &st.LessonNote, &st.LastAsk, &st.ChatID, &st.MandateID, &st.ReferralCode, &st.Phone, &st.Network, &st.CreatedAt, &st.UpdatedAt)
	return st, err
}

func referralCodeFor(telegramID string) string {
	sum := sha256.Sum256([]byte("tmp-ref:" + strings.TrimSpace(telegramID)))
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	var b strings.Builder
	for i := 0; i < 6; i++ {
		b.WriteByte(alphabet[int(sum[i])%len(alphabet)])
	}
	return b.String()
}

func (s *Store) ensureReferralCode(ctx context.Context, st CommunityStudent) (CommunityStudent, error) {
	if strings.TrimSpace(st.ReferralCode) != "" || st.TelegramID == "" {
		return st, nil
	}
	code := referralCodeFor(st.TelegramID)
	_, err := s.pool.Exec(ctx, `UPDATE community_students SET referral_code = $2, updated_at = now() WHERE id = $1 AND referral_code = ''`, st.ID, code)
	if err != nil {
		return st, err
	}
	st.ReferralCode = code
	return st, nil
}

func (s *Store) BindLMSMandateChat(ctx context.Context, chatID string) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
UPDATE community_mandates
SET telegram_chat_id = $1, updated_at = now()
WHERE slug = 'lms' AND (telegram_chat_id = '' OR telegram_chat_id = $1)
`, chatID)
	return err
}

func (s *Store) MigrateTelegramChat(ctx context.Context, from, to string) error {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" || to == "" || from == to {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
UPDATE community_mandates SET telegram_chat_id = $2, updated_at = now() WHERE telegram_chat_id = $1;
UPDATE community_reward_days SET group_chat_id = $2, updated_at = now() WHERE group_chat_id = $1;
`, from, to)
	return err
}
