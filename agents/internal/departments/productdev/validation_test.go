package productdev

import (
	"context"
	"strings"
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

func TestHandleHypothesesProposesCard(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Riskiest unknowns first.","structured_data":{"hypotheses":[{"title":"Shops pay","statement":"A workshop will pay TMP for a VIN closeout this month.","importance":5}]}}`}, contract.Request{
		TaskDescription: "Generate hypotheses for Mechazone",
		ContextData:     []byte(`{"action":"generate_hypotheses","project_slug":"mechazone"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["save_memory"] != true || resp.StructuredData["cannot_write_stamp"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if resp.StructuredData["card_hypothesis"] == "" || resp.StructuredData["title"] != "Card · hypothesis" {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if resp.StructuredData["project_slug"] != "mechazone" {
		t.Fatalf("slug %v", resp.StructuredData["project_slug"])
	}
}

func TestHandleEvidenceProposesCard(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Named desks.","structured_data":{"summary":"Ada at XYZ quoted 22% reducing and will send the sheet."}}`}, contract.Request{
		TaskDescription: "Analyse evidence: Ada at XYZ Mortgage quoted 22%.",
		ContextData:     []byte(`{"action":"analyse_evidence","project":{"slug":"homegauge"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["save_memory"] != true || !strings.Contains(str(resp.StructuredData["card_evidence"]), "Ada") {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if resp.StructuredData["project_slug"] != "homegauge" {
		t.Fatalf("slug %v", resp.StructuredData["project_slug"])
	}
}

func TestHandleDecisionDoesNotWriteCard(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Keep testing.","structured_data":{"decision":"CONTINUE_TESTING"}}`}, contract.Request{
		TaskDescription: "GO / PIVOT / KILL for Mechazone",
		ContextData:     []byte(`{"action":"generate_decision_report"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["save_memory"] == true {
		t.Fatal("decision must not write the Card")
	}
	if resp.StructuredData["needs_human"] != true || resp.StructuredData["cannot_write_stamp"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
}

func TestFlattenMissionObject(t *testing.T) {
	m, due := flattenMission(map[string]any{
		"mission": map[string]any{"title": "Three Ikeja desks", "objective": "Get a named officer quote", "due": "2026-10-31"},
	})
	if !strings.Contains(m, "Three Ikeja desks") || due != "2026-10-31" {
		t.Fatalf("%q %q", m, due)
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
