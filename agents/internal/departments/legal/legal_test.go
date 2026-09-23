package legal

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

func TestSpecsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Specs() {
		if s.ID == "" || seen[s.ID] {
			t.Fatalf("id %q", s.ID)
		}
		seen[s.ID] = true
		if _, err := templateFS.ReadFile("templates/" + s.File); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(s.Help, "TMP") && s.ID != "nda" && s.ID != "customer_terms" {
			// help may omit TMP; files must name the company
		}
		body, _ := templateFS.ReadFile("templates/" + s.File)
		if !strings.Contains(string(body), "TheMobileProf Technologies") {
			t.Fatalf("%s missing company name", s.ID)
		}
		if strings.Contains(strings.ToLower(string(body)), "this is legal advice") {
			t.Fatalf("%s claims to be advice", s.ID)
		}
	}
}

func TestInfer(t *testing.T) {
	a, tmpl := infer(map[string]any{"template": "nda", "action": "draft"}, "whatever")
	if a != "draft" || tmpl != "nda" {
		t.Fatalf("%s %s", a, tmpl)
	}
	a, tmpl = infer(map[string]any{}, "Flag predatory clauses in this contractor NDA: paste")
	if a != "review" || tmpl != "nda" {
		t.Fatalf("review nda got %s %s", a, tmpl)
	}
	a, tmpl = infer(map[string]any{}, "Draft a contractor agreement for Ada on Mechazone")
	if a != "draft" || tmpl != "contractor" {
		t.Fatalf("got %s %s", a, tmpl)
	}
}

func TestInferReviewWithoutTemplate(t *testing.T) {
	a, tmpl := infer(map[string]any{}, "Is this clause predatory? paste: the vendor owns all data forever")
	if a != "review" {
		t.Fatalf("action %s", a)
	}
	if tmpl != "" {
		t.Fatalf("tmpl %s", tmpl)
	}
}

func TestInferWatch(t *testing.T) {
	a, tmpl := infer(map[string]any{"action": "watch"}, "Draft a contractor agreement for Ada")
	if a != "watch" || tmpl != "" {
		t.Fatalf("watch must not draft, got %s %s", a, tmpl)
	}
}

func TestWatchWorthy(t *testing.T) {
	if !watchWorthy("Ada is joining as cofounder of Mechazone", nil) {
		t.Fatal("cofounder")
	}
	if !watchWorthy("Hire an intern on PAYE next month", nil) {
		t.Fatal("hire")
	}
	if watchWorthy("Paid Shoprite 1200 NGN for office snacks", nil) {
		t.Fatal("retail receipt should stay silent")
	}
	if !watchWorthy("place on journey", map[string]any{"primary_zoho_action": "invoice"}) {
		t.Fatal("invoice")
	}
}

func TestWatchUnworthySkipsLLM(t *testing.T) {
	resp, err := Handle(nil, nil, contract.Request{
		TaskDescription: "Paid Shoprite 1200 NGN for office snacks",
		ContextData:     []byte(`{"action":"watch","channel":"telegram"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.StatusSuccess {
		t.Fatalf("status %s", resp.Status)
	}
	if resp.OutputText != "" {
		t.Fatalf("spoke: %s", resp.OutputText)
	}
	if boolish(resp.StructuredData["speak"]) {
		t.Fatal("speak")
	}
}

func TestHandleDraftCannotSend(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Accept on Legal.","structured_data":{"save_draft":true}}`}, contract.Request{
		TaskDescription: "Draft an NDA for Ada on Mechazone",
		ContextData:     []byte(`{"action":"draft","template":"nda"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["cannot_send"] != true || resp.StructuredData["action"] != "draft" {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if resp.StructuredData["template"] != "nda" || resp.StructuredData["task_type"] != "legal" {
		t.Fatalf("%+v", resp.StructuredData)
	}
}
