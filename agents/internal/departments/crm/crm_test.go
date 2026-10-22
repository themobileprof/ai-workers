package crm

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

type stubLLM struct {
	text string
	got  llm.Request
}

func (s *stubLLM) Complete(_ context.Context, req llm.Request) (string, error) {
	s.got = req
	return s.text, nil
}

func TestHandle(t *testing.T) {
	stub := &stubLLM{text: `{"status":"success","output_text":"Qualify Ada.","structured_data":{"task_type":"qualify_lead","record_lead":true}}`}
	resp, err := Handle(context.Background(), stub, contract.Request{
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

func TestHandleReadsEmailCorrespondence(t *testing.T) {
	stub := &stubLLM{text: `{"status":"success","output_text":"Follow up Ada.","structured_data":{"task_type":"follow_up","record_lead":true,"notes":"Asked for a 30-day pilot; draft offers a Lagos demo; next is a named officer."}}`}
	resp, err := Handle(context.Background(), stub, contract.Request{
		TaskDescription: "Inbound email to info@. Read the correspondence.",
		ContextData: []byte(`{
			"channel":"email",
			"from":"ada@apex.ng",
			"subject":"Pilot",
			"correspondence":"We want a 30-day books pilot starting next month.",
			"draft_reply":"Happy to walk Apex through a Lagos demo.",
			"existing_contact":{"notes":"2026-10-01 email: first hello"}
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.StatusSuccess || resp.StructuredData["task_type"] != "follow_up" {
		t.Fatalf("%+v", resp)
	}
	if !strings.Contains(stub.got.System, "client story") {
		t.Fatal("crm prompt must tell the model to read email correspondence")
	}
	if !strings.Contains(stub.got.System, "Do not draft a reply") {
		t.Fatal("inbound email must not draft a reply")
	}
	if !strings.Contains(stub.got.System, "file_thread") {
		t.Fatal("inbound email must file a thread summary")
	}
	if !strings.Contains(stub.got.Messages[0].Content, "We want a 30-day books pilot") {
		t.Fatal("crm user prompt must include the letter")
	}
	if !strings.Contains(stub.got.Messages[0].Content, "first hello") {
		t.Fatal("crm user prompt must include prior Books notes")
	}
}
