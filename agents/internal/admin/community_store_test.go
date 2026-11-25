package admin

import (
	"strings"
	"testing"
	"time"
)

func TestCollapsePingMandatesOneAcademyRoom(t *testing.T) {
	old := CommunityMandate{ID: 1, Slug: "lms", Title: "Academy", TelegramChatID: "-5459918440", Active: true}
	fresh := CommunityMandate{ID: 2, Slug: "academy-lms", Title: "Community", TelegramChatID: "-1003968683203", Active: true, UpdatedAt: old.UpdatedAt.Add(time.Hour)}
	got := collapsePingMandates([]CommunityMandate{old, fresh})
	if len(got) != 1 || got[0].TelegramChatID != "-1003968683203" {
		t.Fatalf("%+v", got)
	}
}

func TestCollapsePingMandatesKeepsOtherBets(t *testing.T) {
	a := CommunityMandate{Slug: "lms", TelegramChatID: "-1001", Active: true}
	b := CommunityMandate{Slug: "homegauge", TelegramChatID: "-1002", Active: true}
	got := collapsePingMandates([]CommunityMandate{a, b})
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestPickTelegramChatPrefersSupergroup(t *testing.T) {
	if pickTelegramChat([]string{"-5459918440", "-1003968683203"}) != "-1003968683203" {
		t.Fatal(pickTelegramChat([]string{"-5459918440", "-1003968683203"}))
	}
}

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
	if got["project_slug"] != "academy" {
		t.Fatalf("lms mandate must pin academy, got %+v", got)
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
	if !strings.Contains(got, "Academy Telegram") || !strings.Contains(got, "New mandate") || !strings.Contains(got, "Students") {
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
	if !strings.Contains(lmsMandateBrief, "escalate") || !strings.Contains(lmsMandateBrief, "lesson") || !strings.Contains(lmsMandateBrief, "/cm") || !strings.Contains(lmsMandateBrief, "Telegram") || !strings.Contains(lmsMandateBrief, "workshop") || !strings.Contains(lmsMandateBrief, "scholarship") || !strings.Contains(lmsMandateBrief, "airtime") || !strings.Contains(lmsMandateBrief, "quiz") || !strings.Contains(lmsMandateBrief, "quiet class") {
		t.Fatal("seed brief")
	}
}
