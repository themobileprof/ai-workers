package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

const defaultPaceHours = 6.0

type WorkItem struct {
	ID         int64
	ProjectID  int64
	Project    string
	Stage      string
	AssigneeID int64
	Assignee   string
	Kind       string
	Title      string
	Status     string
	Due        string
	Hours      float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (w WorkItem) Open() bool   { return w.Status == "open" || w.Status == "" }
func (w WorkItem) Queued() bool { return w.Status == "queued" }

func (s *Store) migrateDelivery(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS user_pace (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  hours_per_week DOUBLE PRECISION NOT NULL DEFAULT 6
);
CREATE TABLE IF NOT EXISTS work_items (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  assignee_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  kind TEXT NOT NULL DEFAULT 'task',
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  due_date DATE,
  hours DOUBLE PRECISION NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS work_items_assignee_idx ON work_items (assignee_id) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS work_items_project_idx ON work_items (project_id, status);
CREATE TABLE IF NOT EXISTS delivery_pins (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
  pinned_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}

func parsePace(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultPaceHours
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || n <= 0 {
		return defaultPaceHours
	}
	if n > 40 {
		return 40
	}
	return n
}

func (s *Store) SetUserPace(ctx context.Context, userID int64, hours float64) error {
	if hours <= 0 {
		hours = defaultPaceHours
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO user_pace (user_id, hours_per_week) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET hours_per_week = EXCLUDED.hours_per_week
`, userID, hours)
	return err
}

func (s *Store) UserPace(ctx context.Context, userID int64) float64 {
	var h float64
	err := s.pool.QueryRow(ctx, `SELECT hours_per_week FROM user_pace WHERE user_id = $1`, userID).Scan(&h)
	if err != nil || h <= 0 {
		return defaultPaceHours
	}
	return h
}

func (s *Store) ListWorkItems(ctx context.Context, projectID, assigneeID int64) ([]WorkItem, error) {
	rows, err := s.pool.Query(ctx, `
SELECT w.id, w.project_id, p.name, p.stage, COALESCE(w.assignee_id, 0), COALESCE(u.name, ''),
       w.kind, w.title, w.status, COALESCE(to_char(w.due_date, 'YYYY-MM-DD'), ''), w.hours, w.created_at, w.updated_at
FROM work_items w
JOIN projects p ON p.id = w.project_id
LEFT JOIN users u ON u.id = w.assignee_id
WHERE ($1 = 0 OR w.project_id = $1)
  AND ($2 = 0 OR w.assignee_id = $2)
ORDER BY w.status, w.due_date NULLS LAST, w.id
`, projectID, assigneeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorkItem
	for rows.Next() {
		var w WorkItem
		if err := rows.Scan(&w.ID, &w.ProjectID, &w.Project, &w.Stage, &w.AssigneeID, &w.Assignee,
			&w.Kind, &w.Title, &w.Status, &w.Due, &w.Hours, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) CountOpenWork(ctx context.Context, projectID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM work_items WHERE project_id = $1 AND status = 'open'`, projectID).Scan(&n)
	return n, err
}

func (s *Store) InsertWorkItem(ctx context.Context, it WorkItem) (WorkItem, error) {
	it.Title = strings.TrimSpace(it.Title)
	if it.Title == "" {
		return WorkItem{}, errors.New("title is required")
	}
	if it.Kind == "" {
		it.Kind = "task"
	}
	if it.Hours <= 0 {
		it.Hours = 1
	}
	var due any
	if strings.TrimSpace(it.Due) != "" {
		due = it.Due
	}
	var assignee any
	if it.AssigneeID > 0 {
		assignee = it.AssigneeID
	}
	status := "open"
	if it.Status == "queued" {
		status = "queued"
	}
	err := s.pool.QueryRow(ctx, `
INSERT INTO work_items (project_id, assignee_id, kind, title, status, due_date, hours)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at, updated_at
`, it.ProjectID, assignee, it.Kind, it.Title, status, due, it.Hours).Scan(&it.ID, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		return WorkItem{}, friendlyDBErr(err)
	}
	it.Status = status
	return it, nil
}

func (s *Store) SetWorkItemStatus(ctx context.Context, id int64, status string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "open", "done", "skipped", "queued":
	default:
		return errors.New("unknown status")
	}
	tag, err := s.pool.Exec(ctx, `UPDATE work_items SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("task not found")
	}
	return nil
}

func (s *Store) GetWorkItem(ctx context.Context, id int64) (WorkItem, error) {
	list, err := s.ListWorkItems(ctx, 0, 0)
	if err != nil {
		return WorkItem{}, err
	}
	for _, w := range list {
		if w.ID == id {
			return w, nil
		}
	}
	return WorkItem{}, errors.New("task not found")
}

func (s *Store) PromoteWorkItem(ctx context.Context, id int64) error {
	it, err := s.GetWorkItem(ctx, id)
	if err != nil {
		return err
	}
	if !it.Queued() {
		return errors.New("only a held task can be promoted")
	}
	p, err := s.GetProject(ctx, it.ProjectID)
	if err != nil {
		return err
	}
	eng := EngagementFor(p.Stage)
	open, _ := s.CountOpenWork(ctx, p.ID)
	if eng.MaxOpenTasks <= 0 || open >= eng.MaxOpenTasks {
		return errors.New("no open slot at this stage")
	}
	return s.SetWorkItemStatus(ctx, id, "open")
}

func (s *Store) PinProject(ctx context.Context, userID, projectID int64) error {
	if projectID <= 0 {
		_, err := s.pool.Exec(ctx, `DELETE FROM delivery_pins WHERE user_id = $1`, userID)
		return err
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO delivery_pins (user_id, project_id, pinned_at) VALUES ($1, $2, now())
ON CONFLICT (user_id) DO UPDATE SET project_id = EXCLUDED.project_id, pinned_at = now()
`, userID, projectID)
	return err
}

func (s *Store) PinnedProject(ctx context.Context, userID int64) int64 {
	var id int64
	_ = s.pool.QueryRow(ctx, `
SELECT COALESCE(project_id, 0) FROM delivery_pins
WHERE user_id = $1 AND pinned_at > now() - interval '8 hours'
`, userID).Scan(&id)
	return id
}

func (s *Store) ApplyCheckins(ctx context.Context, actor User, ids []int64, status string) (int, error) {
	n := 0
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		it, err := s.GetWorkItem(ctx, id)
		if err != nil {
			continue
		}
		if actor.Role != "owner" && actor.Role != "bdm" && it.AssigneeID != actor.ID {
			continue
		}
		if err := s.SetWorkItemStatus(ctx, id, status); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (s *Store) AcceptPlanItems(ctx context.Context, actor User, items []WorkItem) (int, error) {
	n := 0
	openBy := map[int64]int{}
	for _, it := range items {
		p, err := s.GetProject(ctx, it.ProjectID)
		if err != nil {
			continue
		}
		if p.Stage == "killed" {
			continue
		}
		eng := EngagementFor(p.Stage)
		open, ok := openBy[p.ID]
		if !ok {
			open, _ = s.CountOpenWork(ctx, p.ID)
			openBy[p.ID] = open
		}
		if it.AssigneeID > 0 {
			u, err := s.GetUser(ctx, it.AssigneeID)
			if err != nil || !canAssign(u, p.ID, p.Members) {
				it.AssigneeID = 0
			}
		}
		if eng.MaxOpenTasks <= 0 || open >= eng.MaxOpenTasks {
			it.Status = "queued"
		} else {
			it.Status = "open"
			openBy[p.ID] = open + 1
		}
		if _, err := s.InsertWorkItem(ctx, it); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (s *Store) deliveryOverlay(ctx context.Context, actor *User, projectSlug string) map[string]any {
	out := map[string]any{
		"cannot_send": true,
		"cannot_hire": true,
		"engagement": map[string]any{
			"idea":     EngagementFor("idea"),
			"testing":  EngagementFor("testing"),
			"selling":  EngagementFor("selling"),
			"fundable": EngagementFor("fundable"),
			"parked":   EngagementFor("parked"),
			"killed":   EngagementFor("killed"),
		},
	}
	projects, err := s.ListProjects(ctx)
	if err != nil {
		return out
	}
	users, _ := s.ListUsers(ctx)
	var people []map[string]any
	var loads []PersonLoad
	for _, u := range users {
		if !u.Active {
			continue
		}
		seats := map[int64]bool{}
		for _, p := range projects {
			if canAssign(u, p.ID, p.Members) && (u.Role == "owner" || u.Role == "bdm" || len(p.Members) > 0) {
				for _, m := range p.Members {
					if m.UserID == u.ID {
						seats[p.ID] = true
					}
				}
				if u.Role == "owner" || u.Role == "bdm" {
					seats[p.ID] = true
				}
			}
		}
		hours := s.UserPace(ctx, u.ID)
		open, _ := s.ListWorkItems(ctx, 0, u.ID)
		used := 0.0
		for _, it := range open {
			if it.Open() {
				used += it.Hours
			}
		}
		left := hours - used
		if left < 0 {
			left = 0
		}
		u.HoursPerWeek = hours
		people = append(people, map[string]any{
			"id": u.ID, "name": u.Name, "role": u.Role, "phone": u.Phone,
			"hours_per_week": hours, "hours_left": left, "office": u.Role == "owner" || u.Role == "bdm",
		})
		loads = append(loads, PersonLoad{
			UserID: u.ID, Name: u.Name, Hours: left,
			Office: u.Role == "owner" || u.Role == "bdm", Projects: seats,
		})
	}
	var bets []map[string]any
	var betLoads []BetLoad
	var visible []Project
	for _, p := range projects {
		if actor != nil && !canSeeBet(*actor, p.Members) {
			continue
		}
		visible = append(visible, p)
		eng := EngagementFor(p.Stage)
		open, _ := s.CountOpenWork(ctx, p.ID)
		slots := eng.MaxOpenTasks - open
		if slots < 0 {
			slots = 0
		}
		items, _ := s.ListWorkItems(ctx, p.ID, 0)
		var itemJSON []map[string]any
		for _, it := range items {
			itemJSON = append(itemJSON, map[string]any{
				"id": it.ID, "title": it.Title, "status": it.Status, "assignee": it.Assignee,
				"assignee_id": it.AssigneeID, "hours": it.Hours, "due": it.Due, "kind": it.Kind,
			})
		}
		bets = append(bets, map[string]any{
			"id": p.ID, "slug": p.Slug, "name": p.Name, "stage": p.Stage,
			"one_liner": p.OneLiner, "notes": p.Notes,
			"hypothesis": p.Hypothesis, "mission": p.Mission, "mission_due": p.MissionDue, "burn_ngn": p.BurnNGN,
			"max_open_tasks": eng.MaxOpenTasks, "weight": eng.Weight,
			"pings": eng.Pings, "idea_talk": eng.IdeaTalk,
			"open_count": open, "slots": slots, "items": itemJSON,
		})
		betLoads = append(betLoads, BetLoad{
			ProjectID: p.ID, Slug: p.Slug, Name: p.Name, Stage: p.Stage,
			Weight: eng.Weight, Slots: slots,
		})
	}
	out["people"] = people
	out["bets"] = bets
	out["suggest_slots"] = Apportion(loads, betLoads)
	if actor != nil {
		out["actor"] = map[string]any{"id": actor.ID, "name": actor.Name, "role": actor.Role, "hours_per_week": s.UserPace(ctx, actor.ID)}
		if pin := s.PinnedProject(ctx, actor.ID); pin > 0 {
			out["pinned_project_id"] = pin
		}
	}
	slug := strings.ToLower(strings.TrimSpace(projectSlug))
	if slug != "" {
		found := false
		for _, p := range visible {
			if p.Slug == slug {
				found = true
				eng := EngagementFor(p.Stage)
				out["project"] = map[string]any{"id": p.ID, "slug": p.Slug, "name": p.Name, "stage": p.Stage, "idea_talk": eng.IdeaTalk, "pings": eng.Pings}
				break
			}
		}
		if !found {
			out["access_denied"] = true
			out["project_slug"] = slug
		}
	}
	upcoming, _ := s.ListProjectEvents(ctx, 0, true)
	var visibleIDs map[int64]bool
	if actor != nil {
		visibleIDs = map[int64]bool{}
		for _, p := range visible {
			visibleIDs[p.ID] = true
		}
	}
	filtered := upcoming
	if visibleIDs != nil {
		filtered = nil
		for _, e := range upcoming {
			if visibleIDs[e.ProjectID] {
				filtered = append(filtered, e)
			}
		}
	}
	out["upcoming_events"] = upcomingEventJSON(filtered)
	return out
}

func (s *Store) LookupUserByPhone(ctx context.Context, phone string) (User, bool, error) {
	phone = normalizePhone(phone)
	if phone == "" {
		return User{}, false, nil
	}
	users, err := s.ListUsers(ctx)
	if err != nil {
		return User{}, false, err
	}
	for _, u := range users {
		if u.Active && normalizePhone(u.Phone) != "" && (normalizePhone(u.Phone) == phone || strings.HasSuffix(phone, normalizePhone(u.Phone)) || strings.HasSuffix(normalizePhone(u.Phone), phone)) {
			u.HoursPerWeek = s.UserPace(ctx, u.ID)
			return u, true, nil
		}
	}
	return User{}, false, nil
}

func (s *Store) MorningDigest(ctx context.Context) []map[string]any {
	users, err := s.ListUsers(ctx)
	if err != nil {
		return nil
	}
	projects, _ := s.ListProjects(ctx)
	byProject := map[int64]Project{}
	for _, p := range projects {
		byProject[p.ID] = p
	}
	upcoming, _ := s.ListProjectEvents(ctx, 0, true)
	today := time.Now()
	var out []map[string]any
	for _, u := range users {
		if !u.Active || u.Phone == "" {
			continue
		}
		items, err := s.ListWorkItems(ctx, 0, u.ID)
		if err != nil {
			continue
		}
		var lines []string
		for _, it := range items {
			if !it.Open() {
				continue
			}
			if !EngagementFor(it.Stage).Pings {
				continue
			}
			line := it.Project + ": " + it.Title
			if it.Due != "" {
				line += " (due " + it.Due + ")"
			}
			lines = append(lines, line)
		}
		coming := comingEventLines(u, upcoming, byProject, today)
		if len(lines) == 0 && len(coming) == 0 {
			continue
		}
		text := "Today on the desk (" + u.Name + "):"
		if len(lines) > 0 {
			text += "\n- " + strings.Join(lines, "\n- ")
		}
		if len(coming) > 0 {
			text += "\nComing up:\n- " + strings.Join(coming, "\n- ")
		}
		text += "\n\nReply /today done 1, still on 2 — or open Delivery. Dates live on the bet."
		out = append(out, map[string]any{
			"user_id": u.ID, "name": u.Name, "phone": u.Phone, "text": text, "speak": true,
		})
	}
	return out
}

func workItemsFromPlan(raw any) []WorkItem {
	var out []WorkItem
	arr, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, row := range arr {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		it := WorkItem{
			ProjectID:  int64Val(m["project_id"]),
			AssigneeID: int64Val(m["assignee_id"]),
			Title:      strVal(m["title"]),
			Kind:       firstNonEmpty(strVal(m["kind"]), "task"),
			Due:        strVal(m["due"]),
			Hours:      floatVal(m["hours"]),
		}
		if it.Hours <= 0 {
			it.Hours = 1
		}
		if it.Title != "" && it.ProjectID > 0 {
			out = append(out, it)
		}
	}
	return out
}

func intIDs(raw any) []int64 {
	var out []int64
	switch v := raw.(type) {
	case []any:
		for _, x := range v {
			if id := int64Val(x); id > 0 {
				out = append(out, id)
			}
		}
	case []int64:
		return v
	}
	return out
}

func floatVal(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
	}
}
