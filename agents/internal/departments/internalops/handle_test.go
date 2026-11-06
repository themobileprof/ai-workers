package internalops

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
