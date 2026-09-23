package hr

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
		body, err := templateFS.ReadFile("templates/" + s.File)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "TheMobileProf Technologies") {
			t.Fatalf("%s missing company name", s.ID)
		}
		if strings.Contains(strings.ToLower(string(body)), "this is legal advice") {
			t.Fatalf("%s claims advice", s.ID)
		}
	}
}

func TestInfer(t *testing.T) {
	a, tmpl := infer(map[string]any{"action": "draft_jd", "template": "intern"}, "whatever")
	if a != "draft_jd" || tmpl != "intern" {
		t.Fatalf("%s %s", a, tmpl)
	}
	a, tmpl = infer(map[string]any{}, "Draft a JD for a field intern on Mechazone")
	if a != "draft_jd" || tmpl != "intern" {
		t.Fatalf("jd intern got %s %s", a, tmpl)
	}
	a, tmpl = infer(map[string]any{}, "I am applying for the intern role. Please find my CV: Lagos, Python.")
	if a != "ingest" {
		t.Fatalf("ingest got %s", a)
	}
	a, tmpl = infer(map[string]any{"action": "score", "template": "contractor"}, "Score this CV against the contractor JD")
	if a != "score" || tmpl != "contractor" {
		t.Fatalf("score got %s %s", a, tmpl)
	}
	a, _ = infer(map[string]any{}, "Why reject Chinedu for the intern role?")
	if a != "explain" {
		t.Fatalf("explain got %s", a)
	}
}

func TestLooksLikeApplication(t *testing.T) {
	if !looksLikeApplication("i am applying for the intern seat") {
		t.Fatal("applying")
	}
	if looksLikeApplication("invoice apex 250000 for a 30-day pilot") {
		t.Fatal("invoice is not an application")
	}
}

func TestHandleStampsCannotHire(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Accept on HR.","structured_data":{"save_role":true}}`}, contract.Request{
		TaskDescription: "Draft a JD for a field intern on Mechazone",
		ContextData:     []byte(`{"action":"draft_jd","template":"intern"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["cannot_hire"] != true || resp.StructuredData["cannot_send"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if resp.StructuredData["action"] != "draft_jd" || resp.StructuredData["template"] != "intern" {
		t.Fatalf("%+v", resp.StructuredData)
	}
}
