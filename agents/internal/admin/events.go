package admin

import (
	"fmt"
	"strings"
	"time"
)

// ProjectEvent is a dated thing on a bet: a campus training, field week, or launch.
// It is not a second company and not a Community mandate (those are Telegram rooms).
type ProjectEvent struct {
	ID          int64
	ProjectID   int64
	ProjectName string
	Title       string
	Kind        string
	StartsOn    time.Time
	EndsOn      time.Time
	Venue       string
	Audience    string
	Outcome     string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func eventKinds() []stageSpec {
	return []stageSpec{
		{ID: "training", Label: "Training", Help: "Campus, workshop, intro class. Funnel to a paid course if that is the point."},
		{ID: "field", Label: "Field week", Help: "Go to the place and bring names back."},
		{ID: "launch", Label: "Launch", Help: "A date the offer goes live."},
		{ID: "other", Label: "Other date", Help: "Anything that must not vanish into This week."},
	}
}

func validEventKind(id string) bool {
	for _, k := range eventKinds() {
		if k.ID == id {
			return true
		}
	}
	return false
}

func (e ProjectEvent) Planned() bool { return e.Status == "planned" || e.Status == "" }

func (e ProjectEvent) StatusStamp() string {
	switch e.Status {
	case "delivered":
		return "ok"
	case "skipped":
		return "dark"
	default:
		return "brass"
	}
}

func (e ProjectEvent) KindLabel() string {
	for _, k := range eventKinds() {
		if k.ID == e.Kind {
			return k.Label
		}
	}
	if e.Kind == "" {
		return "Training"
	}
	return e.Kind
}

func (e ProjectEvent) WhenLabel() string {
	if e.StartsOn.IsZero() {
		return ""
	}
	start := e.StartsOn
	end := e.EndsOn
	if end.IsZero() {
		end = start
	}
	if start.Equal(end) || sameDay(start, end) {
		return start.Format("2 Jan 2006")
	}
	if start.Month() == end.Month() && start.Year() == end.Year() {
		return fmt.Sprintf("%s–%s", start.Format("2"), end.Format("2 Jan 2006"))
	}
	return start.Format("2 Jan 2006") + " – " + end.Format("2 Jan 2006")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func (e ProjectEvent) ThisWeekLine() string {
	var b strings.Builder
	b.WriteString(e.Title)
	if when := e.WhenLabel(); when != "" {
		b.WriteString(" — ")
		b.WriteString(when)
	}
	if e.Venue != "" {
		b.WriteString(" at ")
		b.WriteString(e.Venue)
	}
	if e.Audience != "" {
		b.WriteString(" · ")
		b.WriteString(e.Audience)
	}
	if e.Outcome != "" {
		b.WriteString(" → ")
		b.WriteString(e.Outcome)
	}
	return b.String()
}

func parseISODate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("use a calendar date")
	}
	return t, nil
}

func cleanEvent(title, kind, start, end, venue, audience, outcome string) (ProjectEvent, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return ProjectEvent{}, fmt.Errorf("name the date")
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		kind = "training"
	}
	if !validEventKind(kind) {
		return ProjectEvent{}, fmt.Errorf("unknown kind")
	}
	starts, err := parseISODate(start)
	if err != nil {
		return ProjectEvent{}, err
	}
	var ends time.Time
	if strings.TrimSpace(end) == "" {
		ends = starts
	} else {
		ends, err = parseISODate(end)
		if err != nil {
			return ProjectEvent{}, err
		}
	}
	if ends.Before(starts) {
		return ProjectEvent{}, fmt.Errorf("end cannot be before start")
	}
	return ProjectEvent{
		Title:    title,
		Kind:     kind,
		StartsOn: starts,
		EndsOn:   ends,
		Venue:    strings.TrimSpace(venue),
		Audience: strings.TrimSpace(audience),
		Outcome:  strings.TrimSpace(outcome),
		Status:   "planned",
	}, nil
}

func eventStillOpen(e ProjectEvent, today time.Time) bool {
	if !e.Planned() {
		return false
	}
	end := e.EndsOn
	if end.IsZero() {
		end = e.StartsOn
	}
	if end.IsZero() {
		return true
	}
	return !civilDay(end).Before(civilDay(today))
}

func comingEventLines(u User, events []ProjectEvent, byProject map[int64]Project, today time.Time) []string {
	var out []string
	for _, e := range events {
		if !eventStillOpen(e, today) {
			continue
		}
		p, ok := byProject[e.ProjectID]
		if !ok || !EngagementFor(p.Stage).Pings || !canSeeBet(u, p.Members) {
			continue
		}
		line := e.ProjectName + ": " + e.Title + " (" + e.WhenLabel() + ")"
		if e.Venue != "" {
			line += " at " + e.Venue
		}
		out = append(out, line)
	}
	return out
}

func attachNextEvents(projects []Project, events []ProjectEvent) {
	by := map[int64]ProjectEvent{}
	for _, e := range events {
		if !e.Planned() {
			continue
		}
		if _, ok := by[e.ProjectID]; ok {
			continue
		}
		by[e.ProjectID] = e
	}
	for i := range projects {
		if e, ok := by[projects[i].ID]; ok {
			ev := e
			projects[i].NextEvent = &ev
		}
	}
}
