package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func parseSeoPlanItems(raw any, projectID int64) []seoPlanItem {
	var rows []any
	switch t := raw.(type) {
	case []any:
		rows = t
	case []map[string]any:
		for _, m := range t {
			rows = append(rows, m)
		}
	default:
		return nil
	}
	var out []seoPlanItem
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		it := seoPlanItem{
			ProjectID:  int64Val(m["project_id"]),
			AssigneeID: int64Val(m["assignee_id"]),
			Title:      strVal(m["title"]),
			Hours:      floatVal(m["hours"]),
			Due:        strVal(m["due"]),
			Kind:       firstNonEmpty(strVal(m["kind"]), "seo_offsite"),
		}
		if it.ProjectID <= 0 {
			it.ProjectID = projectID
		}
		if it.Hours <= 0 {
			it.Hours = 1
		}
		if it.Title != "" && it.ProjectID > 0 {
			out = append(out, it)
		}
		if len(out) >= 8 {
			break
		}
	}
	return spreadSeoOffsiteDues(out)
}

func spreadSeoOffsiteDues(items []seoPlanItem) []seoPlanItem {
	now := time.Now().In(lagosLocation())
	for i := range items {
		if items[i].Kind == "" {
			items[i].Kind = "seo_offsite"
		}
		if items[i].Due != "" {
			continue
		}
		items[i].Due = now.AddDate(0, 0, 7*(i+1)).Format("2006-01-02")
	}
	return items
}

func seoPlanPayload(items []seoPlanItem) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"project_id":  it.ProjectID,
			"assignee_id": it.AssigneeID,
			"title":       it.Title,
			"hours":       it.Hours,
			"due":         it.Due,
			"kind":        it.Kind,
		})
	}
	return out
}

func (s *Server) withSEODeliveryContext(ctx context.Context, req contract.Request, p Project) contract.Request {
	if s == nil || s.store == nil || p.ID == 0 {
		return req
	}
	overlay := s.store.deliveryOverlay(ctx, nil, p.Slug)
	data := contract.ContextObject(req.ContextData)
	data["delivery"] = map[string]any{
		"people":     overlay["people"],
		"stage":      p.Stage,
		"suggest":    overlay["suggest_slots"],
		"engagement": overlay["engagement"],
	}
	if bets, ok := overlay["bets"].([]map[string]any); ok {
		for _, b := range bets {
			if int64Val(b["id"]) == p.ID {
				data["delivery"].(map[string]any)["bet"] = b
				break
			}
		}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return req
	}
	req.ContextData = raw
	return req
}

func (s *Server) startSeoAutomation(ctx context.Context, id int64) bool {
	if s == nil || s.store == nil || id <= 0 {
		return false
	}
	_, _ = s.store.MintSeoPackToken(ctx, id)
	d, err := s.store.GetSeoBrief(ctx, id)
	if err != nil || d.Status == "rejected" {
		return false
	}
	actor, _ := s.officeActor(ctx)
	return s.fileSeoOffsitePlan(ctx, actor, d)
}

func (s *Server) afterSeoApproved(ctx context.Context, actor User, id int64) bool {
	_, _ = s.store.MintSeoPackToken(ctx, id)
	if s.hurrySeoOffsite(ctx, actor, id) {
		return true
	}
	d, err := s.store.GetSeoBrief(ctx, id)
	if err != nil {
		return false
	}
	return s.fileSeoOffsitePlan(ctx, actor, d)
}

func (s *Server) hurrySeoOffsite(ctx context.Context, actor User, id int64) bool {
	jobs, err := s.store.ListDeliveryPlansForSeo(ctx, id)
	if err != nil {
		return false
	}
	wrote := false
	for _, j := range jobs {
		if j.Proposed() {
			if err := s.applyJobStamp(ctx, actor, j, "accepted"); err == nil {
				wrote = true
			}
			continue
		}
		if j.Status == "accepted" {
			wrote = true
		}
	}
	return wrote
}

func (s *Server) stopSeoAutomation(ctx context.Context, id int64) {
	if s == nil || s.store == nil || id <= 0 {
		return
	}
	jobs, err := s.store.ListDeliveryPlansForSeo(ctx, id)
	if err != nil {
		return
	}
	for _, j := range jobs {
		if !j.Proposed() {
			continue
		}
		if err := s.store.SetDeskJobStatus(ctx, j.ID, "rejected"); err != nil {
			continue
		}
		_, _ = s.store.InsertJobMessage(ctx, JobMessage{
			JobID: j.ID, AuthorKind: "system",
			Body: "SEO brief was turned away. This offsite plan will not write.",
		})
	}
}

func (s *Server) fileSeoOffsitePlan(ctx context.Context, actor User, d SeoBrief) bool {
	if s == nil || s.store == nil || d.ID <= 0 || d.Status == "rejected" {
		return false
	}
	items := d.PlanItems
	if len(items) == 0 {
		return false
	}
	existing, err := s.store.ListDeliveryPlansForSeo(ctx, d.ID)
	if err == nil {
		for _, j := range existing {
			if j.Status != "rejected" {
				return true
			}
		}
	}
	for i := range items {
		if items[i].ProjectID <= 0 {
			items[i].ProjectID = d.ProjectID
		}
	}
	items = spreadSeoOffsiteDues(items)
	sd := map[string]any{
		"save_plan":    true,
		"cannot_send":  true,
		"task_type":    "plan",
		"items":        seoPlanPayload(items),
		"seo_brief_id": d.ID,
		"project_id":   d.ProjectID,
		"kind":         "seo_offsite",
	}
	output := d.Rationale
	if output == "" {
		output = d.Body
	}
	if fn := s.workers["delivery"]; fn != nil {
		p, _ := s.store.GetProject(ctx, d.ProjectID)
		overlay := s.store.deliveryOverlay(ctx, &actor, p.Slug)
		overlay["action"] = "plan"
		overlay["channel"] = "desk"
		overlay["seo_offsite"] = true
		overlay["seed_items"] = seoPlanPayload(items)
		overlay["pin_project_id"] = d.ProjectID
		overlay["cannot_send"] = true
		raw, _ := json.Marshal(overlay)
		resp, err := fn(ctx, contract.Request{
			TaskDescription: "Apportion this SEO offsite roadmap into a week plan. Spread due dates over coming weeks, not all this week. kind seo_offsite. Stay inside slots; extras still in items (backlog). Do not invent a person. cannot_send. Writes after the delay unless Turn away.",
			ContextData:     raw,
		})
		if err == nil && resp.StructuredData != nil && boolVal(resp.StructuredData["save_plan"]) {
			if parsed := parseSeoPlanItems(resp.StructuredData["items"], d.ProjectID); len(parsed) > 0 {
				sd["items"] = seoPlanPayload(parsed)
			}
			if resp.OutputText != "" {
				output = resp.OutputText
			}
		}
	}
	stampAutoAt(sd, time.Now())
	title := "SEO offsite · " + d.ProjectName
	if d.Title != "" {
		title = "SEO offsite · " + d.Title
	}
	job, err := s.store.InsertDeskJob(ctx, jobFromWorker("delivery", "seo", title, sd, output))
	if err != nil {
		return false
	}
	if output != "" {
		_, _ = s.store.InsertJobMessage(ctx, JobMessage{JobID: job.ID, AuthorKind: "worker", Body: output})
	}
	return true
}
