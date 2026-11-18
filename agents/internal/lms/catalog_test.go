package lms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMatchTitleAndPath(t *testing.T) {
	cat := Catalog{
		Courses: []Course{
			{ID: "1", Title: "Starting Termux", Format: "micro", Price: "0.00", Published: true},
			{ID: "2", Title: "Linux Navigation", Format: "micro", Price: "0.00", Published: true},
			{ID: "3", Title: "Package Management", Format: "micro", Price: "10.00", Published: true},
		},
		Paths: []Path{{Title: "Introduction to Linux Terminal", Micros: []string{"Starting Termux", "Linux Navigation", "Package Management"}}},
	}
	got := Match(cat, "I am stuck on Starting Termux lesson 1")
	if len(got) != 1 || got[0].Title != "Starting Termux" {
		t.Fatalf("%+v", got)
	}
	got = Match(cat, "help with Introduction to Linux Terminal")
	if len(got) != 3 {
		t.Fatalf("path micros %+v", got)
	}
	if len(Match(cat, "lol ok")) != 0 {
		t.Fatal("chatter")
	}
	got = Match(cat, "linux terminal on my phone")
	if len(got) != 3 {
		t.Fatalf("path tokens %+v", got)
	}
}

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/courses":
			_, _ = w.Write([]byte(`{"courses":[{"id":"a","title":"Starting Termux","format":"micro","price":"0.00","isPublished":true,"objectives":"Install Termux"}]}`))
		case "/api/mini-courses":
			_, _ = w.Write([]byte(`{"miniCourses":[{"title":"Introduction to Linux Terminal","microCourses":[{"title":"Starting Termux"}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	cat, err := c.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Courses) != 1 || cat.Courses[0].Title != "Starting Termux" {
		t.Fatalf("%+v", cat.Courses)
	}
	block := PromptBlock(cat, cat.Courses)
	if !strings.Contains(block, "Starting Termux") || !strings.Contains(block, "list_price=0.00") {
		t.Fatal(block)
	}
}
