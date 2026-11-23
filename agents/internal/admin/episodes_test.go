package admin

import (
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func TestSkipEpisode(t *testing.T) {
	if !skipEpisode("growth", "whatsapp", "whatsapp", "sales_support") {
		t.Fatal("whatsapp")
	}
	if !skipEpisode("community", "telegram", "telegram", "progress") {
		t.Fatal("community telegram")
	}
	if skipEpisode("community", "desk", "desk", "progress") {
		t.Fatal("community desk Ask is staff")
	}
	if !skipEpisode("growth", "n8n", "desk", "ads_glance") {
		t.Fatal("glance")
	}
	if skipEpisode("growth", "desk", "desk", "ads") {
		t.Fatal("staff ads")
	}
	if skipEpisode("legal", "telegram", "telegram", "legal") {
		t.Fatal("ops legal")
	}
}

func TestRedactEpisodeText(t *testing.T) {
	got := redactEpisodeText("Call Ada on 08012345678 or ada@themobileprof.com")
	if strings.Contains(got, "08012345678") || strings.Contains(got, "ada@") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "[phone]") || !strings.Contains(got, "[email]") {
		t.Fatal(got)
	}
}

func TestEpisodeInputDropsPII(t *testing.T) {
	in := episodeInput(map[string]any{
		"from":         "2348012345678",
		"phone":        "08012345678",
		"email":        "ada@themobileprof.com",
		"thread":       []string{"you: 08012345678"},
		"project_slug": "homegauge",
		"looks":        []string{"card", "Call 08012345678"},
		"action":       "ads",
	})
	if _, ok := in["from"]; ok {
		t.Fatal("from")
	}
	if _, ok := in["thread"]; ok {
		t.Fatal("thread")
	}
	if strVal(in["project_slug"]) != "homegauge" {
		t.Fatal(in)
	}
	looks, _ := in["looks"].([]string)
	if len(looks) != 2 || strings.Contains(looks[1], "080") {
		t.Fatalf("%v", looks)
	}
}

func TestLessonLines(t *testing.T) {
	got := lessonLines(
		[]DeskEpisode{{TaskType: "ads", ProjectSlug: "homegauge", HumanAct: "accepted", Title: "Search ads pack"}},
		[]DeskEpisode{{TaskType: "ads", ProjectSlug: "homegauge", HumanAct: "rejected", Reason: "invented Ikeja"}},
	)
	acc, _ := got["accepted"].([]string)
	rej, _ := got["turned_away"].([]string)
	if len(acc) != 1 || !strings.Contains(acc[0], "homegauge") || !strings.Contains(acc[0], "Search ads") {
		t.Fatalf("accepted %v", acc)
	}
	if len(rej) != 1 || !strings.Contains(rej[0], "Ikeja") {
		t.Fatalf("rejected %v", rej)
	}
	if lessonLines(nil, []DeskEpisode{{HumanAct: "rejected"}}) != nil {
		t.Fatal("empty reason is not a lesson")
	}
}

func TestEpisodeProposalClipsOutput(t *testing.T) {
	got := episodeProposal(contract.Response{
		OutputText: strings.Repeat("x", 800),
		StructuredData: map[string]any{
			"task_type":    "ads",
			"cannot_spend": true,
			"from":         "should-drop",
		},
	})
	if strVal(got["task_type"]) != "ads" || got["cannot_spend"] != true {
		t.Fatal(got)
	}
	if _, ok := got["from"]; ok {
		t.Fatal("from leaked")
	}
	if n := len([]rune(strVal(got["output"]))); n > 400 {
		t.Fatalf("output %d", n)
	}
}

func TestExportEpisodesMarkdown(t *testing.T) {
	md := ExportEpisodesMarkdown([]DeskEpisode{{
		Desk: "growth", ProjectSlug: "homegauge", TaskType: "ads",
		HumanAct: "rejected", Reason: "invented a city", Title: "Search ads",
	}})
	if !strings.Contains(md, "Staff episodes") || !strings.Contains(md, "turned away") || !strings.Contains(md, "invented a city") {
		t.Fatal(md)
	}
	if strings.Contains(ExportEpisodesMarkdown(nil), "turned away") {
		t.Fatal("empty")
	}
}

func TestFormReason(t *testing.T) {
	if formReason(nil) != "" {
		t.Fatal("nil")
	}
}
