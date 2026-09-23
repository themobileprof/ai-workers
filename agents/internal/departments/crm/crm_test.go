package crm

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
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Qualify Ada.","structured_data":{"task_type":"qualify_lead","record_lead":true}}`}, contract.Request{
		TaskDescription: "Ada from Apex wants a 30-day books pilot. Email ada@apex.ng",
		ContextData:     []byte(`{"channel":"telegram"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.StatusSuccess || resp.StructuredData["task_type"] != "qualify_lead" {
		t.Fatalf("%+v", resp)
	}
}
