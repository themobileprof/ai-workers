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
	if d.Kind != KindWork || d.Department != "delivery" || d.Action != "report" {
		t.Fatalf("%+v", d)
	}
}

func TestLooksLikeReport(t *testing.T) {
	ikorodu := "We went to Ikorodu General Hospital yesterday, had a 5 minutes speech with the pregnant women at the ANC, registered names, and left fliers. The HOD said the next opportunity is next Monday."
	if !LooksLikeReport(ikorodu) {
		t.Fatal("hospital visit")
	}
	sultan := "On Tuesday, Sultan created a discount code THEMOBILEPROF and takes 10% commission. I have asked him to share it with the students he already teaches."
	if !LooksLikeReport(sultan) {
		t.Fatal("partner discount")
	}
	unilag := "Unilag training the trainer has been postponed. We will wait for the faculty to name a new week."
	if !LooksLikeReport(unilag) {
		t.Fatal("postponed training")
	}
	spend := "On Monday we spent N20,000 on fuel and N15,000 on fliers for the hospital visits."
	if !LooksLikeReport(spend) {
		t.Fatal("spend narrative")
	}
	if LooksLikeReport("tighten slide 3") {
		t.Fatal("growth copy")
	}
	if LooksLikeReport("I am stuck on Starting Termux and pkg upgrade will not finish") {
		t.Fatal("workshop chatter")
	}
	if LooksLikeReport("/today") {
		t.Fatal("prefix")
	}
}

func TestDecideReportOverridesStickyAndFile(t *testing.T) {
	ikorodu := "We went to Ikorodu General Hospital yesterday, had a 5 minutes speech with the pregnant women at the ANC, registered names, and left fliers."
	d := Decide(Inbound{ChatID: "1", FromID: "1", Text: ikorodu}, "accounts")
	if d.Department != "delivery" || d.Action != "report" {
		t.Fatalf("sticky accounts %+v", d)
	}
	d = Decide(Inbound{ChatID: "1", FromID: "1", Text: ikorodu, HasDocument: true, Filename: "flier.pdf"}, "")
	if d.Department != "delivery" || d.Action != "report" {
		t.Fatalf("document %+v", d)
	}
	d = Decide(Inbound{ChatID: "1", FromID: "1", HasPhoto: true, PhotoFileID: "x"}, "accounts")
	if d.Department != "accounts" {
		t.Fatalf("bare photo %+v", d)
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
	d = Decide(Inbound{ChatID: "1", FromID: "1", Text: "tighten slide 3"}, "delivery")
	if d.Kind != KindWork || d.Department != "" {
		t.Fatalf("non-report after Delivery must classify %+v", d)
	}
}

func TestDecideGreeting(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "1", Text: "hello"}, "")
	if d.Kind != KindGreet {
		t.Fatalf("%+v", d)
	}
	d = Decide(Inbound{ChatID: "1", FromID: "1", Text: "hello"}, "delivery")
	if d.Department == "delivery" {
		t.Fatalf("short hello after Delivery must not stay trapped %+v", d)
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
