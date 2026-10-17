package delivery

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

type stubLLM struct{ text string }

func (s stubLLM) Complete(context.Context, llm.Request) (string, error) { return s.text, nil }

func TestAccessDenied(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{}, contract.Request{
		TaskDescription: "What is Finchest's pipeline?",
		ContextData:     []byte(`{"access_denied":true,"project_slug":"finchest"}`),
	})
	if err != nil || resp.StructuredData["task_type"] != "access_denied" {
		t.Fatalf("%v %+v", err, resp)
	}
	if strings.Contains(resp.OutputText, "Finchest") {
		t.Fatal(resp.OutputText)
	}
}

func TestSilentChatter(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success"}`}, contract.Request{
		TaskDescription: "ok",
		ContextData:     []byte(`{}`),
	})
	if err != nil || resp.StructuredData["task_type"] != "silent" {
		t.Fatalf("%v %+v", err, resp)
	}
}

func TestDigestSkipsLLM(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: "SHOULD NOT RUN"}, contract.Request{
		TaskDescription: "unused",
		ContextData:     []byte(`{"action":"digest","digest_text":"Today: Academy lesson editor"}`),
	})
	if err != nil || resp.OutputText != "Today: Academy lesson editor" {
		t.Fatalf("%v %s", err, resp.OutputText)
	}
}

