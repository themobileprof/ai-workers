package productdev

import "testing"

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

func TestValidationPromptsEmbedded(t *testing.T) {
	for action := range validationFiles {
		if _, err := validationPrompts.ReadFile("prompts/" + validationFiles[action]); err != nil {
			t.Fatalf("%s: %v", action, err)
		}
	}
}
