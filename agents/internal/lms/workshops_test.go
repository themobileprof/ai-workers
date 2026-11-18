package lms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLooksLikeWorkshop(t *testing.T) {
	for _, s := range []string{
		"what does the workshop expect",
		"I don't know what to type",
		"stuck on step 2",
		"what should I type in the exercise",
		"the prompt is confusing",
		"I'm stuck",
		"what do I do",
	} {
		if !LooksLikeWorkshop(s) {
			t.Fatalf("want workshop: %s", s)
		}
	}
	if LooksLikeWorkshop("what is a pipe") {
		t.Fatal("concept question is not a workshop ask")
	}
	if WorkshopStepIndex("stuck on step 3 of the workshop") != 3 {
		t.Fatal(WorkshopStepIndex("stuck on step 3 of the workshop"))
	}
	if BareStepIndex("2") != 2 || BareStepIndex("the second one") != 2 || BareStepIndex("lesson 2") != 0 {
		t.Fatalf("bare step: 2=%d second=%d lesson2=%d", BareStepIndex("2"), BareStepIndex("the second one"), BareStepIndex("lesson 2"))
	}
}

func TestParseWorkshopSkipsComments(t *testing.T) {
	obj := map[string]any{
		"workshop": map[string]any{
			"is_enabled": true,
			"spec": map[string]any{
				"exercise": map[string]any{
					"steps": []any{
						map[string]any{
							"instructions":      []any{"<p>Type <code>pwd</code> in Termux.</p>"},
							"expected_commands": []any{"pwd", "echo $PWD"},
							"success_response":  "You printed the working directory.",
							"failure_response":  "Run pwd again.",
							"comments": []any{
								map[string]any{"trigger": "always", "text": "A working directory is the folder the shell is in. This is the concept."},
							},
						},
					},
				},
			},
		},
	}
	w := parseWorkshop(obj)
	if !w.HasSteps() || len(w.Steps) != 1 {
		t.Fatalf("%+v", w)
	}
	block := WorkshopBlock(Lesson{Title: "Linux Navigation", Order: 2, Workshop: w}, 1)
	if !strings.Contains(block, "pwd") || !strings.Contains(block, "Type") {
		t.Fatal(block)
	}
	if !strings.Contains(block, "Type one of these") || !strings.Contains(block, "`pwd`") {
		t.Fatal(block)
	}
	if strings.Contains(block, "folder the shell") {
		t.Fatalf("concept comment leaked:\n%s", block)
	}
	if !strings.Contains(block, "Student named step 1") {
		t.Fatal(block)
	}
}

func TestWorkshopMenuDoesNotDumpCommands(t *testing.T) {
	w := Workshop{Enabled: true, Steps: []WorkshopStep{
		{Instructions: []string{"Type pwd in Termux."}, Expected: []string{"pwd"}},
		{Instructions: []string{"Use echo to print $PWD."}, Expected: []string{"echo $PWD"}},
	}}
	lesson := Lesson{Title: "Linux Navigation", Order: 2, Workshop: w}
	menu := WorkshopOverlay(lesson, 0)
	if !strings.Contains(menu, "Step 1") || !strings.Contains(menu, "Type pwd") || !strings.Contains(menu, "One question") {
		t.Fatal(menu)
	}
	if strings.Contains(menu, "expected_commands") && strings.Contains(menu, "`pwd`") {
		t.Fatalf("menu dumped commands:\n%s", menu)
	}
	if strings.Contains(menu, "Type one of these") {
		t.Fatalf("menu dumped full step:\n%s", menu)
	}
	full := WorkshopOverlay(lesson, 2)
	if !strings.Contains(full, "`echo $PWD`") || strings.Contains(full, "Step 1:") {
		t.Fatal(full)
	}
}

func TestHydrateWorkshops(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/api/auth/admin/login"):
			_, _ = w.Write([]byte(`{"token":"jwt-ws"}`))
		case strings.Contains(r.URL.Path, "/api/admin/lessons/") && strings.HasSuffix(r.URL.Path, "/workshop"):
			hits++
			if r.Header.Get("Authorization") != "Bearer jwt-ws" {
				http.Error(w, `{"error":"no"}`, http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"lesson":{"id":"L2","title":"Linux Navigation"},"workshop":{"id":"W1","lesson_id":"L2","is_enabled":true,"spec":{"exercise":{"steps":[{"instructions":["Print your current directory."],"expected_commands":["pwd"],"success_response":"A path starting with /","failure_response":"Try pwd"}]}}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	c.email = "bot@example.com"
	c.pass = "secret"
	got := c.HydrateWorkshops(context.Background(), []Lesson{{ID: "L2", Title: "Linux Navigation", Order: 2}})
	if len(got) != 1 || !got[0].Workshop.HasSteps() {
		t.Fatalf("%+v", got)
	}
	if got[0].Workshop.Steps[0].Expected[0] != "pwd" {
		t.Fatalf("%+v", got[0].Workshop)
	}
	_ = c.HydrateWorkshops(context.Background(), []Lesson{{ID: "L2"}})
	if hits != 1 {
		t.Fatalf("cache missed, hits=%d", hits)
	}
}
