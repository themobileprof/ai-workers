package admin

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/opschat"
)

type DeliveryReport struct {
	ID        int64
	ActorID   int64
	ActorName string
	Summary   string
	Body      string
	CreatedAt time.Time
}

func (r DeliveryReport) WhenLabel() string {
	if r.CreatedAt.IsZero() {
		return ""
	}
	return r.CreatedAt.In(lagosLocation()).Format("2 Jan 2006 15:04")
}

func (s *Store) migrateDeliveryReports(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS delivery_reports (
  id BIGSERIAL PRIMARY KEY,
  actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  summary TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS delivery_reports_created_idx ON delivery_reports (created_at DESC);
`)
	return err
}

func (s *Store) InsertDeliveryReport(ctx context.Context, r DeliveryReport, payload map[string]any) (DeliveryReport, error) {
	r.Summary = clipRunes(strings.TrimSpace(r.Summary), 400)
	r.Body = clipRunes(strings.TrimSpace(r.Body), 4000)
	if r.Summary == "" {
		r.Summary = clipRunes(r.Body, 120)
	}
	var actor any
	if r.ActorID > 0 {
		actor = r.ActorID
	}
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return r, err
	}
	err = s.pool.QueryRow(ctx, `
INSERT INTO delivery_reports (actor_id, summary, body, payload)
VALUES ($1, $2, $3, $4::jsonb)
RETURNING id, created_at
`, actor, r.Summary, r.Body, raw).Scan(&r.ID, &r.CreatedAt)
	return r, err
}

func (s *Store) ListDeliveryReports(ctx context.Context, limit int) ([]DeliveryReport, error) {
	if limit <= 0 {
		limit = 40
	}
	rows, err := s.pool.Query(ctx, `
SELECT r.id, COALESCE(r.actor_id, 0), COALESCE(u.name, ''), r.summary, r.body, r.created_at
FROM delivery_reports r
LEFT JOIN users u ON u.id = r.actor_id
ORDER BY r.created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeliveryReport
	for rows.Next() {
		var r DeliveryReport
		if err := rows.Scan(&r.ID, &r.ActorID, &r.ActorName, &r.Summary, &r.Body, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Server) applyDeliveryReport(ctx context.Context, u User, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	sd := resp.StructuredData
	taskType := strings.ToLower(strVal(sd["task_type"]))
	if taskType == "silent" || taskType == "access_denied" || taskType == "weekly_digest" {
		return
	}
	data := contract.ContextObject(req.ContextData)
	source := firstNonEmpty(req.TaskDescription, strVal(data["reply_to"]))
	logBody := firstNonEmpty(strVal(sd["log"]), strings.TrimSpace(req.TaskDescription))
	summary := firstNonEmpty(strVal(sd["summary"]), clipRunes(logBody, 120))
	if strVal(sd["log"]) != "" || opschat.LooksLikeReport(req.TaskDescription) {
		_, _ = s.store.InsertDeliveryReport(ctx, DeliveryReport{
			ActorID: u.ID,
			Summary: summary,
			Body:    logBody,
		}, map[string]any{
			"output_text": resp.OutputText,
			"contacts":    sd["contacts"],
			"events":      sd["events"],
			"spend":       sd["spend"],
			"projects":    sd["projects"],
		})
	}

	open, _ := s.store.ListWorkItems(ctx, 0, 0)
	doneIDs := intIDs(sd["done_ids"])
	doneIDs = append(doneIDs, matchOpenWork(open, u, stringList(sd["done_titles"]))...)
	skippedIDs := intIDs(sd["skipped_ids"])
	skippedIDs = append(skippedIDs, matchOpenWork(open, u, stringList(sd["skipped_titles"]))...)
	if len(doneIDs) > 0 {
		_, _ = s.store.ApplyCheckins(ctx, u, uniqIDs(doneIDs), "done")
	}
	if len(skippedIDs) > 0 {
		_, _ = s.store.ApplyCheckins(ctx, u, uniqIDs(skippedIDs), "skipped")
	}

	if boolVal(sd["save_own_items"]) {
		var own []WorkItem
		for _, it := range workItemsFromPlan(sd["items"]) {
			if it.AssigneeID == 0 {
				it.AssigneeID = u.ID
			}
			if u.Role != "owner" && u.Role != "bdm" && it.AssigneeID != u.ID {
				continue
			}
			own = append(own, it)
		}
		if len(own) > 0 {
			_, _ = s.store.AcceptPlanItems(ctx, u, own)
		}
	}

	s.applyReportContacts(ctx, u, source, sd)
	s.applyReportEvents(ctx, sd)
	s.applyReportCardNotes(ctx, sd)
	s.fileReportSpend(ctx, u, sd)
}

func (s *Server) applyReportContacts(ctx context.Context, u User, source string, sd map[string]any) {
	for _, raw := range anyMaps(sd["contacts"]) {
		if !boolVal(raw["record_lead"]) && strVal(raw["name"]) == "" && strVal(raw["company"]) == "" && strVal(raw["contact_name"]) == "" {
			continue
		}
		if !boolVal(raw["record_lead"]) {
			continue
		}
		name := firstNonEmpty(strVal(raw["name"]), strVal(raw["contact_name"]))
		company := firstNonEmpty(strVal(raw["company"]), strVal(raw["company_name"]))
		if name == "" {
			name = company
		}
		if name == "" {
			continue
		}
		email := strVal(raw["email"])
		if email != "" && (!strings.Contains(email, "@") || !mentionedIn(source, email)) {
			email = ""
		}
		phone := strVal(raw["phone"])
		if phone != "" && !mentionedIn(source, stripDigits(phone)) {
			phone = ""
		}
		pid := int64Val(raw["project_id"])
		if pid == 0 {
			if slug := strVal(raw["project_slug"]); slug != "" {
				if p, err := s.store.GetProjectBySlug(ctx, slug); err == nil {
					pid = p.ID
				}
			}
		}
		c, err := s.store.UpsertCRMContact(ctx, CRMContact{
			ProjectID:    pid,
			Name:         name,
			Company:      company,
			Email:        email,
			Phone:        phone,
			Stage:        strVal(raw["pipeline_stage"]),
			Offer:        strVal(raw["offer"]),
			NextAction:   strVal(raw["next_action"]),
			NextActionBy: strVal(raw["next_action_by"]),
			Notes:        firstNonEmpty(strVal(raw["notes"]), clipRunes(source, 800)),
			Source:       "telegram",
			LastChannel:  "telegram",
		})
		if err != nil || c.ID == 0 {
			continue
		}
		body := firstNonEmpty(strVal(raw["notes"]), clipRunes(source, 2000))
		_ = s.store.InsertCRMEvent(ctx, CRMEvent{
			ContactID: c.ID,
			Channel:   "telegram",
			Direction: "note",
			Summary:   clipCRM(firstNonEmpty(strVal(sd["summary"]), "Field report from "+u.Name), 400),
			Body:      body,
		})
	}
}

func (s *Server) applyReportEvents(ctx context.Context, sd map[string]any) {
	for _, raw := range anyMaps(sd["events"]) {
		slug := strVal(raw["project_slug"])
		title := strVal(raw["title"])
		if slug == "" || title == "" {
			continue
		}
		p, err := s.store.GetProjectBySlug(ctx, slug)
		if err != nil {
			continue
		}
		start, ok := parseLagosDay(strVal(raw["starts_on"]))
		if !ok {
			continue
		}
		kind := strVal(raw["kind"])
		if !validEventKind(kind) {
			kind = "field"
		}
		_, _ = s.store.InsertProjectEvent(ctx, ProjectEvent{
			ProjectID: p.ID,
			Title:     title,
			Kind:      kind,
			StartsOn:  start,
			Venue:     strVal(raw["venue"]),
			Audience:  strVal(raw["audience"]),
			Outcome:   strVal(raw["outcome"]),
		})
	}
}

func (s *Server) applyReportCardNotes(ctx context.Context, sd map[string]any) {
	for _, raw := range anyMaps(sd["projects"]) {
		slug := strVal(raw["slug"])
		note := strVal(raw["card_note"])
		if slug == "" || note == "" {
			continue
		}
		p, err := s.store.GetProjectBySlug(ctx, slug)
		if err != nil {
			continue
		}
		next := appendCardNote(p.Notes, note)
		if next == p.Notes {
			continue
		}
		_ = s.store.SetProjectNotes(ctx, p.ID, next)
	}
}

func (s *Server) fileReportSpend(ctx context.Context, u User, sd map[string]any) {
	for _, raw := range anyMaps(sd["spend"]) {
		job, ok := spendJobFromReport(u, raw)
		if !ok {
			continue
		}
		if _, err := s.store.InsertDeskJob(ctx, job); err != nil {
			continue
		}
	}
}

func spendJobFromReport(u User, raw map[string]any) (DeskJob, bool) {
	amount := floatVal(raw["amount_ngn"])
	if amount <= 0 {
		amount = floatVal(raw["amount"])
	}
	what := firstNonEmpty(strVal(raw["what"]), strVal(raw["notes"]))
	vendor := strVal(raw["vendor"])
	if amount <= 0 || what == "" {
		return DeskJob{}, false
	}
	summary := "₦" + formatAmount(amount)
	if vendor != "" {
		summary += " · " + vendor
	}
	summary += " · " + what
	return DeskJob{
		Desk:    "accounts",
		Title:   clipRunes("Field spend · "+what, 120),
		Status:  "proposed",
		Source:  "telegram",
		Summary: clipRunes(summary, 400),
		Payload: map[string]any{
			"task_type":    "accounts",
			"amount_ngn":   amount,
			"what":         what,
			"vendor":       vendor,
			"from_report":  true,
			"actor_id":     u.ID,
			"actor_name":   u.Name,
			"cannot_send":  true,
			"cannot_spend": true,
		},
	}, true
}

func matchOpenWork(items []WorkItem, actor User, needles []string) []int64 {
	var out []int64
	seen := map[int64]bool{}
	for _, needle := range needles {
		id := matchOneWork(items, actor, needle)
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func matchOneWork(items []WorkItem, actor User, needle string) int64 {
	needle = strings.ToLower(strings.TrimSpace(needle))
	if needle == "" {
		return 0
	}
	var best int64
	bestScore := 0
	for _, it := range items {
		if !it.Open() {
			continue
		}
		if actor.Role != "owner" && actor.Role != "bdm" && it.AssigneeID != actor.ID {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(it.Title))
		if title == "" {
			continue
		}
		score := 0
		if title == needle {
			score = 3
		} else if strings.Contains(title, needle) || strings.Contains(needle, title) {
			score = 2
		} else if titleWordsMatch(title, needle) {
			score = 1
		}
		if score > bestScore {
			bestScore = score
			best = it.ID
		}
	}
	return best
}

func titleWordsMatch(title, needle string) bool {
	words := strings.FieldsFunc(needle, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	if len(words) == 0 {
		return false
	}
	hit := 0
	for _, w := range words {
		if len(w) < 4 {
			continue
		}
		if strings.Contains(title, w) {
			hit++
		}
	}
	need := 0
	for _, w := range words {
		if len(w) >= 4 {
			need++
		}
	}
	return need > 0 && hit == need
}

func stringList(raw any) []string {
	var out []string
	switch t := raw.(type) {
	case []any:
		for _, x := range t {
			if s := strVal(x); s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, s := range t {
			if strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
	case string:
		if strings.TrimSpace(t) != "" {
			out = append(out, strings.TrimSpace(t))
		}
	}
	return out
}

func anyMaps(raw any) []map[string]any {
	var out []map[string]any
	switch t := raw.(type) {
	case []any:
		for _, x := range t {
			if m, ok := x.(map[string]any); ok {
				out = append(out, m)
			}
		}
	case []map[string]any:
		return t
	}
	return out
}

func uniqIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	var out []int64
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func mentionedIn(hay, needle string) bool {
	needle = strings.TrimSpace(strings.ToLower(needle))
	if needle == "" {
		return false
	}
	return strings.Contains(strings.ToLower(hay), needle)
}

func stripDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func parseLagosDay(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, lagosLocation())
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func formatAmount(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(n, 'f', 2, 64), "0"), ".")
}
