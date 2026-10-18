package admin

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMustGuideFiles(t *testing.T) {
	if err := mustGuideFiles(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readGuideFile("handbook.md"), "/approve") {
		t.Fatal("handbook must mention /approve")
	}
	if !strings.Contains(readGuideFile("faq.md"), "People") {
		t.Fatal("faq")
	}
}

func TestParseFAQ(t *testing.T) {
	got := parseFAQ(readGuideFile("faq.md"))
	if len(got) < 8 {
		t.Fatalf("entries %d", len(got))
	}
	hit, n := bestFAQ("how do I add someone to the office", got)
	if n < 2 || !strings.Contains(strings.ToLower(hit.A), "people") {
		t.Fatalf("hit=%q n=%d", hit.A, n)
	}
	event, n := bestFAQ("how do I track a training at UNILAG on the academy project", got)
	if n < 2 || !strings.Contains(strings.ToLower(event.A), "dates") {
		t.Fatalf("event faq=%q n=%d", event.A, n)
	}
}

func TestStripHTML(t *testing.T) {
	got := stripHTML(`<p class="lede">Hello <strong>Ada</strong></p><script>alert(1)</script>`)
	if got != "Hello Ada" {
		t.Fatalf("%q", got)
	}
}

func TestPickPlaybooksBoostsPage(t *testing.T) {
	hits := pickPlaybooks("stamp a task", "/admin/delivery", 3)
	if len(hits) == 0 || hits[0].Meta.Slug != "delivery" {
		t.Fatalf("%+v", hits)
	}
}

func TestParseGuideModel(t *testing.T) {
	got := parseGuideModel("```json\n{\"answer\":\"Open People\",\"unsure\":false,\"hrefs\":[\"/admin/users\",\"https://evil\"]}\n```")
	if got.Answer != "Open People" || got.Unsure || len(got.Hrefs) != 1 || got.Hrefs[0] != "/admin/users" {
		t.Fatalf("%+v", got)
	}
	plain := parseGuideModel("Just open Docs.")
	if plain.Answer != "Just open Docs." {
		t.Fatalf("%q", plain.Answer)
	}
}

func TestFaqFallbackUnsure(t *testing.T) {
	got := faqFallback("what is the wifi password for the lagos office kitchen")
	if !got.Unsure || got.Source != "none" {
		t.Fatalf("%+v", got)
	}
}

func TestAnswerGuideWithoutLLMUsesFAQ(t *testing.T) {
	var s *Server
	got := s.answerGuide(context.Background(), guideInput{
		User:     User{Name: "Ada", Role: "assistant"},
		Question: "How do I add someone to the office?",
	})
	if got.Unsure || !strings.Contains(strings.ToLower(got.Answer), "people") {
		t.Fatalf("%+v", got)
	}
	event := s.answerGuide(context.Background(), guideInput{
		User:     User{Name: "Ada", Role: "owner"},
		Question: "How do I track a UNILAG training on TheMobileProf Academy until delivery?",
	})
	if event.Unsure || !strings.Contains(strings.ToLower(event.Answer), "dates") {
		t.Fatalf("%+v", event)
	}
}

func TestExportGuideAsksMarkdown(t *testing.T) {
	raw := ExportGuideAsksMarkdown([]GuideAsk{{
		UserName:  "Ada",
		Question:  "How do I add someone?",
		Answer:    "Open People.",
		Page:      "/admin/users",
		Unsure:    true,
		CreatedAt: time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC),
	}})
	if !strings.Contains(raw, "UNSURE") || !strings.Contains(raw, "Ada") || !strings.Contains(raw, "faq.md") {
		t.Fatalf("%s", raw)
	}
}

func TestRuntimeDeskMapHasPrefixes(t *testing.T) {
	got := runtimeDeskMap()
	for _, want := range []string{"/legal", "/seo", "/today", "/accounts"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in %s", want, got)
		}
	}
}
