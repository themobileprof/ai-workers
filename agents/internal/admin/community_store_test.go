package admin

import (
	"strings"
	"testing"
)

func TestCommunitySlug(t *testing.T) {
	if communitySlug("Academy LMS") != "academy-lms" {
		t.Fatal(communitySlug("Academy LMS"))
	}
}

func TestMandateJSON(t *testing.T) {
	m := CommunityMandate{Slug: "lms", Title: "Academy LMS", SiteURL: "https://lms.themobileprof.com", Active: true}
	got := m.JSON()
	if got["slug"] != "lms" || got["site_url"] != "https://lms.themobileprof.com" {
		t.Fatalf("%+v", got)
	}
}

func TestCommunityTemplateRenders(t *testing.T) {
	srv, err := New(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	u := &User{ID: 1, Name: "Sam", Role: "owner"}
	data := pageData{Title: "Community", Nav: "community", User: u, NavDesks: []Desk{{Slug: "community", Title: "Community"}}}
	var buf strings.Builder
	if err := srv.pages["community"].ExecuteTemplate(&buf, "layout.html", data); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "Community mandates") || !strings.Contains(got, "New mandate") {
		t.Fatal(got)
	}
	m := CommunityMandate{ID: 1, Title: "Academy LMS", SiteURL: "https://lms.themobileprof.com", Brief: "Host the room.", Catalog: "Mini, Path, Professional", Active: true}
	data.Mandate = &m
	data.Mandates = []CommunityMandate{m}
	buf.Reset()
	if err := srv.pages["community"].ExecuteTemplate(&buf, "layout.html", data); err != nil {
		t.Fatal(err)
	}
	got = buf.String()
	if !strings.Contains(got, "Academy LMS") || !strings.Contains(got, "lms.themobileprof.com") {
		t.Fatal(got)
	}
}

func TestLMSCatalogMentionsTiers(t *testing.T) {
	for _, part := range []string{"Mini", "Path", "Professional", "lms.themobileprof.com", "Clio", "Termux Essentials"} {
		if !strings.Contains(lmsMandateCatalog, part) {
			t.Fatal(part)
		}
	}
	if !strings.Contains(lmsMandateBrief, "escalate") || !strings.Contains(lmsMandateBrief, "lesson") || !strings.Contains(lmsMandateBrief, "/cm") {
		t.Fatal("seed brief")
	}
}
