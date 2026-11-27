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

func TestReportRecapSkipsLLM(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: "SHOULD NOT RUN"}, contract.Request{
		TaskDescription: "What happened on the bets. Log it in your own words.",
		ContextData:     []byte(`{"action":"report","actor":{"id":1,"name":"Samuel"},"bets":[{"name":"Academy","items":[{"id":9,"title":"lesson editor","status":"open","assignee_id":1}]}]}`),
	})
	if err != nil || resp.StructuredData["task_type"] != "report" {
		t.Fatalf("%v %+v", err, resp)
	}
	if strings.Contains(resp.OutputText, "SHOULD NOT RUN") {
		t.Fatal(resp.OutputText)
	}
	if !strings.Contains(resp.OutputText, "lesson editor") || !strings.Contains(resp.OutputText, "own words") {
		t.Fatal(resp.OutputText)
	}
	if strings.Contains(strings.ToLower(resp.OutputText), "done 1") {
		t.Fatal(resp.OutputText)
	}
}

func TestHandleReportUsesClerk(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Logged Ikorodu General Hospital.","structured_data":{"task_type":"report","summary":"ANC visit","log":"We went to Ikorodu General Hospital.","contacts":[{"name":"HOD ANC","company":"Ikorodu General Hospital","record_lead":true}],"cannot_spend":true}}`}, contract.Request{
		TaskDescription: "We went to Ikorodu General Hospital yesterday, had a 5 minutes speech with the pregnant women at the ANC, registered names, and left fliers.",
		ContextData:     []byte(`{"action":"report","actor":{"id":1,"name":"Samuel"}}`),
	})
	if err != nil || resp.StructuredData["task_type"] != "report" {
		t.Fatalf("%v %+v", err, resp)
	}
	if resp.StructuredData["cannot_spend"] != true || resp.StructuredData["cannot_send"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if _, ok := resp.StructuredData["zoho_action"]; ok {
		t.Fatal("mixed report must not post Books")
	}
	if !strings.Contains(resp.OutputText, "Ikorodu") {
		t.Fatal(resp.OutputText)
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
