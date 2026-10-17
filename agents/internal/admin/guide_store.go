package admin

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Store) migrateGuide(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS guide_asks (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  question TEXT NOT NULL,
  answer TEXT NOT NULL DEFAULT '',
  page TEXT NOT NULL DEFAULT '',
  unsure BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS guide_asks_created_idx ON guide_asks (created_at DESC);
`)
	return err
}

func (s *Store) InsertGuideAsk(ctx context.Context, userID int64, question, answer, page string, unsure bool) error {
	question = strings.TrimSpace(question)
	if question == "" {
		return fmt.Errorf("empty question")
	}
	if len(question) > 800 {
		question = question[:800]
	}
	if len(answer) > 8000 {
		answer = answer[:8000]
	}
	page = strings.TrimSpace(page)
	if len(page) > 300 {
		page = page[:300]
	}
	var uid *int64
	if userID > 0 {
		uid = &userID
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO guide_asks (user_id, question, answer, page, unsure)
VALUES ($1, $2, $3, $4, $5)
`, uid, question, answer, page, unsure)
	return err
}

func (s *Store) ListGuideAsks(ctx context.Context, limit int) ([]GuideAsk, error) {
	if limit <= 0 || limit > 400 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
SELECT a.id, COALESCE(a.user_id, 0), COALESCE(u.name, ''), a.question, a.answer, a.page, a.unsure, a.created_at
FROM guide_asks a
LEFT JOIN users u ON u.id = a.user_id
ORDER BY a.created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GuideAsk
	for rows.Next() {
		var a GuideAsk
		if err := rows.Scan(&a.ID, &a.UserID, &a.UserName, &a.Question, &a.Answer, &a.Page, &a.Unsure, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) RecentGuideQA(ctx context.Context, limit int) ([]GuideAsk, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := s.pool.Query(ctx, `
SELECT a.id, COALESCE(a.user_id, 0), COALESCE(u.name, ''), a.question, a.answer, a.page, a.unsure, a.created_at
FROM guide_asks a
LEFT JOIN users u ON u.id = a.user_id
WHERE a.unsure = false AND a.answer <> ''
ORDER BY a.created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GuideAsk
	for rows.Next() {
		var a GuideAsk
		if err := rows.Scan(&a.ID, &a.UserID, &a.UserName, &a.Question, &a.Answer, &a.Page, &a.Unsure, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func ExportGuideAsksMarkdown(asks []GuideAsk) string {
	var b strings.Builder
	b.WriteString("# Guide asks (live dump)\n\n")
	b.WriteString("Fold unanswered or weakly answered items into `agents/internal/admin/guide/faq.md`.\n")
	b.WriteString("If the playbook was wrong, fix `playbook/*.html` and `guide/handbook.md` too.\n\n")
	if len(asks) == 0 {
		b.WriteString("No questions yet.\n")
		return b.String()
	}
	for _, a := range asks {
		when := a.CreatedAt.In(time.Local).Format("2006-01-02 15:04")
		who := strings.TrimSpace(a.UserName)
		if who == "" {
			who = "unknown"
		}
		flag := ""
		if a.Unsure {
			flag = " · UNSURE"
		}
		page := a.Page
		if page == "" {
			page = "(no page)"
		}
		b.WriteString("## ")
		b.WriteString(when)
		b.WriteString(" · ")
		b.WriteString(who)
		b.WriteString(" · ")
		b.WriteString(page)
		b.WriteString(flag)
		b.WriteString("\n\n**Q:** ")
		b.WriteString(a.Question)
		b.WriteString("\n\n**A:** ")
		b.WriteString(a.Answer)
		b.WriteString("\n\n")
	}
	return b.String()
}
