package community

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

func TestAccountsDeniedFlag(t *testing.T) {
	if !accountsDenied(map[string]any{"accounts_denied": true}) {
		t.Fatal("bool true")
	}
	if accountsDenied(map[string]any{"accounts_denied": false}) {
		t.Fatal("bool false")
	}
	if !accountsDenied(map[string]any{"accounts_denied": "true"}) {
		t.Fatal("string true")
	}
	if accountsDenied(map[string]any{}) {
		t.Fatal("missing")
	}
}

func TestAccessDeniedSkipsLLMAndHidesPayload(t *testing.T) {
	resp, err := Handle(context.Background(), nil, contract.Request{
		TaskDescription: "I paid 999999 to Acme Consulting",
		ContextData:     []byte(`{"accounts_denied":true,"chat_kind":"dm"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.StatusSuccess {
		t.Fatalf("status=%s", resp.Status)
	}
	if resp.StructuredData["task_type"] != "access_denied" {
		t.Fatalf("task_type=%v", resp.StructuredData["task_type"])
	}
	if strings.Contains(resp.OutputText, "999999") || strings.Contains(resp.OutputText, "Acme") {
		t.Fatalf("payload leaked: %s", resp.OutputText)
	}
	if !strings.Contains(resp.OutputText, "not authorized") {
		t.Fatalf("missing refuse: %s", resp.OutputText)
	}
	if !strings.Contains(resp.OutputText, "books") {
		t.Fatalf("missing books desk: %s", resp.OutputText)
	}
}

func TestHandleFAQ(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"We meet Thursdays.","structured_data":{"task_type":"faq","speak":true}}`}, contract.Request{
		TaskDescription: "When is the next session?",
		ContextData:     []byte(`{"channel":"whatsapp","chat_kind":"group"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "faq" || resp.OutputText == "" {
		t.Fatalf("%+v", resp)
	}
	if resp.StructuredData["record_lead"] != false {
		t.Fatalf("community must not record a lead: %+v", resp.StructuredData)
	}
}

func TestSilentClearsReply(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"lol","structured_data":{"task_type":"silent","speak":false}}`}, contract.Request{
		TaskDescription: "lol",
		ContextData:     []byte(`{"channel":"whatsapp","chat_kind":"group"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OutputText != "" || resp.StructuredData["speak"] != false {
		t.Fatalf("%+v", resp)
	}
}

func TestMandateInjectsCatalog(t *testing.T) {
	rec := &recLLM{text: `{"status":"success","output_text":"Which Mini are you on?","structured_data":{"task_type":"progress","speak":true}}`}
	_, err := Handle(context.Background(), rec, contract.Request{
		TaskDescription: "I just finished lesson 3",
		ContextData: []byte(`{
			"channel":"whatsapp","chat_kind":"group",
			"mandate":{"slug":"lms","title":"Academy LMS","site_url":"https://lms.themobileprof.com","catalog":"Mini, Path, Professional"}
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rec.last.System, "lms.themobileprof.com") || !strings.Contains(rec.last.System, "Mini, Path, Professional") {
		t.Fatalf("system missing mandate:\n%s", rec.last.System)
	}
}

func TestGroupWithoutMandate(t *testing.T) {
	if !strings.Contains(overlayFor(map[string]any{"chat_kind": "group"}), "no mandate") {
		t.Fatal("unmatched group")
	}
	if overlayFor(map[string]any{"chat_kind": "dm"}) != "" {
		t.Fatal("dm has no unmatched overlay")
	}
}

func TestLiveCatalogOverlay(t *testing.T) {
	got := overlayFor(map[string]any{"chat_kind": "group", "catalog_live": "Live LMS catalog\n- Starting Termux"})
	if !strings.Contains(got, "Starting Termux") {
		t.Fatal(got)
	}
	if strings.Contains(got, "no mandate") {
		t.Fatal("live catalog should skip unmatched overlay")
	}
}

func TestLessonLiveOverlay(t *testing.T) {
	got := overlayFor(map[string]any{
		"chat_kind":    "group",
		"catalog_live": "Live LMS catalog",
		"lesson_live":  "LMS lesson bodies for The Free Cloud VM\n- lesson 1: The flags that keep it free",
	})
	if !strings.Contains(got, "The flags that keep it free") {
		t.Fatal(got)
	}
}

func TestWorkshopLiveOverlay(t *testing.T) {
	got := overlayFor(map[string]any{
		"chat_kind":     "group",
		"workshop_live": "LMS workshop for lesson 2 \"Linux Navigation\" (3 steps). Restate what the current prompt expects",
	})
	if !strings.Contains(got, "Restate what the current prompt expects") {
		t.Fatal(got)
	}
	if !strings.Contains(systemPrompt, "workshop_help") || !strings.Contains(systemPrompt, "Do NOT lecture the concept") {
		t.Fatal("system prompt missing workshop_help")
	}
}

func TestConversationAsksOneThing(t *testing.T) {
	if !strings.Contains(systemPrompt, "One question") || !strings.Contains(systemPrompt, "never ask for course, lesson, and step in the same message") {
		t.Fatal(systemPrompt)
	}
	got := overlayFor(map[string]any{
		"chat_kind": "dm",
		"student":   map[string]any{"course_title": "Linux Navigation", "lesson_note": "pwd"},
		"gather":    map[string]any{"missing": "step", "course": "Linux Navigation", "lesson": "pwd"},
	})
	if !strings.Contains(got, "Ask ONLY for this next: step") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "course=Linux Navigation") {
		t.Fatal(got)
	}
}

func TestAnsweredEnquiryStopsFollowUp(t *testing.T) {
	if !strings.Contains(systemPrompt, "Do not ask a follow-up") || !strings.Contains(systemPrompt, "mini-courses/scholarship") {
		t.Fatal(systemPrompt)
	}
	got := overlayFor(map[string]any{
		"chat_kind": "group",
		"gather":    map[string]any{"missing": "", "course": "Linux Navigation", "lesson": "pwd"},
	})
	if !strings.Contains(got, "Do not ask a follow-up") {
		t.Fatal(got)
	}
	if strings.Contains(got, "Ask ONLY") {
		t.Fatal(got)
	}
}

func TestHandleDropsFollowUpAfterAnswer(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"pwd prints the working directory. Which lesson is next?","structured_data":{"task_type":"faq","speak":true}}`}, contract.Request{
		TaskDescription: "what does pwd do",
		ContextData:     []byte(`{"channel":"telegram","chat_kind":"group"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.OutputText, "?") {
		t.Fatalf("follow-up leaked: %s", resp.OutputText)
	}
	if !strings.Contains(resp.OutputText, "pwd prints") {
		t.Fatalf("%s", resp.OutputText)
	}
}

func TestHandleKeepsGatherQuestion(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Which step are you on?","structured_data":{"task_type":"workshop_help","speak":true}}`}, contract.Request{
		TaskDescription: "I am stuck on the workshop",
		ContextData:     []byte(`{"channel":"telegram","chat_kind":"group","gather":{"missing":"step","course":"Linux Navigation"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.OutputText, "Which step") {
		t.Fatalf("%s", resp.OutputText)
	}
}

func TestTelegramWelcomePointsAtScholarship(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Welcome Ada. Which course are you on?","structured_data":{"task_type":"welcome","speak":true}}`}, contract.Request{
		TaskDescription: "Welcome Ada to the Academy Telegram room.",
		ContextData:     []byte(`{"channel":"telegram","chat_kind":"group"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.OutputText, "Welcome Ada") || !strings.Contains(resp.OutputText, scholarshipMiniCourseURL) {
		t.Fatalf("%s", resp.OutputText)
	}
	if strings.Contains(resp.OutputText, "Which course") {
		t.Fatalf("welcome still asked a course: %s", resp.OutputText)
	}
}

func TestScholarshipLiveOverlay(t *testing.T) {
	got := overlayFor(map[string]any{
		"chat_kind":        "group",
		"scholarship_live": "Live Scholarship season (LMS). Never invent a discount code.",
	})
	if !strings.Contains(got, "Never invent a discount code") {
		t.Fatal(got)
	}
}

func TestScholarshipNudgeSilentWhenNotApproaching(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"should not send","structured_data":{"task_type":"scholarship_nudge","speak":true}}`}, contract.Request{
		TaskDescription: "Daily scholarship check-in.",
		ContextData:     []byte(`{"channel":"telegram","chat_kind":"group","action":"scholarship_nudge","scholarship_approaching":false}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OutputText != "" || resp.StructuredData["speak"] != false {
		t.Fatalf("%+v", resp)
	}
}

func TestScholarshipNudgeFallback(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"","structured_data":{"task_type":"silent","speak":false}}`}, contract.Request{
		TaskDescription: "Daily scholarship check-in.",
		ContextData:     []byte(`{"channel":"telegram","chat_kind":"group","action":"scholarship_nudge","scholarship_approaching":true,"scholarship_live":"Live Scholarship season"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "scholarship_nudge" || resp.StructuredData["speak"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if !strings.Contains(resp.OutputText, "course rep") || strings.Contains(resp.OutputText, "UNILAG") {
		t.Fatalf("%s", resp.OutputText)
	}
}

func TestWeeklyIntroFallback(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"","structured_data":{"task_type":"silent","speak":false}}`}, contract.Request{
		TaskDescription: "Weekly check-in.",
		ContextData:     []byte(`{"channel":"whatsapp","chat_kind":"group","action":"weekly_intro"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "weekly_intro" || resp.StructuredData["speak"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if !strings.Contains(resp.OutputText, "/cm") || !strings.Contains(resp.OutputText, "working on") {
		t.Fatalf("intro missing how-to-call: %s", resp.OutputText)
	}
}

type recLLM struct {
	text string
	last llm.Request
}

func (s *recLLM) Complete(_ context.Context, req llm.Request) (string, error) {
	s.last = req
	return s.text, nil
}

type stubLLM struct {
	text string
}

func (s stubLLM) Complete(context.Context, llm.Request) (string, error) {
	return s.text, nil
}
