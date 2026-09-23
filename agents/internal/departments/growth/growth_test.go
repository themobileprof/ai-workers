package growth

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

func TestHandle(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Thursday works.","structured_data":{"task_type":"sales_support"}}`}, contract.Request{
		TaskDescription: "When can we start the pilot?",
		ContextData:     []byte(`{"channel":"whatsapp"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.StatusSuccess || resp.StructuredData["task_type"] != "sales_support" {
		t.Fatalf("%+v", resp)
	}
}
