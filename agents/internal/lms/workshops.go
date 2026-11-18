package lms

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Workshop is the LMS exercise attached to a lesson. Steps are what the prompt is waiting for.
type Workshop struct {
	Enabled bool
	Title   string
	Steps   []WorkshopStep
}

// WorkshopStep is one exercise prompt: instructions, accepted commands, success/fail copy.
type WorkshopStep struct {
	Instructions []string
	Expected     []string
	Success      string
	FailHint     string
}

func (w Workshop) HasSteps() bool {
	return w.Enabled && len(w.Steps) > 0
}

var (
	workshopRe = regexp.MustCompile(`(?i)\b(workshop|exercise|what (do i|should i|to) (type|do)|what does (it|this|the workshop|the exercise) (want|expect)|stuck|confused|don'?t get (it|this)|the prompt)\b`)
	stepNumRe  = regexp.MustCompile(`(?i)\bstep\s*#?\s*(\d+)\b`)
	bareStepRe = regexp.MustCompile(`(?i)^\s*(?:step\s*)?#?\s*(\d{1,2})\s*$`)
)

func LooksLikeWorkshop(text string) bool {
	return workshopRe.MatchString(text)
}

func WorkshopStepIndex(text string) int {
	m := stepNumRe.FindStringSubmatch(text)
	if len(m) != 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// BareStepIndex is a short reply to "which step?" — "2", "step 2", "the second one".
func BareStepIndex(text string) int {
	if n := WorkshopStepIndex(text); n > 0 {
		return n
	}
	if m := bareStepRe.FindStringSubmatch(text); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	switch strings.Trim(strings.ToLower(strings.TrimSpace(text)), ".!") {
	case "first", "1st", "the first", "the first one":
		return 1
	case "second", "2nd", "the second", "the second one":
		return 2
	case "third", "3rd", "the third", "the third one":
		return 3
	case "fourth", "4th", "the fourth", "the fourth one":
		return 4
	default:
		return 0
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		var out []string
		for _, x := range t {
			if s := str(x); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if s := strings.TrimSpace(t); s != "" {
			return []string{s}
		}
	}
	return nil
}

func cleanLines(in []string, n int) []string {
	var out []string
	for _, s := range in {
		s = clip(plain(s), n)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func parseWorkshop(obj map[string]any) Workshop {
	raw := asMap(obj["workshop"])
	if raw == nil {
		return Workshop{}
	}
	spec := asMap(raw["spec"])
	exercise := asMap(spec["exercise"])
	if exercise == nil {
		exercise = spec
	}
	if exercise == nil {
		exercise = raw
	}
	w := Workshop{
		Enabled: boolish(firstAny(raw, "is_enabled", "isEnabled")),
		Title:   firstStr(raw, "title"),
	}
	if w.Title == "" {
		w.Title = firstStr(exercise, "title", "id")
	}
	for _, step := range asList(exercise, "steps") {
		st := WorkshopStep{
			Instructions: cleanLines(stringSlice(step["instructions"]), 400),
			Expected:     cleanLines(stringSlice(step["expected_commands"]), 200),
			Success:      clip(plain(str(step["success_response"])), 400),
			FailHint:     clip(plain(str(step["failure_response"])), 400),
		}
		if len(st.Instructions) == 0 && len(st.Expected) == 0 {
			continue
		}
		w.Steps = append(w.Steps, st)
	}
	if !w.Enabled && len(w.Steps) > 0 {
		w.Enabled = true
	}
	return w
}

// HydrateWorkshops fills lesson.Workshop from GET /api/admin/lessons/{id}/workshop.
func (c *Client) HydrateWorkshops(ctx context.Context, lessons []Lesson) []Lesson {
	if c == nil || !c.HasAuth() || len(lessons) == 0 {
		return lessons
	}
	out := make([]Lesson, len(lessons))
	copy(out, lessons)
	for i := range out {
		if out[i].ID == "" {
			continue
		}
		if ws, ok := c.workshop(ctx, out[i].ID); ok {
			out[i].Workshop = ws
		}
	}
	return out
}

func (c *Client) workshop(ctx context.Context, lessonID string) (Workshop, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.workshops[lessonID]; ok && time.Since(cached.at) < c.ttl {
		return cached.item, cached.ok
	}
	obj, err := c.authJSON(ctx, "/api/admin/lessons/"+lessonID+"/workshop")
	if err != nil {
		c.workshops[lessonID] = cachedWorkshop{at: time.Now(), ok: false}
		return Workshop{}, false
	}
	w := parseWorkshop(obj)
	ok := w.HasSteps()
	c.workshops[lessonID] = cachedWorkshop{at: time.Now(), item: w, ok: ok}
	return w, ok
}

// WorkshopOverlay is the workshop text for this turn: full step if known, else a short menu to ask which step.
func WorkshopOverlay(lesson Lesson, stepN int) string {
	w := lesson.Workshop
	if !w.HasSteps() {
		return ""
	}
	if len(w.Steps) == 1 {
		return WorkshopBlock(lesson, 1)
	}
	if stepN > 0 && stepN <= len(w.Steps) {
		return WorkshopBlock(lesson, stepN)
	}
	return WorkshopMenu(lesson)
}

// WorkshopMenu lists step labels so the host can ask which step. No commands.
func WorkshopMenu(lesson Lesson) string {
	w := lesson.Workshop
	if !w.HasSteps() {
		return ""
	}
	var b strings.Builder
	b.WriteString("LMS workshop menu for lesson ")
	if lesson.Order > 0 {
		b.WriteString(fmt.Sprintf("%d ", lesson.Order))
	}
	b.WriteString(fmt.Sprintf("%q (%d steps). Student has not named a step yet. Ask which step they are on (number or the first words of the prompt). One question. Do not quote expected_commands or dump the full step until they pick.\n", lesson.Title, len(w.Steps)))
	for i, st := range w.Steps {
		preview := ""
		if len(st.Instructions) > 0 {
			preview = clip(st.Instructions[0], 120)
		}
		b.WriteString(fmt.Sprintf("- Step %d: %s\n", i+1, preview))
	}
	return b.String()
}

// WorkshopBlock is what the exercise prompt is waiting for. No concept lecture.
func WorkshopBlock(lesson Lesson, stepN int) string {
	w := lesson.Workshop
	if !w.HasSteps() {
		return ""
	}
	var b strings.Builder
	b.WriteString("LMS workshop for lesson ")
	if lesson.Order > 0 {
		b.WriteString(fmt.Sprintf("%d ", lesson.Order))
	}
	b.WriteString(fmt.Sprintf("%q (%d steps). Restate what the current prompt expects: what to type, on which screen, what success looks like. Do not lecture the concept. Do not invent a command that is not listed.\n", lesson.Title, len(w.Steps)))
	start, end := 0, len(w.Steps)
	if stepN > 0 && stepN <= len(w.Steps) {
		start, end = stepN-1, stepN
		b.WriteString(fmt.Sprintf("Student named step %d of %d.\n", stepN, len(w.Steps)))
	}
	for i := start; i < end; i++ {
		st := w.Steps[i]
		b.WriteString(fmt.Sprintf("\nStep %d:\n", i+1))
		if len(st.Instructions) > 0 {
			b.WriteString("  The exercise asks:\n")
			for _, line := range st.Instructions {
				b.WriteString("  - " + line + "\n")
			}
		}
		if len(st.Expected) > 0 {
			b.WriteString("  Type one of these (verbatim):\n")
			for _, cmd := range st.Expected {
				b.WriteString("  - `" + cmd + "`\n")
			}
		}
		if st.Success != "" {
			b.WriteString("  Success looks like: " + st.Success + "\n")
		}
		if st.FailHint != "" {
			b.WriteString("  If it fails, the exercise hint is: " + st.FailHint + "\n")
		}
	}
	s := b.String()
	if len(s) > 5000 {
		return s[:4990] + "\n…"
	}
	return s
}
