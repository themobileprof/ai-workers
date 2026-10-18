package admin

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/llm"
)

type stubGuide struct{ text string }

func (s stubGuide) Complete(context.Context, llm.Request) (string, error) { return s.text, nil }

func TestCatalogPromptListsFormsNotKeywords(t *testing.T) {
	got := catalogPrompt(formCatalog())
	if !strings.Contains(got, "id=add_date") || !strings.Contains(got, "id=add_task") {
		t.Fatalf("%s", got)
	}
	if strings.Contains(got, "unilag") {
		t.Fatal("catalog must not hard-code example keywords as matchers")
	}
}

func TestApplyLLMFormFillsDateFromModel(t *testing.T) {
	world := formWorld{Projects: []Project{{ID: 7, Name: "TheMobileProf Academy", Slug: "academy"}}}
	rep, ok := applyLLMForm(guideInput{
		User:     User{Name: "Ada", Role: "owner"},
		Question: "please put the campus week on the academy card",
	}, guideLLM{
		Answer: "Saving the Dates form.",
		Recipe: "add_date",
		Fields: map[string]string{
			"project_id": "Academy",
			"title":      "Linux on Mobile — UNILAG",
			"kind":       "training",
			"starts_on":  "2026-11-17",
			"ends_on":    "2026-11-19",
			"venue":      "University of Lagos",
		},
	}, world)
	if !ok || rep.Draft == nil || !rep.Draft.Ready {
		t.Fatalf("%+v ok=%v", rep, ok)
	}
	if rep.Draft.Values["project_id"] != "7" || rep.Draft.Action != "/admin/projects/7/events" {
		t.Fatalf("%+v", rep.Draft)
	}
}

func TestApplyLLMFormRejectsUnknownRecipe(t *testing.T) {
	rep, ok := applyLLMForm(guideInput{User: User{Role: "owner"}, Question: "x"}, guideLLM{
		Answer: "ok", Recipe: "invented_form", Fields: map[string]string{"title": "no"},
	}, formWorld{})
	if ok {
		t.Fatalf("invented recipe must be ignored, got %+v", rep)
	}
}

func TestApplyLLMFormOwnerOnly(t *testing.T) {
	rep, ok := applyLLMForm(guideInput{
		User:     User{Name: "Ada", Role: "assistant"},
		Question: "add Chidi",
	}, guideLLM{Recipe: "add_person", Fields: map[string]string{"name": "Chidi"}}, formWorld{})
	if !ok || !strings.Contains(strings.ToLower(rep.Answer), "only the owner") {
		t.Fatalf("%+v ok=%v", rep, ok)
	}
}

func TestContinueDraftFillsNextFieldWithoutLLM(t *testing.T) {
	rep, ok := continueDraft(guideInput{
		User:     User{Name: "Ada", Role: "owner"},
		Question: "Linux on Mobile",
		Draft:    &GuideDraft{Recipe: "add_date", Values: map[string]string{"project_id": "7"}},
	}, formWorld{Projects: []Project{{ID: 7, Name: "Academy", Slug: "academy"}}})
	if !ok || rep.Draft == nil || rep.Draft.Values["title"] != "Linux on Mobile" {
		t.Fatalf("%+v ok=%v", rep, ok)
	}
}

func TestAnswerGuideLLMPicksFormFromIntent(t *testing.T) {
	s := &Server{guide: stubGuide{text: `{"answer":"I will fill Dates.","unsure":false,"hrefs":["/admin/projects"],"recipe":"add_date","fields":{"title":"UNILAG training","kind":"training","starts_on":"2026-11-17","project_id":"academy"},"cancel":false}`}}
	got := s.answerGuide(context.Background(), guideInput{
		User:     User{Name: "Ada", Role: "owner"},
		Question: "can you log the CS class the faculty asked us to run in November?",
	})
	if got.Draft == nil || got.Draft.Recipe != "add_date" || got.Draft.Values["title"] != "UNILAG training" {
		t.Fatalf("%+v", got)
	}
}

func TestSafeFormActionBlocksStamps(t *testing.T) {
	if safeFormAction("/admin/projects/1/proposal/accept") != "" {
		t.Fatal("accept")
	}
	if safeFormAction("/admin/legal/3/approve") != "" {
		t.Fatal("approve")
	}
	if safeFormAction("/admin/projects/1/events") != "/admin/projects/1/events" {
		t.Fatal("dates form should pass")
	}
}
