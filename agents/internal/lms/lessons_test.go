package lms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMatchLessonsNumberAndTitle(t *testing.T) {
	ls := []Lesson{
		{ID: "a", Title: "The flags that keep it free", Order: 1},
		{ID: "b", Title: "Create the always-free VM", Order: 2},
		{ID: "c", Title: "Open the firewall", Order: 3},
	}
	got := MatchLessons(ls, "I am stuck on lesson 2")
	if len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("%+v", got)
	}
	got = MatchLessons(ls, "help with Open the firewall")
	if len(got) != 1 || got[0].ID != "c" {
		t.Fatalf("%+v", got)
	}
	if len(MatchLessons(ls, "lol")) != 0 {
		t.Fatal("chatter")
	}
}

func TestLessonsAdminFetch(t *testing.T) {
	var logins int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/api/auth/admin/login"):
			logins++
			_, _ = w.Write([]byte(`{"user":{"email":"bot@example.com","role":"admin"},"token":"jwt-1"}`))
		case strings.Contains(r.URL.Path, "/api/admin/courses/") && strings.Contains(r.URL.Path, "/lessons"):
			if r.Header.Get("Authorization") != "Bearer jwt-1" {
				http.Error(w, `{"error":"Access token required"}`, http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"lessons":[{"id":"L1","title":"The flags that keep it free","order_index":1,"content":"<p>Always-free ARM</p>","is_published":true,"video_url":"https://youtube.com/watch?v=x"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	c.email = "bot@example.com"
	c.pass = "secret"
	got, err := c.Lessons(context.Background(), "course-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "The flags that keep it free" || got[0].Order != 1 {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got[0].Content, "Always-free ARM") {
		t.Fatal(got[0].Content)
	}
	block := LessonBlock("The Free Cloud VM", got, got)
	if !strings.Contains(block, "lesson 1") || !strings.Contains(block, "Always-free ARM") || !strings.Contains(block, "youtube.com") {
		t.Fatal(block)
	}
	if logins != 1 {
		t.Fatalf("logins=%d", logins)
	}
	_, _ = c.Lessons(context.Background(), "course-1")
	if logins != 1 {
		t.Fatalf("cache should skip login, logins=%d", logins)
	}
}

func TestLessonsWithoutAuthAreEmpty(t *testing.T) {
	c := NewClient("http://127.0.0.1:1")
	c.email = ""
	c.pass = ""
	got, err := c.Lessons(context.Background(), "x")
	if err != nil || got != nil {
		t.Fatalf("%v %+v", err, got)
	}
}
