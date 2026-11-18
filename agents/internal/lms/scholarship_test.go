package lms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDaysUntilAndApproaching(t *testing.T) {
	end := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	if d := daysUntil(end, now); d != 15 {
		t.Fatalf("days=%d", d)
	}
	if daysUntil(end, time.Date(2026, 11, 30, 8, 0, 0, 0, time.UTC)) != 0 {
		t.Fatal("last day")
	}
}

func TestScholarshipBlockHidesCodeAndNames(t *testing.T) {
	s := Season{
		Open:            true,
		Title:           "Unilag discount",
		PathTitle:       "Scholarship",
		DiscountName:    "Unilag discount",
		DiscountPercent: 50,
		Openings:        5,
		Prize:           "₦300,000",
		EndsOn:          time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
		DaysLeft:        15,
		Approaching:     true,
		CodesUsed:       1,
		MaxStudents:     200,
		CompletionPct:   "0.00",
		TestsPassed:     0,
		AwardsGiven:     0,
		Micros:          []string{"Starting Termux", "Linux Navigation"},
		Details:         "Ask your course rep for the code.",
	}
	got := ScholarshipBlock(s)
	if !strings.Contains(got, "Unilag discount") || !strings.Contains(got, "30 November 2026") || !strings.Contains(got, "days_left: 15") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "discount seats used: 1 of 200") || !strings.Contains(got, "lab tests passed: 0") {
		t.Fatal(got)
	}
	if strings.Contains(got, "UNILAGCS2026") || strings.Contains(strings.ToLower(got), "email") {
		t.Fatalf("leaked: %s", got)
	}
	if !strings.Contains(got, "course rep") {
		t.Fatal(got)
	}
}

func TestScholarshipFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/api/auth/admin/login"):
			_, _ = w.Write([]byte(`{"token":"jwt-sch"}`))
		case r.URL.Path == "/api/mini-courses/scholarship" && r.Method == http.MethodGet && r.Header.Get("Authorization") == "":
			_, _ = w.Write([]byte(`{"title":"Scholarship","slug":"scholarship","microCourses":[{"title":"Starting Termux"}],"season":{"title":"Unilag discount","details":"Ask your course rep for the code.","openings":5,"prize":"₦300,000","startsOn":"2026-11-01T00:00:00.000Z","endsOn":"2026-11-30T00:00:00.000Z","discountName":"Unilag discount","discountPercent":50}}`))
		case strings.HasSuffix(r.URL.Path, "/capstone/ranking"):
			if r.Header.Get("Authorization") != "Bearer jwt-sch" {
				http.Error(w, `{}`, http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"ranking":[{"name":"Ada"}],"season":{"discountCode":"UNILAGCS2026","maxStudents":200,"winners":[]},"pastSeasons":[],"earlierWinners":[]}`))
		case strings.Contains(r.URL.Path, "/api/admin/sponsorships") && !strings.Contains(r.URL.Path, "/usage"):
			_, _ = w.Write([]byte(`{"sponsorships":[{"id":"sp1","discount_code":"UNILAGCS2026","students_used":1,"max_students":200,"completion_rate":"0.00","end_date":"2026-11-30T00:00:00.000Z","status":"active"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	c.email = "bot@example.com"
	c.pass = "secret"
	got, err := c.Scholarship(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Open || got.DiscountName != "Unilag discount" || got.CodesUsed != 1 || got.TestsPassed != 1 {
		t.Fatalf("%+v", got)
	}
	block := ScholarshipBlock(got)
	if strings.Contains(block, "UNILAGCS2026") || strings.Contains(block, "Ada") {
		t.Fatalf("code or name leaked:\n%s", block)
	}
}

func TestLooksLikeScholarship(t *testing.T) {
	if !LooksLikeScholarship("when does the scholarship end") || LooksLikeScholarship("what is pwd") {
		t.Fatal("detect")
	}
}
