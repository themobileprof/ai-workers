package productdev

import (
	"context"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

type stubLLM struct {
	text string
}

func (s stubLLM) Complete(context.Context, llm.Request) (string, error) {
	return s.text, nil
}

func TestValidationActionFromContext(t *testing.T) {
	action, ok := validationAction(map[string]any{"action": "analyse_evidence"}, "whatever")
	if !ok || action != "analyse_evidence" {
		t.Fatalf("got %q ok=%v", action, ok)
	}
}

func TestValidationActionFromProductDescription(t *testing.T) {
	action, ok := validationAction(map[string]any{"product_description": "POS for mechanics"}, "Validate this idea")
	if !ok || action != "generate_hypotheses" {
		t.Fatalf("got %q ok=%v", action, ok)
	}
}

func TestValidationActionIgnoredForQA(t *testing.T) {
	_, ok := validationAction(map[string]any{"diff": "--- a/main.go"}, "Review this pull request for secrets")
	if ok {
		t.Fatal("QA task should not route to validation")
	}
}

func TestValidationActionPlaceOnJourney(t *testing.T) {
	action, ok := validationAction(map[string]any{"action": "place_on_journey"}, "whatever")
	if !ok || action != "place_on_journey" {
		t.Fatalf("got %q ok=%v", action, ok)
	}
	action, ok = validationAction(map[string]any{"project": map[string]any{"slug": "mechazone"}}, "Place this bet")
	if !ok || action != "place_on_journey" {
		t.Fatalf("project context got %q ok=%v", action, ok)
	}
}

func TestValidationPromptsEmbedded(t *testing.T) {
	for action := range validationFiles {
		if _, err := validationPrompts.ReadFile("prompts/" + validationFiles[action]); err != nil {
			t.Fatalf("%s: %v", action, err)
		}
	}
}

func TestHandleValidationStampsProposal(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Named desks, not vibes.","structured_data":{}}`}, contract.Request{
		TaskDescription: "Analyse evidence: Ada at XYZ Mortgage quoted 22%.",
		ContextData:     []byte(`{"action":"analyse_evidence"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "product_validation" || resp.StructuredData["action"] != "analyse_evidence" {
		t.Fatalf("%+v", resp.StructuredData)
	}
}

func TestHandleQAUsesLabPrompt(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Redact the token.","structured_data":{"task_type":"product_ops_qa"}}`}, contract.Request{
		TaskDescription: "Review this pull request for secrets",
		ContextData:     []byte(`{"diff":"--- a/main.go"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "product_ops_qa" {
		t.Fatalf("%+v", resp.StructuredData)
	}
}
