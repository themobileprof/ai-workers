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
	if !strings.Contains(resp.OutputText, "HR") {
		t.Fatalf("missing HR desk: %s", resp.OutputText)
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
