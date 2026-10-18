package admin

import (
	"context"
	"errors"
	"strings"
	"time"
)

func (s *Store) ListProjectEvents(ctx context.Context, projectID int64, plannedOnly bool) ([]ProjectEvent, error) {
	q := `
SELECT e.id, e.project_id, p.name, e.title, e.kind, e.starts_on, COALESCE(e.ends_on, e.starts_on),
       e.venue, e.audience, e.outcome, e.status, e.created_at, e.updated_at
FROM project_events e
JOIN projects p ON p.id = e.project_id
`
	args := []any{}
	if projectID > 0 {
		q += ` WHERE e.project_id = $1`
		args = append(args, projectID)
		if plannedOnly {
			q += ` AND e.status = 'planned'`
		}
	} else if plannedOnly {
		q += ` WHERE e.status = 'planned'`
	}
	q += ` ORDER BY e.starts_on ASC, e.id ASC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectEvent
	for rows.Next() {
		var e ProjectEvent
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.ProjectName, &e.Title, &e.Kind, &e.StartsOn, &e.EndsOn, &e.Venue, &e.Audience, &e.Outcome, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) InsertProjectEvent(ctx context.Context, e ProjectEvent) (ProjectEvent, error) {
	if e.ProjectID <= 0 {
		return ProjectEvent{}, errors.New("project is required")
	}
	if e.EndsOn.IsZero() {
		e.EndsOn = e.StartsOn
	}
	var ends any
	if !e.EndsOn.IsZero() {
		ends = e.EndsOn
	}
	err := s.pool.QueryRow(ctx, `
INSERT INTO project_events (project_id, title, kind, starts_on, ends_on, venue, audience, outcome, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'planned')
RETURNING id, created_at, updated_at
`, e.ProjectID, e.Title, e.Kind, e.StartsOn, ends, e.Venue, e.Audience, e.Outcome).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return ProjectEvent{}, friendlyDBErr(err)
	}
	e.Status = "planned"
	return e, nil
}

func (s *Store) GetProjectEvent(ctx context.Context, projectID, id int64) (ProjectEvent, error) {
	var e ProjectEvent
	err := s.pool.QueryRow(ctx, `
SELECT e.id, e.project_id, p.name, e.title, e.kind, e.starts_on, COALESCE(e.ends_on, e.starts_on),
       e.venue, e.audience, e.outcome, e.status, e.created_at, e.updated_at
FROM project_events e
JOIN projects p ON p.id = e.project_id
WHERE e.id = $1 AND e.project_id = $2
`, id, projectID).Scan(&e.ID, &e.ProjectID, &e.ProjectName, &e.Title, &e.Kind, &e.StartsOn, &e.EndsOn, &e.Venue, &e.Audience, &e.Outcome, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}

func (s *Store) SetProjectEventStatus(ctx context.Context, projectID, id int64, status string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "planned", "delivered", "skipped":
	default:
		return errors.New("unknown status")
	}
	tag, err := s.pool.Exec(ctx, `
UPDATE project_events SET status = $3, updated_at = now()
WHERE id = $1 AND project_id = $2
`, id, projectID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("date not found")
	}
	return nil
}

func (s *Store) SetProjectNotes(ctx context.Context, id int64, notes string) error {
	_, err := s.pool.Exec(ctx, `UPDATE projects SET notes = $2, updated_at = now() WHERE id = $1`, id, strings.TrimSpace(notes))
	return err
}

func eventJSON(e ProjectEvent) map[string]any {
	end := e.EndsOn
	if end.IsZero() {
		end = e.StartsOn
	}
	return map[string]any{
		"id": e.ID, "project_id": e.ProjectID, "project": e.ProjectName,
		"title": e.Title, "kind": e.Kind, "starts_on": e.StartsOn.Format("2006-01-02"),
		"ends_on": end.Format("2006-01-02"), "venue": e.Venue, "audience": e.Audience,
		"outcome": e.Outcome, "status": e.Status, "when": e.WhenLabel(),
	}
}

func civilDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func upcomingEventJSON(events []ProjectEvent) []map[string]any {
	var out []map[string]any
	today := civilDay(time.Now())
	for _, e := range events {
		if !eventStillOpen(e, today) {
			continue
		}
		out = append(out, eventJSON(e))
	}
	return out
}
