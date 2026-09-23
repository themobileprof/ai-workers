package departments

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/llm"
)

type stubLLM struct {
	text string
	err  error
}

func (s stubLLM) Complete(context.Context, llm.Request) (string, error) {
	return s.text, s.err
}

func TestDecodeResponseJSONObject(t *testing.T) {
	raw := `{
	  "status": "success",
	  "output_text": "VAT is 7.5%.",
	  "structured_data": {"vat_rate": 0.075},
	  "task_type": "accounts"
	}`
	got, err := decodeResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "success" || got.OutputText != "VAT is 7.5%." {
		t.Fatalf("unexpected contract: %+v", got)
	}
	if got.StructuredData["task_type"] != "accounts" {
		t.Fatalf("task_type not merged: %+v", got.StructuredData)
	}
}

func TestDecodeResponseFenced(t *testing.T) {
	raw := "```json\n{\"status\":\"failed\",\"output_text\":\"no\",\"structured_data\":{}}\n```"
	got, err := decodeResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestDecodeResponseUnknownStatusAndNilData(t *testing.T) {
	got, err := decodeResponse(`{"status":"maybe","output_text":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" || got.StructuredData == nil {
		t.Fatalf("%+v", got)
	}
}

func TestRun(t *testing.T) {
	resp, err := Run(context.Background(), stubLLM{text: `{"status":"success","output_text":"hi","structured_data":{"task_type":"faq"}}`}, "sys", "user")
	if err != nil || resp.OutputText != "hi" || resp.StructuredData["task_type"] != "faq" {
		t.Fatalf("%+v %v", resp, err)
	}
	resp, err = Run(context.Background(), stubLLM{err: errors.New("timeout")}, "sys", "user")
	if err == nil || resp.Status != "failed" || !strings.Contains(resp.OutputText, "timeout") {
		t.Fatalf("err path %+v %v", resp, err)
	}
	resp, err = Run(context.Background(), stubLLM{text: "not json at all"}, "sys", "user")
	if err != nil || resp.Status != "failed" || resp.StructuredData["parse_error"] == nil {
		t.Fatalf("parse %+v %v", resp, err)
	}
}

func TestUserPrompt(t *testing.T) {
	got := UserPrompt("Draft an NDA", map[string]any{"channel": "desk"})
	if !strings.Contains(got, "Draft an NDA") || !strings.Contains(got, `"channel": "desk"`) {
		t.Fatal(got)
	}
}
