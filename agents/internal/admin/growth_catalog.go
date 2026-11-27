package admin

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/lms"
)

func publicOffer(p Project) bool {
	st := strings.ToLower(strings.TrimSpace(p.Stage))
	return st != "parked" && st != "killed"
}

func publicGrowthOffers(projects []Project) []map[string]any {
	list := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		if !publicOffer(p) {
			continue
		}
		list = append(list, map[string]any{
			"name":      p.Name,
			"slug":      p.Slug,
			"one_liner": p.OneLiner,
			"url":       p.URL,
		})
	}
	return list
}

func attachGrowthCatalog(data map[string]any, projects []Project) {
	if data == nil {
		return
	}
	data["company"] = "TheMobileProf Technologies"
	data["company_city"] = "Lagos"
	data["products"] = publicGrowthOffers(projects)
}

func attachCustomerOffer(data map[string]any, p Project) {
	if data == nil {
		return
	}
	data["project_id"] = p.ID
	data["project_slug"] = p.Slug
	data["project"] = map[string]any{
		"name":      p.Name,
		"slug":      p.Slug,
		"one_liner": p.OneLiner,
		"url":       p.URL,
	}
}

const (
	offerNamed   = "named"
	offerSticky  = "sticky"
	offerSeveral = "several"
	offerNone    = "none"
)

func pickWhatsAppOffer(projects []Project, task string, sticky Project) (Project, string) {
	var hits []Project
	blob := strings.ToLower(task)
	for _, p := range projects {
		if projectMentioned(blob, p) && publicOffer(p) {
			hits = append(hits, p)
		}
	}
	if len(hits) == 1 {
		return hits[0], offerNamed
	}
	if len(hits) > 1 {
		return Project{}, offerSeveral
	}
	if sticky.ID > 0 && publicOffer(sticky) {
		return sticky, offerSticky
	}
	return Project{}, offerNone
}

func (s *Server) stickyWhatsAppOffer(ctx context.Context, data map[string]any, projects []Project) Project {
	if s == nil || s.store == nil || data == nil {
		return Project{}
	}
	phone := normalizePhone(strVal(data["from"]))
	if phone == "" {
		return Project{}
	}
	hit, ok, err := s.store.LookupCRMContact(ctx, "", phone, "")
	if err != nil || !ok || hit.ProjectID == 0 {
		return Project{}
	}
	for _, p := range projects {
		if p.ID == hit.ProjectID {
			return p
		}
	}
	return Project{}
}

func (s *Server) rememberWhatsAppOffer(ctx context.Context, phone string, p Project) {
	if s == nil || s.store == nil || p.ID == 0 {
		return
	}
	phone = normalizePhone(phone)
	if phone == "" {
		return
	}
	_, _ = s.store.UpsertCRMContact(ctx, CRMContact{
		ProjectID: p.ID,
		Phone:     phone,
		Offer:     p.Name,
		Source:    "whatsapp",
	})
}

func (s *Server) bindWhatsAppOffer(ctx context.Context, data map[string]any, task string, projects []Project) Project {
	if data == nil {
		return Project{}
	}
	p, kind := pickWhatsAppOffer(projects, task, s.stickyWhatsAppOffer(ctx, data, projects))
	if p.ID == 0 {
		return Project{}
	}
	attachCustomerOffer(data, p)
	if kind == offerSticky {
		data["project_sticky"] = true
	}
	if kind == offerNamed {
		s.rememberWhatsAppOffer(ctx, strVal(data["from"]), p)
	}
	return p
}

func academyAsk(task string, p Project) bool {
	blob := strings.ToLower(strings.TrimSpace(task + " " + p.Slug + " " + p.Name))
	for _, w := range []string{"academy", "lms", "course", "tuition", "syllabus", "scholarship"} {
		if strings.Contains(blob, w) {
			return true
		}
	}
	return false
}

func (s *Server) attachGrowthAcademyCatalog(ctx context.Context, data map[string]any, task string, p Project) {
	if data == nil || !academyAsk(task, p) {
		return
	}
	if s.lms == nil {
		s.lms = lms.NewClient("")
	}
	cat, err := s.lms.Get(ctx)
	if err != nil {
		return
	}
	matched := lms.Match(cat, task)
	data["catalog_live"] = lms.PromptBlock(cat, matched)
	if cards := lms.Cards(matched); len(cards) > 0 {
		data["catalog_cards"] = cards
	}
}
