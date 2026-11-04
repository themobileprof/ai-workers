package internalops

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

func TestAccountsOnly(t *testing.T) {
	if !accountsOnly(map[string]any{"department": "accounts"}) {
		t.Fatal("department")
	}
	if !accountsOnly(map[string]any{"task_type": "accounts"}) {
		t.Fatal("task_type")
	}
	if accountsOnly(map[string]any{"department": "legal"}) {
		t.Fatal("legal")
	}
}

func TestHandleRoutesAccountsDepartment(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{
		"status":"success",
		"output_text":"Shoprite 10000 NGN",
		"structured_data":{
			"zoho_action":"expense",
			"vendor_name":"Shoprite",
			"base_amount":10000,
			"currency":"NGN",
			"transaction_type":"expense",
			"record_expense":true
		}
	}`}, contract.Request{
		TaskDescription: "I paid Shoprite 10000 for office supplies",
		ContextData:     []byte(`{"department":"accounts"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "accounts" || resp.StructuredData["zoho_action"] != "expense" {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if resp.StructuredData["tax_engine"] != "zoho_books" {
		t.Fatalf("tax_engine %+v", resp.StructuredData)
	}
}

func TestHandleRouteSuggestsDesks(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Delivery should take this check-in. Reply yes to send /today.","structured_data":{"task_type":"route","route_to":["delivery"],"route_task":"done HomeGauge calculator"}}`}, contract.Request{
		TaskDescription: "done the HomeGauge one, still on Academy",
		ContextData:     []byte(`{"channel":"telegram","action":"route","reply_to":"Today on the desk (Samuel):\n- HomeGauge: calculator"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "route" || resp.StructuredData["speak"] != true || resp.StructuredData["cannot_send"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if !strings.Contains(resp.OutputText, "/today") && !strings.Contains(resp.OutputText, "Delivery") {
		t.Fatal(resp.OutputText)
	}
}

func TestHandleLeavesLegalUntouched(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"NDA looks fine.","structured_data":{"task_type":"legal"}}`}, contract.Request{
		TaskDescription: "Review this NDA",
		ContextData:     []byte(`{"department":"internal-ops"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OutputText != "NDA looks fine." {
		t.Fatal(resp.OutputText)
	}
	if _, ok := resp.StructuredData["tax_engine"]; ok {
		t.Fatal("tax overlay on legal")
	}
}
