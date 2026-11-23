package admin

import (
	"context"
	"strings"
)

type OpsChatLine struct {
	Author string
	Body   string
}

func (s *Store) migrateOpsChat(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS ops_chats (
  chat_id TEXT PRIMARY KEY,
  user_id BIGINT NOT NULL DEFAULT 0,
  department TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS ops_chat_messages (
  id BIGSERIAL PRIMARY KEY,
  chat_id TEXT NOT NULL,
  author TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS ops_chat_messages_chat_idx ON ops_chat_messages (chat_id, id DESC);
ALTER TABLE ops_chats ADD COLUMN IF NOT EXISTS pending_job_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE ops_chats ADD COLUMN IF NOT EXISTS await_reject BOOLEAN NOT NULL DEFAULT false;
`)
	return err
}

type OpsChatState struct {
	Desk        string
	PendingJob  int64
	AwaitReject bool
}

func (s *Store) OpsChatState(ctx context.Context, chatID string) OpsChatState {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" || s == nil || s.pool == nil {
		return OpsChatState{}
	}
	var st OpsChatState
	err := s.pool.QueryRow(ctx, `
SELECT department, pending_job_id, await_reject FROM ops_chats WHERE chat_id = $1
`, chatID).Scan(&st.Desk, &st.PendingJob, &st.AwaitReject)
	if err != nil {
		return OpsChatState{}
	}
	st.Desk = strings.TrimSpace(st.Desk)
	return st
}

func (s *Store) SetOpsChatPending(ctx context.Context, chatID string, jobID int64) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" || s == nil || s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
UPDATE ops_chats SET pending_job_id=$2, await_reject=false, updated_at=NOW() WHERE chat_id=$1
`, chatID, jobID)
	return err
}

func (s *Store) SetOpsChatAwaitReject(ctx context.Context, chatID string, on bool) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" || s == nil || s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
UPDATE ops_chats SET await_reject=$2, updated_at=NOW() WHERE chat_id=$1
`, chatID, on)
	return err
}

func (s *Store) OpsChatDesk(ctx context.Context, chatID string) string {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return ""
	}
	var dept string
	err := s.pool.QueryRow(ctx, `SELECT department FROM ops_chats WHERE chat_id = $1`, chatID).Scan(&dept)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(dept)
}

func (s *Store) SetOpsChatDesk(ctx context.Context, chatID string, userID int64, dept string) error {
	chatID = strings.TrimSpace(chatID)
	dept = strings.TrimSpace(dept)
	if chatID == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO ops_chats (chat_id, user_id, department, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (chat_id) DO UPDATE SET user_id = EXCLUDED.user_id, department = EXCLUDED.department, updated_at = NOW()
`, chatID, userID, dept)
	return err
}

func (s *Store) AppendOpsChat(ctx context.Context, chatID, author, body string) error {
	chatID = strings.TrimSpace(chatID)
	body = strings.TrimSpace(body)
	if chatID == "" || body == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO ops_chat_messages (chat_id, author, body) VALUES ($1, $2, $3)`, chatID, author, body)
	return err
}

func (s *Store) ListOpsChat(ctx context.Context, chatID string, n int) ([]OpsChatLine, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, nil
	}
	if n <= 0 {
		n = 12
	}
	rows, err := s.pool.Query(ctx, `
SELECT author, body FROM (
  SELECT author, body, id FROM ops_chat_messages WHERE chat_id = $1 ORDER BY id DESC LIMIT $2
) t ORDER BY id ASC
`, chatID, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OpsChatLine
	for rows.Next() {
		var line OpsChatLine
		if err := rows.Scan(&line.Author, &line.Body); err != nil {
			return nil, err
		}
		out = append(out, line)
	}
	return out, rows.Err()
}
