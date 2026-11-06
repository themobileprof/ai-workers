package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func TestBooksWriteBodyAccountsExpense(t *testing.T) {
	body, ok := booksWriteBody("accounts", contract.Response{
		Status: contract.StatusSuccess,
		StructuredData: map[string]any{
			"task_type":   "accounts",
			"zoho_action": "expense",
			"amount":      12500,
			"vendor_name": "Shoprite",
		},
	})
	if !ok || body["action"] != "expense" {
		t.Fatalf("%v ok=%v", body, ok)
	}
}

func TestBooksWriteBodyOpsNonAccountsSkipped(t *testing.T) {
	_, ok := booksWriteBody("internal-ops", contract.Response{
		Status:         contract.StatusSuccess,
		StructuredData: map[string]any{"task_type": "legal", "zoho_action": "expense"},
	})
	if ok {
		t.Fatal("legal via /ops must not post Books")
	}
}

func TestBooksWriteBodyRecordExpenseFallback(t *testing.T) {
	body, ok := booksWriteBody("accounts", contract.Response{
		Status: contract.StatusSuccess,
		StructuredData: map[string]any{
			"task_type":      "accounts",
			"zoho_action":    "none",
			"record_expense": true,
			"amount":         10,
		},
	})
	if !ok || body["action"] != "expense" {
		t.Fatalf("%v ok=%v", body, ok)
	}
}

func TestBooksWriteBodyNoneSkipped(t *testing.T) {
	_, ok := booksWriteBody("accounts", contract.Response{
		Status:         contract.StatusSuccess,
		StructuredData: map[string]any{"zoho_action": "none"},
	})
	if ok {
		t.Fatal("none")
	}
}

func TestBooksWriteBodyWhatsAppNotHere(t *testing.T) {
	_, ok := booksWriteBody("growth", contract.Response{
		Status:         contract.StatusSuccess,
		StructuredData: map[string]any{"zoho_action": "expense"},
	})
	if ok {
		t.Fatal("growth")
	}
}

func TestExtractField(t *testing.T) {
	got := extractField(map[string]any{"audit_text": "Recorded in Zoho Books as expense 1"}, "audit_text")
	if got != "Recorded in Zoho Books as expense 1" {
		t.Fatal(got)
	}
	got = extractField([]any{map[string]any{"json": map[string]any{"text": "Sent to a@b.com."}}}, "text")
	if got != "Sent to a@b.com." {
		t.Fatal(got)
	}
}

func TestN8nClientPostsTokenAndPath(t *testing.T) {
	var gotPath, gotTok, gotAction string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTok = r.Header.Get("X-Internal-Token")
		raw, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		gotAction, _ = m["action"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"audit_text":"Recorded in Zoho Books as expense 9"}`))
	}))
	defer srv.Close()

	c := newN8nClient(srv.URL, "tok")
	out, err := c.post(context.Background(), n8nBooksPath, map[string]any{"action": "expense"})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != n8nBooksPath || gotTok != "tok" || gotAction != "expense" {
		t.Fatalf("path=%s tok=%s action=%s", gotPath, gotTok, gotAction)
	}
	if extractField(out, "audit_text") != "Recorded in Zoho Books as expense 9" {
		t.Fatal(out)
	}
}

func TestN8nClientHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()
	c := newN8nClient(srv.URL, "tok")
	_, err := c.post(context.Background(), n8nBooksPath, map[string]any{"action": "lookup"})
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("%v", err)
	}
}
