package growth

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

func TestPromptCuts(t *testing.T) {
	if strings.Contains(systemPrompt, "grants_growth") || strings.Contains(systemPrompt, "suggested_slot_window") {
		t.Fatal("growth no longer books calendars or hunts grants")
	}
	if !strings.Contains(systemPrompt, "next_action_by") {
		t.Fatal("high intent needs a Lagos date")
	}
	if !strings.Contains(salesOverlay, `Never say “bet”`) && !strings.Contains(salesOverlay, `Never say "bet"`) {
		t.Fatal("WhatsApp front desk must not say bet")
	}
	if !strings.Contains(salesOverlay, "context_data.products") {
		t.Fatal("WhatsApp front desk needs the product list")
	}
	if !strings.Contains(salesOverlay, "project_sticky") {
		t.Fatal("WhatsApp must keep the product they already named")
	}
}

type stubLLM struct {
	text   string
	system *string
}

func (s stubLLM) Complete(_ context.Context, req llm.Request) (string, error) {
	if s.system != nil {
		*s.system = req.System
	}
	return s.text, nil
}

func TestHandle(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Thursday works.","structured_data":{"task_type":"sales_support"}}`}, contract.Request{
		TaskDescription: "When can we start the pilot?",
		ContextData:     []byte(`{"channel":"whatsapp"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.StatusSuccess || resp.StructuredData["task_type"] != "sales_support" {
		t.Fatalf("%+v", resp)
	}
}

func TestWhatsAppIsNotADeck(t *testing.T) {
	if LooksLike(map[string]any{"channel": "whatsapp"}, "pitch deck for HomeGauge") {
		t.Fatal("customer WhatsApp stays sales")
	}
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"We are TheMobileProf.","structured_data":{"task_type":"sales_support"}}`}, contract.Request{
		TaskDescription: "Send me your pitch deck",
		ContextData:     []byte(`{"channel":"whatsapp"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "sales_support" {
		t.Fatalf("%+v", resp)
	}
}

func TestWhatsAppFrontDeskUsesSalesOverlay(t *testing.T) {
	if !CustomerChannel(map[string]any{"channel": "whatsapp"}) {
		t.Fatal("whatsapp is the customer channel")
	}
	var system string
	_, err := Handle(context.Background(), stubLLM{
		text:   `{"status":"success","output_text":"HomeGauge helps salaried homebuyers.","structured_data":{"task_type":"sales_support"}}`,
		system: &system,
	}, contract.Request{
		TaskDescription: "What is HomeGauge?",
		ContextData:     []byte(`{"channel":"whatsapp","products":[{"name":"HomeGauge","one_liner":"Salaried Nigerian homebuyers get mortgage help.","url":"https://homegauge.themobileprof.com"}]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(system, "front desk") {
		t.Fatal("WhatsApp must use the front-desk overlay")
	}
	if !strings.Contains(system, "product, offer, or project") {
		t.Fatal("customers hear product, not bet")
	}
	if !strings.Contains(system, "Switch only when they clearly named a different product") {
		t.Fatal("sticky product until they name another")
	}
}

func TestDeckInfer(t *testing.T) {
	if !LooksLike(map[string]any{"channel": "telegram", "action": "walk"}, "continue") {
		t.Fatal("desk walk")
	}
	if Infer(map[string]any{"action": "walk"}, "Pitch deck for HomeGauge, Ada in the room") != "session" {
		t.Fatal("named pitch is session")
	}
	if !LooksLike(map[string]any{"channel": "desk", "ref_kind": "growth_deck"}, "make the title punchier") {
		t.Fatal("job chat is a deck session")
	}
	if Infer(map[string]any{"action": "audience"}, "next") != "audience" {
		t.Fatal("named kind")
	}
	if Infer(nil, "Business plan for Mechazone") != "plan" {
		t.Fatal("plan")
	}
}

func TestMergeSlidesKeepsAgreed(t *testing.T) {
	base := EmptySlides()
	base[0].Copy = "HomeGauge · TheMobileProf Technologies"
	base[0].Status = "agreed"
	got := MergeSlides(base, []DeckSlide{{N: 2, Copy: "A desk will quote today.", Status: "draft"}})
	if got[0].Status != "agreed" || got[0].Copy != "HomeGauge · TheMobileProf Technologies" {
		t.Fatalf("agreed slide must stay, %+v", got[0])
	}
	if got[1].Copy != "A desk will quote today." {
		t.Fatalf("current slide copy, %+v", got[1])
	}
}

func TestKawasakiSlides(t *testing.T) {
	slides := KawasakiSlides()
	if len(slides) != 10 {
		t.Fatalf("want 10, got %d", len(slides))
	}
	want := []string{
		"Title",
		"Problem / Opportunity",
		"Value Proposition",
		"Underlying Magic",
		"Business Model",
		"Go-to-Market Plan",
		"Competitive Analysis",
		"Management Team",
		"Financials and Key Metrics",
		"Current Status, Timeline, and Use of Funds",
	}
	for i, title := range want {
		if slides[i].N != i+1 || slides[i].Title != title {
			t.Fatalf("slide %d: %+v", i+1, slides[i])
		}
	}
}

func TestHandleDeck(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"Ada in the room.","structured_data":{"action":"audience","body":"Ada (cofounder).","title":"HomeGauge audience"}}`}, contract.Request{
		TaskDescription: "Pitch deck for HomeGauge. Ada is in the room.",
		ContextData:     []byte(`{"channel":"desk","action":"audience","project_slug":"homegauge","project_id":7}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "deck" || resp.StructuredData["cannot_send"] != true || resp.StructuredData["save_deck"] != true {
		t.Fatalf("%+v", resp)
	}
}
