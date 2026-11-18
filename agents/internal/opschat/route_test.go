package opschat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/llm"
)

func TestParseUpdate(t *testing.T) {
	raw := []byte(`{"update_id":1,"message":{"message_id":2,"date":1,"from":{"id":99,"is_bot":false,"first_name":"S"},"chat":{"id":99,"type":"private"},"text":"hello"}}`)
	in, ok := ParseUpdate(raw)
	if !ok || in.ChatID != "99" || in.Text != "hello" {
		t.Fatalf("%+v ok=%v", in, ok)
	}
}

func TestDecideMorningCardIsDelivery(t *testing.T) {
	d := Decide(Inbound{
		ChatID:  "1",
		FromID:  "1",
		Text:    "done the calculator",
		ReplyTo: "Today on the desk (Samuel):\n- HomeGauge: calculator",
	}, "")
	if d.Kind != KindWork || d.Department != "delivery" || d.Action != "today" {
		t.Fatalf("%+v", d)
	}
}

func TestDecidePhotoIsAccounts(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", HasPhoto: true, PhotoFileID: "x"}, "")
	if d.Department != "accounts" {
		t.Fatalf("%+v", d)
	}
}

func TestDecideDocumentIsFiles(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", HasDocument: true, DocumentFileID: "d", Filename: "a.pdf"}, "")
	if d.Department != "files" {
		t.Fatalf("%+v", d)
	}
}

func TestDecidePrefixOverridesThread(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", Text: "/legal review this NDA"}, "growth")
	if !d.Prefixed || d.Department != "legal" {
		t.Fatalf("%+v", d)
	}
}

func TestDecideSeoGone(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", Text: "/seo HomeGauge rates"}, "")
	if d.Kind != KindNote || d.Department != "" || !strings.Contains(d.Reply, "no SEO worker") {
		t.Fatalf("%+v", d)
	}
}

func TestDecideKeepsCurrentDesk(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", Text: "tighten slide 3"}, "growth")
	if d.Department != "growth" || d.Kind != KindWork {
		t.Fatalf("%+v", d)
	}
}

func TestDecideGreeting(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", Text: "hello"}, "")
	if d.Kind != KindGreet {
		t.Fatalf("%+v", d)
	}
	d = Decide(Inbound{ChatID: "1", FromID: "1", Text: "hello"}, "delivery")
	if d.Kind != KindWork || d.Department != "delivery" {
		t.Fatalf("thread %+v", d)
	}
}

func TestDecideApproveIsOutbox(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", Text: "/approve"}, "")
	if d.Kind != KindOutbox || d.Action != "send" {
		t.Fatalf("%+v", d)
	}
	d = Decide(Inbound{ChatID: "1", FromID: "1", Text: "/stop"}, "growth")
	if d.Kind != KindOutbox || d.Action != "kill" {
		t.Fatalf("stop %+v", d)
	}
}

func TestDecideNeedsClassify(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", Text: "NDA for the facilitator"}, "")
	if d.Kind != KindWork || d.Department != "" {
		t.Fatalf("%+v", d)
	}
}

func TestClassifyOneDepartment(t *testing.T) {
	c := stubLLM{text: `{"status":"success","output_text":"ok","structured_data":{"department":"legal"}}`}
	got, err := Classify(context.Background(), c, "NDA for the facilitator", "")
	if err != nil || got != "legal" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestClassifyIgnoresList(t *testing.T) {
	c := stubLLM{text: `{"status":"success","output_text":"ok","structured_data":{"department":"delivery","route_to":["delivery","growth"]}}`}
	got, err := Classify(context.Background(), c, "done calculator", "Today on the desk")
	if err != nil || got != "delivery" {
		t.Fatalf("%q %v", got, err)
	}
}

type stubLLM struct{ text string }

func (s stubLLM) Complete(context.Context, llm.Request) (string, error) {
	return s.text, nil
}

func TestHelpText(t *testing.T) {
	s := HelpText("Samuel", []string{"/today", "/growth"})
	if s == "" {
		t.Fatal("empty")
	}
	raw, _ := json.Marshal(s)
	if len(raw) < 8 {
		t.Fatal(s)
	}
}
