package admin

import (
	"strings"
	"testing"
	"time"
)

func TestCleanEvent(t *testing.T) {
	if _, err := cleanEvent("", "training", "2026-11-17", "2026-11-19", "UNILAG", "CS final years", "LMS sign-ups"); err == nil {
		t.Fatal("title required")
	}
	if _, err := cleanEvent("Linux on Mobile", "party", "2026-11-17", "", "", "", ""); err == nil {
		t.Fatal("unknown kind")
	}
	if _, err := cleanEvent("Linux on Mobile", "training", "2026-11-19", "2026-11-17", "", "", ""); err == nil {
		t.Fatal("end before start")
	}
	e, err := cleanEvent("Linux on Mobile", "training", "2026-11-17", "2026-11-19", "University of Lagos", "Final year CS", "sign-ups for lms.themobileprof.com")
	if err != nil {
		t.Fatal(err)
	}
	if e.Kind != "training" || e.WhenLabel() != "17–19 Nov 2026" {
		t.Fatalf("%+v %q", e, e.WhenLabel())
	}
	line := e.ThisWeekLine()
	if !strings.Contains(line, "University of Lagos") || !strings.Contains(line, "lms.themobileprof.com") {
		t.Fatalf("line %q", line)
	}
}

func TestEventWhenLabelSameDay(t *testing.T) {
	e := ProjectEvent{StartsOn: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC), EndsOn: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)}
	if e.WhenLabel() != "17 Nov 2026" {
		t.Fatal(e.WhenLabel())
	}
}

func TestAttachNextEvents(t *testing.T) {
	projects := []Project{{ID: 1, Name: "Academy"}, {ID: 2, Name: "HomeGauge"}}
	events := []ProjectEvent{
		{ProjectID: 1, Title: "UNILAG", Status: "planned", StartsOn: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)},
		{ProjectID: 1, Title: "Later", Status: "planned", StartsOn: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)},
		{ProjectID: 2, Title: "Done", Status: "delivered"},
	}
	attachNextEvents(projects, events)
	if projects[0].NextEvent == nil || projects[0].NextEvent.Title != "UNILAG" {
		t.Fatalf("%+v", projects[0].NextEvent)
	}
	if projects[1].NextEvent != nil {
		t.Fatal("delivered must not attach")
	}
}

func TestEventFlash(t *testing.T) {
	if eventFlash("week") == "" || eventFlash("dated") == "" || eventFlash("event_task") == "" {
		t.Fatal("event flashes")
	}
	if eventFlash("nope") != "" {
		t.Fatal("unknown ok")
	}
}

func TestComingEventLines(t *testing.T) {
	today := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	owner := User{ID: 1, Role: "owner"}
	asst := User{ID: 9, Role: "assistant"}
	by := map[int64]Project{
		1: {ID: 1, Stage: "selling", Members: []ProjectMember{{UserID: 9}}},
		2: {ID: 2, Stage: "parked"},
	}
	events := []ProjectEvent{
		{ProjectID: 1, ProjectName: "Academy", Title: "UNILAG", Venue: "UNILAG", Status: "planned", StartsOn: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC), EndsOn: time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)},
		{ProjectID: 2, ProjectName: "Finchest", Title: "Silent", Status: "planned", StartsOn: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)},
	}
	got := comingEventLines(owner, events, by, today)
	if len(got) != 1 || !strings.Contains(got[0], "UNILAG") {
		t.Fatalf("%v", got)
	}
	if n := comingEventLines(asst, events, by, today); len(n) != 1 {
		t.Fatalf("seated %v", n)
	}
	stranger := User{ID: 3, Role: "assistant"}
	if n := comingEventLines(stranger, events, by, today); len(n) != 0 {
		t.Fatalf("leak %v", n)
	}
}

func TestUpcomingEventJSON(t *testing.T) {
	past := ProjectEvent{
		Title: "old", Status: "planned",
		StartsOn: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:   time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	next := ProjectEvent{
		Title: "UNILAG", Status: "planned", ProjectName: "Academy",
		StartsOn: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC),
		EndsOn:   time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC),
	}
	done := ProjectEvent{Title: "done", Status: "delivered", StartsOn: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)}
	got := upcomingEventJSON([]ProjectEvent{past, next, done})
	if len(got) != 1 || got[0]["title"] != "UNILAG" {
		t.Fatalf("%+v", got)
	}
}
