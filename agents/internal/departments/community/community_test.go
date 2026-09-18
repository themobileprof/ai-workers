package community

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func TestAccountsDeniedFlag(t *testing.T) {
	if !accountsDenied(map[string]any{"accounts_denied": true}) {
		t.Fatal("bool true")
	}
	if accountsDenied(map[string]any{"accounts_denied": false}) {
		t.Fatal("bool false")
	}
	if !accountsDenied(map[string]any{"accounts_denied": "true"}) {
		t.Fatal("string true")
	}
	if accountsDenied(map[string]any{}) {
		t.Fatal("missing")
	}
}

func TestAccessDeniedSkipsLLMAndHidesPayload(t *testing.T) {
	resp, err := Handle(context.Background(), nil, contract.Request{
		TaskDescription: "I paid 999999 to Acme Consulting",
		ContextData:     []byte(`{"accounts_denied":true,"chat_kind":"dm"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.StatusSuccess {
		t.Fatalf("status=%s", resp.Status)
	}
	if resp.StructuredData["task_type"] != "access_denied" {
		t.Fatalf("task_type=%v", resp.StructuredData["task_type"])
	}
	if strings.Contains(resp.OutputText, "999999") || strings.Contains(resp.OutputText, "Acme") {
		t.Fatalf("payload leaked: %s", resp.OutputText)
	}
	if !strings.Contains(resp.OutputText, "not authorized") {
		t.Fatalf("missing refuse: %s", resp.OutputText)
	}
	if !strings.Contains(resp.OutputText, "HR") {
		t.Fatalf("missing HR desk: %s", resp.OutputText)
	}
}
