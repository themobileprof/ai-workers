package seo

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

type stubLLM struct{ text string }

func (s stubLLM) Complete(context.Context, llm.Request) (string, error) { return s.text, nil }

func TestSpecsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Specs() {
		if s.ID == "" || seen[s.ID] {
			t.Fatalf("id %q", s.ID)
		}
		seen[s.ID] = true
	}
	if _, ok := Lookup("page"); !ok {
		t.Fatal("page")
	}
	if _, ok := Lookup("map"); !ok {
		t.Fatal("map")
	}
	if _, ok := Lookup("onsite"); !ok {
		t.Fatal("onsite")
	}
	if _, ok := Lookup("inventory"); !ok {
		t.Fatal("inventory")
	}
	if _, ok := Lookup("offsite"); !ok {
		t.Fatal("offsite")
	}
	if _, ok := Lookup("missing"); ok {
		t.Fatal("unknown")
	}
}

func TestInfer(t *testing.T) {
	if infer(map[string]any{"action": "outline"}, "whatever") != "outline" {
		t.Fatal("action")
	}
	if infer(map[string]any{}, "Write an outline for mechanic bookkeeping Lagos") != "outline" {
		t.Fatal("outline words")
	}
	if infer(map[string]any{}, "Audit this HTML for missing canonical") != "audit" {
		t.Fatal("audit")
	}
	if infer(map[string]any{}, "Title and meta for https://homegauge.themobileprof.com/rates") != "page" {
		t.Fatal("page")
	}
	if infer(map[string]any{"action": "brief"}, "") != "page" {
		t.Fatal("brief")
	}
	if infer(map[string]any{}, "Keyword map for Academy pages") != "map" {
		t.Fatal("map")
	}
	if infer(map[string]any{}, "Isolate keywords from Academy pages") != "inventory" {
		t.Fatal("inventory")
	}
	if infer(map[string]any{}, "Onsite pack Cursor can implement") != "onsite" {
		t.Fatal("onsite")
	}
	if infer(map[string]any{}, "Offsite listings we already have") != "offsite" {
		t.Fatal("offsite")
	}
}

func TestHandleStampsCannotPublish(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Accept on SEO.","structured_data":{"save_brief":true,"title":"Ikeja mortgage rates"}}`}, contract.Request{
		TaskDescription: "Title for HomeGauge rates page",
		ContextData:     []byte(`{"action":"page","project_slug":"homegauge"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "seo" || resp.StructuredData["action"] != "page" {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if resp.StructuredData["cannot_publish"] != true || resp.StructuredData["cannot_send"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if strings.Contains(resp.OutputText, "published") && strings.Contains(resp.OutputText, "live") {
		t.Fatal(resp.OutputText)
	}
}
