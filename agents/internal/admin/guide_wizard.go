package admin

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type GuideDraft struct {
	Recipe string            `json:"recipe"`
	Title  string            `json:"title,omitempty"`
	Action string            `json:"action,omitempty"`
	Button string            `json:"button,omitempty"`
	Href   string            `json:"href,omitempty"`
	Values map[string]string `json:"values,omitempty"`
	Fields []guideDraftField `json:"fields,omitempty"`
	Ready  bool              `json:"ready,omitempty"`
	Next   string            `json:"next,omitempty"`
	Secret bool              `json:"secret,omitempty"`
}

type guideDraftField struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Value string `json:"value"`
	Kind  string `json:"kind,omitempty"`
	Omit  bool   `json:"omit,omitempty"`
	Multi bool   `json:"multi,omitempty"`
}

type formWorld struct {
	Projects []Project
	People   []User
}

type guideLLM struct {
	Answer string            `json:"answer"`
	Hrefs  []string          `json:"hrefs"`
	Unsure bool              `json:"unsure"`
	Recipe string            `json:"recipe"`
	Fields map[string]string `json:"fields"`
	Cancel bool              `json:"cancel"`
}

var isoDateRe = regexp.MustCompile(`\b(20\d{2})-(\d{2})-(\d{2})\b`)

func (s *Server) loadFormWorld(ctx context.Context) formWorld {
	w := formWorld{}
	if s == nil || s.store == nil {
		return w
	}
	w.Projects, _ = s.store.ListProjects(ctx)
	w.People, _ = s.store.ListUsers(ctx)
	return w
}

func isCancel(q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	switch q {
	case "cancel", "stop", "never mind", "nevermind", "forget it", "start over", "abort":
		return true
	}
	return strings.HasPrefix(q, "cancel ") || strings.HasPrefix(q, "stop filling")
}

func worldPrompt(w formWorld) string {
	var b strings.Builder
	if len(w.Projects) > 0 {
		b.WriteString("Projects (use id in project_id, or name/slug — the desk will resolve):\n")
		for _, p := range w.Projects {
			fmt.Fprintf(&b, "- %d %s %s\n", p.ID, p.Slug, p.Name)
		}
	}
	if len(w.People) > 0 {
		b.WriteString("People (use id in user_id/assignee_id, or name):\n")
		for _, u := range w.People {
			if !u.Active {
				continue
			}
			fmt.Fprintf(&b, "- %d %s (%s)\n", u.ID, u.Name, u.Role)
		}
	}
	return b.String()
}

func draftPrompt(d *GuideDraft) string {
	if d == nil || strings.TrimSpace(d.Recipe) == "" {
		return "No open form."
	}
	var b strings.Builder
	b.WriteString("Open form recipe=")
	b.WriteString(d.Recipe)
	b.WriteString(" values=")
	first := true
	for k, v := range d.Values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if !first {
			b.WriteString(", ")
		}
		first = false
		b.WriteString(k)
		b.WriteString("=")
		if k == "password" {
			b.WriteString("••••")
		} else {
			b.WriteString(v)
		}
	}
	b.WriteString("\nKeep this recipe unless the user wants a different form or cancel.")
	return b.String()
}

func applyLLMForm(in guideInput, model guideLLM, world formWorld) (GuideReply, bool) {
	if model.Cancel || isCancel(in.Question) {
		hrefs := []string{"/admin/"}
		if in.Draft != nil {
			if rec, ok := lookupRecipe(in.Draft.Recipe); ok {
				hrefs = []string{resolveHref(rec, in.Draft.Values)}
			}
		}
		return GuideReply{
			Answer: "Stopped. Nothing was saved. You can type on the page form, or start again in Help?.",
			Hrefs:  hrefs,
			Source: "form",
		}, true
	}
	id := strings.TrimSpace(model.Recipe)
	if id == "" && in.Draft != nil {
		id = in.Draft.Recipe
	}
	if id == "" {
		return GuideReply{}, false
	}
	rec, ok := lookupRecipe(id)
	if !ok {
		return GuideReply{}, false
	}
	if reason := recipeAllowed(rec, in.User, in.Page); reason != "" {
		return GuideReply{
			Answer: reason + " You can still open the page and read the form.",
			Hrefs:  []string{resolveHref(rec, nil)},
			Source: "form",
		}, true
	}
	values := map[string]string{}
	if in.Draft != nil && in.Draft.Recipe == rec.ID {
		for k, v := range in.Draft.Values {
			if strings.TrimSpace(v) != "" {
				values[k] = v
			}
		}
	}
	seedDefaults(rec, values)
	prefillFromPage(rec, in.Page, values)
	mergeFields(rec, model.Fields, world, values)
	rep := wizardReply(rec, values, world)
	if ans := strings.TrimSpace(model.Answer); ans != "" {
		if next := nextMissing(rec, values); next != nil && !strings.Contains(strings.ToLower(ans), "next:") {
			ans = strings.TrimSpace(ans) + "\n\nNext: " + next.Prompt
		}
		rep.Answer = ans
		rep.Hrefs = cleanHrefs(append(rep.Hrefs, model.Hrefs...))
	}
	return rep, true
}

func continueDraft(in guideInput, world formWorld) (GuideReply, bool) {
	if isCancel(in.Question) && in.Draft != nil {
		return applyLLMForm(in, guideLLM{Cancel: true}, world)
	}
	if in.Draft == nil || strings.TrimSpace(in.Draft.Recipe) == "" {
		return GuideReply{}, false
	}
	rec, ok := lookupRecipe(in.Draft.Recipe)
	if !ok {
		return GuideReply{}, false
	}
	if reason := recipeAllowed(rec, in.User, in.Page); reason != "" {
		return GuideReply{Answer: reason, Source: "form"}, true
	}
	values := map[string]string{}
	for k, v := range in.Draft.Values {
		values[k] = v
	}
	seedDefaults(rec, values)
	prefillFromPage(rec, in.Page, values)
	if next := nextMissing(rec, values); next != nil {
		mergeFields(rec, map[string]string{next.Name: in.Question}, world, values)
	}
	return wizardReply(rec, values, world), true
}

func prefillFromPage(r formRecipe, page string, values map[string]string) {
	if values["project_id"] == "" {
		if id := parseProjectID(page); id > 0 {
			values["project_id"] = strconv.FormatInt(id, 10)
		}
	}
	if values["desk"] == "" {
		if d := parseDeskSlug(page); d != "" {
			values["desk"] = d
		}
	}
}

func mergeFields(r formRecipe, raw map[string]string, world formWorld, values map[string]string) {
	if raw == nil {
		return
	}
	known := map[string]formField{}
	for _, f := range r.Fields {
		known[f.Name] = f
	}
	for name, v := range raw {
		f, ok := known[name]
		if !ok {
			continue
		}
		got := coerceField(f, v, world)
		if got == "" {
			continue
		}
		values[f.Name] = clipValue(got)
	}
}

func coerceField(f formField, raw string, world formWorld) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "skip") || strings.EqualFold(raw, "none") {
		return ""
	}
	switch f.Kind {
	case "date":
		if looksLikeISODate(raw) {
			return raw
		}
		start, end := parseLooseDate(raw)
		if f.Name == "ends_on" && end != "" {
			return end
		}
		return start
	case "project":
		if id := resolveProject(raw, world); id != "" {
			return id
		}
		if digitsOnly(raw) {
			return raw
		}
		return ""
	case "person":
		if id := resolvePerson(raw, world); id != "" {
			return id
		}
		if digitsOnly(raw) {
			return raw
		}
		return ""
	case "desk":
		if d, ok := LookupDesk(strings.ToLower(raw)); ok {
			return d.Slug
		}
		return coerceOption(f, raw)
	case "select", "check", "multi":
		got := coerceOption(f, raw)
		if f.Kind != "check" && !validOption(f, got) {
			return ""
		}
		return got
	default:
		return raw
	}
}

func looksLikeISODate(s string) bool {
	s = strings.TrimSpace(s)
	if !isoDateRe.MatchString(s) || len(s) != 10 {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func digitsOnly(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func clipValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 2000 {
		return s[:2000]
	}
	return s
}

func parseLooseDate(s string) (string, string) {
	if m := isoDateRe.FindAllString(s, 2); len(m) > 0 {
		end := ""
		if len(m) > 1 {
			end = m[1]
		}
		return m[0], end
	}
	return "", ""
}

func resolveProject(text string, world formWorld) string {
	text = strings.TrimSpace(text)
	if digitsOnly(text) {
		id, _ := strconv.ParseInt(text, 10, 64)
		for _, p := range world.Projects {
			if p.ID == id {
				return strconv.FormatInt(id, 10)
			}
		}
		if len(world.Projects) == 0 {
			return text
		}
		return ""
	}
	low := strings.ToLower(text)
	bestID := int64(0)
	bestN := 0
	for _, p := range world.Projects {
		name := strings.ToLower(p.Name)
		slug := strings.ToLower(p.Slug)
		n := 0
		if slug != "" && (low == slug || strings.Contains(low, slug) || slug == low) {
			n += 3
		}
		if name != "" && (low == name || strings.Contains(low, name) || strings.Contains(name, low)) {
			n += 2
		}
		if n > bestN {
			bestN = n
			bestID = p.ID
		}
	}
	if bestN == 0 || bestID == 0 {
		return ""
	}
	return strconv.FormatInt(bestID, 10)
}

func resolvePerson(text string, world formWorld) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if digitsOnly(text) {
		return text
	}
	low := strings.ToLower(text)
	bestID := int64(0)
	bestN := 0
	for _, u := range world.People {
		name := strings.ToLower(u.Name)
		n := 0
		if name != "" && (low == name || strings.Contains(low, name) || strings.Contains(name, low)) {
			n += 3
		}
		if n > bestN {
			bestN = n
			bestID = u.ID
		}
	}
	if bestN == 0 || bestID == 0 {
		return ""
	}
	return strconv.FormatInt(bestID, 10)
}

func projectLabel(id string, world formWorld) string {
	n, _ := strconv.ParseInt(id, 10, 64)
	for _, p := range world.Projects {
		if p.ID == n {
			return p.Name
		}
	}
	return id
}

func personLabel(id string, world formWorld) string {
	n, _ := strconv.ParseInt(id, 10, 64)
	for _, u := range world.People {
		if u.ID == n {
			return u.Name
		}
	}
	return id
}

func displayValue(f formField, val string, world formWorld) string {
	if val == "" {
		return ""
	}
	if f.Kind == "secret" {
		return "••••"
	}
	if f.Kind == "project" {
		return projectLabel(val, world)
	}
	if f.Kind == "person" {
		return personLabel(val, world)
	}
	if f.Kind == "check" {
		if val == "on" {
			return "yes"
		}
		return "no"
	}
	if f.Kind == "select" || f.Kind == "desk" {
		for _, opt := range f.Options {
			if optionValue(opt) == val {
				return optionLabel(opt)
			}
		}
	}
	return val
}

func wizardReply(r formRecipe, values map[string]string, world formWorld) GuideReply {
	seedDefaults(r, values)
	next := nextMissing(r, values)
	href := resolveHref(r, values)
	draft := &GuideDraft{
		Recipe: r.ID,
		Title:  r.Title,
		Button: r.Button,
		Href:   href,
		Values: values,
	}
	var b strings.Builder
	if next != nil {
		draft.Next = next.Prompt
		draft.Secret = next.Kind == "secret"
		fmt.Fprintf(&b, "I can fill this with you, same as the page form (%s). Answer the next box, or open the page and type it yourself.\n\nNext: %s", r.Title, next.Prompt)
		if filled := summaryLines(r, values, world); filled != "" {
			fmt.Fprintf(&b, "\n\nHave so far:\n%s", filled)
		}
		b.WriteString("\n\nSay cancel to stop. I never click Accept.")
		return GuideReply{Answer: b.String(), Hrefs: []string{href}, Source: "form", Draft: draft}
	}
	action := resolveAction(r, values)
	if action == "" {
		return GuideReply{
			Answer: "That form is not ready — a project or worker is still missing. Name it, or open the page and fill it there.",
			Hrefs:  []string{href},
			Source: "form",
			Draft:  draft,
		}
	}
	draft.Action = action
	draft.Ready = true
	draft.Fields = postedFields(r, values)
	fmt.Fprintf(&b, "Ready. Check this, then Submit — same as clicking %s on the page. I never click Accept.\n\n%s", r.Button, summaryLines(r, values, world))
	b.WriteString("\nSay cancel to stop, or tell me which line to change.")
	return GuideReply{Answer: b.String(), Hrefs: []string{href}, Source: "form", Draft: draft}
}

func summaryLines(r formRecipe, values map[string]string, world formWorld) string {
	var b strings.Builder
	for _, f := range r.Fields {
		val := strings.TrimSpace(values[f.Name])
		if val == "" {
			continue
		}
		fmt.Fprintf(&b, "• %s: %s\n", f.Label, displayValue(f, val, world))
	}
	return strings.TrimRightFunc(b.String(), unicode.IsSpace)
}

func postedFields(r formRecipe, values map[string]string) []guideDraftField {
	var out []guideDraftField
	for _, f := range r.Fields {
		out = append(out, guideDraftField{
			Name:  f.Name,
			Label: f.Label,
			Value: strings.TrimSpace(values[f.Name]),
			Kind:  f.Kind,
			Omit:  f.PathOnly,
			Multi: f.Multi,
		})
	}
	return out
}

func sanitizeIncomingDraft(d *GuideDraft) *GuideDraft {
	if d == nil {
		return nil
	}
	rec, ok := lookupRecipe(strings.TrimSpace(d.Recipe))
	if !ok {
		return nil
	}
	out := &GuideDraft{Recipe: rec.ID, Values: map[string]string{}}
	for _, f := range rec.Fields {
		if d.Values == nil {
			continue
		}
		v := strings.TrimSpace(d.Values[f.Name])
		if v == "" {
			continue
		}
		out.Values[f.Name] = clipValue(v)
	}
	return out
}

func redactGuideQuestion(q string, in guideInput) string {
	if in.Draft == nil {
		return q
	}
	rec, ok := lookupRecipe(in.Draft.Recipe)
	if !ok {
		return q
	}
	values := map[string]string{}
	for k, v := range in.Draft.Values {
		values[k] = v
	}
	next := nextMissing(rec, values)
	if next != nil && next.Kind == "secret" {
		return "(password entered in Help? — not stored)"
	}
	return q
}
