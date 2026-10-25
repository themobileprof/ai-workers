package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func (s *Server) EnrichProduct(ctx context.Context, req contract.Request) contract.Request {
	if s == nil || s.store == nil {
		return req
	}
	data := contract.ContextObject(req.ContextData)
	projects, err := s.store.ListProjects(ctx)
	if err != nil {
		return req
	}
	list := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		list = append(list, map[string]any{"id": p.ID, "slug": p.Slug, "name": p.Name, "one_liner": p.OneLiner})
	}
	data["projects"] = list
	data["cannot_write_stamp"] = true
	if p, ok := matchProject(projects, data, req.TaskDescription); ok {
		attachProductProject(data, p)
	}
	raw, _ := json.Marshal(data)
	req.ContextData = raw
	return req
}

func (s *Server) applyProductResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	sd := resp.StructuredData
	data := contract.ContextObject(req.ContextData)
	projects, err := s.store.ListProjects(ctx)
	if err != nil {
		return
	}
	blob := strings.Join([]string{req.TaskDescription, strVal(sd["project_slug"]), strVal(sd["project"])}, " ")
	p, ok := matchProject(projects, mergeProductLookup(data, sd), blob)
	if !ok {
		return
	}
	sd["project_id"] = p.ID
	sd["project_slug"] = p.Slug
	sd["project_name"] = p.Name
	if strings.EqualFold(strVal(sd["action"]), "place_on_journey") {
		prop, err := proposalFromStructured(sd)
		if err != nil {
			return
		}
		_ = s.store.SetProposal(ctx, p.ID, prop)
	}
}

func (s *Server) applyProductMemory(ctx context.Context, job DeskJob) error {
	if s == nil || s.store == nil {
		return nil
	}
	p, err := s.productJobProject(ctx, job)
	if err != nil {
		return err
	}
	hyp, mission, due, burn, notes, notesChanged := mergedCard(p, job.Payload)
	if err := s.store.SetProjectMemory(ctx, p.ID, hyp, mission, due, burn); err != nil {
		return err
	}
	if notesChanged {
		return s.store.SetProjectNotes(ctx, p.ID, notes)
	}
	return nil
}

func (s *Server) productJobProject(ctx context.Context, job DeskJob) (Project, error) {
	if job.RefID > 0 {
		return s.store.GetProject(ctx, job.RefID)
	}
	sd := job.Payload
	if sd == nil {
		sd = map[string]any{}
	}
	slug := firstNonEmpty(strVal(sd["project_slug"]), strVal(sd["project"]))
	if slug == "" {
		return Project{}, errors.New("project is missing")
	}
	return s.store.GetProjectBySlug(ctx, slug)
}

func mergedCard(p Project, sd map[string]any) (hypothesis, mission, due, burn, notes string, notesChanged bool) {
	if sd == nil {
		sd = map[string]any{}
	}
	hypothesis = firstNonEmpty(strVal(sd["card_hypothesis"]), p.Hypothesis)
	mission = firstNonEmpty(strVal(sd["card_mission"]), p.Mission)
	due = firstNonEmpty(strVal(sd["card_mission_due"]), p.MissionDue)
	burn = p.BurnNGN
	notes = p.Notes
	if ev := strVal(sd["card_evidence"]); ev != "" {
		notes = appendCardNote(p.Notes, ev)
		notesChanged = notes != p.Notes
	}
	return hypothesis, mission, due, burn, notes, notesChanged
}

func appendCardNote(existing, ev string) string {
	ev = strings.TrimSpace(ev)
	if ev == "" {
		return existing
	}
	existing = strings.TrimSpace(existing)
	if strings.Contains(existing, ev) {
		return existing
	}
	if existing == "" {
		return clipRunes(ev, 800)
	}
	return clipRunes(existing+"\n\n"+ev, 1200)
}

func clipRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

func attachProductProject(data map[string]any, p Project) {
	data["project_id"] = p.ID
	data["project_slug"] = p.Slug
	data["project"] = map[string]any{
		"id":                p.ID,
		"name":              p.Name,
		"slug":              p.Slug,
		"one_liner":         p.OneLiner,
		"url":               p.URL,
		"notes":             p.Notes,
		"hypothesis":        p.Hypothesis,
		"mission":           p.Mission,
		"mission_due":       p.MissionDue,
		"burn_ngn":          p.BurnNGN,
		"stage":             p.Stage,
		"committed_journey": p.Journey,
		"committed_gate":    p.Gate,
	}
}

func mergeProductLookup(data, sd map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range data {
		out[k] = v
	}
	if id := int64Val(sd["project_id"]); id > 0 && int64Val(out["project_id"]) == 0 {
		out["project_id"] = id
	}
	if s := strVal(sd["project_slug"]); s != "" && strVal(out["project_slug"]) == "" {
		out["project_slug"] = s
	}
	if sd["project"] != nil && out["project"] == nil {
		out["project"] = sd["project"]
	}
	return out
}

func matchProject(projects []Project, data map[string]any, task string) (Project, bool) {
	if data == nil {
		data = map[string]any{}
	}
	if id := int64Val(data["project_id"]); id > 0 {
		for _, p := range projects {
			if p.ID == id {
				return p, true
			}
		}
	}
	slug := firstNonEmpty(strVal(data["project_slug"]), strVal(data["project"]))
	if nested, ok := data["project"].(map[string]any); ok {
		slug = firstNonEmpty(slug, strVal(nested["slug"]))
		if id := int64Val(nested["id"]); id > 0 {
			for _, p := range projects {
				if p.ID == id {
					return p, true
				}
			}
		}
	}
	if slug != "" {
		want := normalizeSlug(slug)
		for _, p := range projects {
			if p.Slug == want {
				return p, true
			}
		}
	}
	blob := strings.ToLower(task)
	var hits []Project
	for _, p := range projects {
		if projectMentioned(blob, p) {
			hits = append(hits, p)
		}
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	return Project{}, false
}

func projectMentioned(blob string, p Project) bool {
	if tokenIn(blob, p.Slug) {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(p.Name))
	if name == "" {
		return false
	}
	if strings.Contains(name, " ") {
		return strings.Contains(blob, name)
	}
	return tokenIn(blob, name)
}

func tokenIn(hay, needle string) bool {
	hay = strings.ToLower(hay)
	needle = strings.ToLower(strings.TrimSpace(needle))
	if needle == "" {
		return false
	}
	for _, part := range strings.FieldsFunc(hay, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if part == needle {
			return true
		}
	}
	return false
}
