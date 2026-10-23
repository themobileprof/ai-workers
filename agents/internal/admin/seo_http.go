package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/seo"
)

func (s *Server) WithSEO(fn PlaceFunc) {
	if s != nil {
		s.seo = fn
		s.WithWorker("seo", fn)
	}
}

func seoFlash(ok, step string) string {
	switch ok {
	case "drafted":
		if spec, found := seo.Lookup(step); found {
			if next := nextSeoWalkLabel(step); next != "" {
				return spec.Label + " pack is live. Next: " + next + ". Turn away if a fact is invented."
			}
			return spec.Label + " pack is live. Download it or give Cursor the URL."
		}
		return "Pack URL is live. Download it or give Cursor the URL. Offsite items are a Delivery plan that writes after the delay unless you Turn away."
	case "seo_ok":
		return "Accepted. The pack was already live. The clerk does not publish the site or open a GitHub PR."
	case "seo_plan":
		return "Accepted. Offsite work writes now. Turn away on Job chat if you still need to stop a later week."
	case "seo_no":
		return "Turned away. The pack URL is unpublished. Related Delivery plans will not write."
	default:
		return ""
	}
}

func (s *Server) seoGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireDesk(w, r, "seo")
	if u == nil {
		return
	}
	briefs, err := s.store.ListSeoBriefs(r.Context(), 0)
	projects, _ := s.store.ListProjects(r.Context())
	data := pageData{
		Title:     "SEO",
		Nav:       "seo",
		User:      u,
		SeoBriefs: briefs,
		Projects:  projects,
		SeoKinds:  seo.Specs(),
		CanSEO:    true,
		Flash:     seoFlash(r.URL.Query().Get("ok"), r.URL.Query().Get("step")),
	}
	if pid, _ := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64); pid > 0 {
		if p, err := s.store.GetProject(r.Context(), pid); err == nil {
			data.Project = &p
			owned, _ := s.store.ListSeoBriefs(r.Context(), p.ID)
			data.SeoWalk = seoWalk(owned, p.URL)
			data.SeoSeed = loadSeoSeed(p, owned)
		}
	}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "seo", data)
}

func (s *Server) seoStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireDesk(w, r, "seo")
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.store.SetSeoStatus(r.Context(), id, status); err != nil {
			briefs, _ := s.store.ListSeoBriefs(r.Context(), 0)
			projects, _ := s.store.ListProjects(r.Context())
			s.render(w, "seo", pageData{
				Title:     "SEO",
				Nav:       "seo",
				User:      u,
				SeoBriefs: briefs,
				Projects:  projects,
				SeoKinds:  seo.Specs(),
				CanSEO:    s.seo != nil,
				Error:     err.Error(),
			})
			return
		}
		mapped := "accepted"
		if status == "rejected" {
			mapped = "rejected"
		}
		_ = s.store.StampJobsByRef(r.Context(), "seo_brief", id, mapped)
		ok := "seo_ok"
		if status == "rejected" {
			ok = "seo_no"
			s.stopSeoAutomation(r.Context(), id)
		} else {
			if s.afterSeoApproved(r.Context(), *u, id) {
				ok = "seo_plan"
			}
		}
		http.Redirect(w, r, seoNext(r, "/admin/seo?ok="+ok), http.StatusSeeOther)
	}
}

func seoNext(r *http.Request, fallback string) string {
	_ = r.ParseForm()
	out := fallback
	if next := strings.TrimSpace(r.FormValue("next")); strings.HasPrefix(next, "/admin/") {
		out = next
		if strings.Contains(fallback, "ok=seo_no") {
			out = withQuery(out, "ok", "seo_no")
		} else if strings.Contains(fallback, "ok=seo_ok") || strings.Contains(fallback, "ok=seo_plan") {
			out = withQuery(out, "ok", r.FormValue("ok"))
			if strings.Contains(fallback, "ok=seo_plan") {
				out = withQuery(out, "ok", "seo_plan")
			} else {
				out = withQuery(out, "ok", "seo_ok")
			}
		}
	}
	if strings.Contains(out, "/admin/projects/") {
		return withQuery(out, "tab", "seo")
	}
	if strings.Contains(fallback, "ok=seo_no") {
		return withQuery(out, "tab", "rejected")
	}
	if strings.Contains(fallback, "ok=seo_ok") || strings.Contains(fallback, "ok=seo_plan") {
		return withQuery(out, "tab", "accepted")
	}
	return out
}

func (s *Server) seoAskPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "seo")
	if u == nil {
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	p, err := s.store.GetProject(r.Context(), pid)
	if err != nil {
		briefs, _ := s.store.ListSeoBriefs(r.Context(), 0)
		projects, _ := s.store.ListProjects(r.Context())
		s.render(w, "seo", pageData{Title: "SEO", Nav: "seo", User: u, SeoBriefs: briefs, Projects: projects, SeoKinds: seo.Specs(), CanSEO: true, Error: "pick a project"})
		return
	}
	saved, err := s.runSeoAsk(r.Context(), p, seoAskFromForm(r))
	if err != nil {
		briefs, _ := s.store.ListSeoBriefs(r.Context(), 0)
		projects, _ := s.store.ListProjects(r.Context())
		s.render(w, "seo", pageData{Title: "SEO", Nav: "seo", User: u, SeoBriefs: briefs, Projects: projects, SeoKinds: seo.Specs(), CanSEO: true, Error: err.Error()})
		return
	}
	s.linkJob(r.Context(), "seo", saved.Title, saved.Rationale, "seo_brief", saved.ID, map[string]any{"kind": saved.Kind, "url": saved.URL})
	http.Redirect(w, r, withQuery(withQuery("/admin/seo?ok=drafted", "project_id", strconv.FormatInt(p.ID, 10)), "step", saved.Kind), http.StatusSeeOther)
}

func (s *Server) projectSEO(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "seo")
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	saved, err := s.runSeoAsk(r.Context(), p, seoAskFromForm(r))
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "SEO did not save a brief: "+err.Error(), ""))
		return
	}
	s.linkJob(r.Context(), "seo", saved.Title, saved.Rationale, "seo_brief", saved.ID, map[string]any{"kind": saved.Kind, "url": saved.URL})
	http.Redirect(w, r, withQuery(withQuery(withQuery("/admin/projects/"+strconv.FormatInt(p.ID, 10), "ok", "drafted"), "tab", "seo"), "step", saved.Kind), http.StatusSeeOther)
}

func (s *Server) runSeoAsk(ctx context.Context, p Project, form seoAskForm) (SeoBrief, error) {
	briefs, _ := s.store.ListSeoBriefs(ctx, p.ID)
	action := resolveSeoAction(form.Action, briefs)
	pageURL := firstNonEmpty(strings.TrimSpace(form.URL), strings.TrimSpace(p.URL))
	pages := fetchSitePages(pageURL)
	if len(pages) == 0 {
		pages = fetchHomePage(pageURL)
	}
	seed := applyAskForm(buildSeoSeed(pages, p, briefs), form)
	req, err := seoRequest(p, action, pageURL, seed, form.Feedback, briefs, pages)
	if err != nil {
		return SeoBrief{}, err
	}
	req = s.withSEOSettings(ctx, req)
	req = s.withSEODeliveryContext(ctx, req, p)
	var d SeoBrief
	if s.seo != nil {
		resp, err := s.seo(ctx, req)
		if err == nil && resp.Status == contract.StatusSuccess {
			d, err = seoBriefFromStructured(resp.StructuredData, p.ID)
			if err != nil {
				d = SeoBrief{}
			}
		}
	}
	if d.Kind == "" {
		d = briefFromSeed(p, action, seed, form)
	} else {
		mergeSeoBrief(&d, seed, form)
	}
	if d.URL == "" {
		d.URL = pageURL
	}
	if d.Keyword == "" {
		d.Keyword = seed.Keyword
	}
	if d.Title == "" && d.Body == "" && len(d.Flags) == 0 && len(d.PlanItems) == 0 {
		return SeoBrief{}, errors.New("nothing on the page to file — put a public URL on Card")
	}
	saved, err := s.store.InsertSeoBrief(ctx, d)
	if err != nil {
		return SeoBrief{}, err
	}
	if inv := inventoryFromContext(contract.ContextObject(req.ContextData)); len(inv) > 0 {
		_ = s.store.SetSeoInventory(ctx, saved.ID, inv)
		saved.Inventory = inv
	}
	if len(d.PlanItems) > 0 {
		_ = s.store.SetSeoPlanItems(ctx, saved.ID, d.PlanItems)
		saved.PlanItems = d.PlanItems
	}
	s.startSeoAutomation(ctx, saved.ID)
	if reloaded, err := s.store.GetSeoBrief(ctx, saved.ID); err == nil {
		saved = reloaded
	}
	return saved, nil
}

func seoRequest(p Project, action, pageURL string, seed seoSeed, feedback string, briefs []SeoBrief, pages []fetchedPage) (contract.Request, error) {
	action = resolveSeoAction(action, briefs)
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		pageURL = strings.TrimSpace(p.URL)
	}
	ctx := map[string]any{
		"action":         action,
		"channel":        "desk",
		"cannot_publish": true,
		"cannot_send":    true,
		"url":            pageURL,
		"keyword":        seed.Keyword,
		"human_feedback": strings.TrimSpace(feedback),
		"cite_seed":      seed.Cites,
		"faq_seed":       seedFAQs(seed.FAQs),
		"schema_seed":    seed.JSONLD,
		"project": map[string]any{
			"id":        p.ID,
			"name":      p.Name,
			"slug":      p.Slug,
			"one_liner": p.OneLiner,
			"url":       p.URL,
			"git_repo":  p.GitRepo,
		},
	}
	if len(pages) > 0 {
		ctx["fetched_pages"] = pages
	}
	attachFetchedSEO(ctx, pageURL)
	ctx["walk"] = seoWalkContext(briefs)
	raw, err := json.Marshal(ctx)
	if err != nil {
		return contract.Request{}, err
	}
	task := "Propose SEO for " + p.Name
	if pageURL != "" {
		task += " at " + pageURL
	}
	switch action {
	case "cite":
		task += ". Walk step 1: quotable facts. Start from context_data.cite_seed (already extracted from the page). Keep those sentences; rewrite only for clarity. structured_data.cites. No invented stats. Do not publish."
	case "faq":
		task += ". Walk step 2: FAQ. Start from context_data.faq_seed. Visible Q&A first (faqs), then FAQPage JSON-LD that matches. Do not invent places."
	case "schema":
		task += ". Walk step 3: schema. Start from context_data.schema_seed. Inventory schema_types and fetched_pages.json_ld. Organization is TheMobileProf Technologies unless the page names someone else. Flag robots.blocked_ai. Do not publish."
	case "inventory":
		task += ". The visible page is the source of truth (a designer may never have seen title/meta). Isolate keywords from keyword_inventory and keyword_map. Rewrite visible copy into searcher queries using only attested entities. Prefer gap phrases. Do not invent search volume."
	case "map":
		task += ". One primary phrase per URL from the visible copy. Cluster synonyms. Title tags must match the designed page. No volume."
	case "offsite":
		task += ". Walk step 5: Propose Delivery tasks (items) spread over coming weeks, not all this week. Use people and slots from context_data.delivery. Listings and partners already on the pages. Same legal name as schema. No cold outreach. cannot_send."
	case "onsite":
		task += ". Walk step 4: Write an onsite pack Cursor can implement. Weave walk.cite, walk.faq, and walk.schema into the pack. Customize the visible page (body, H1, subheads, CTAs) from what the user already saw, then title/meta so snippets match that design. Same offers and places. Do not invent volume. Do not publish."
	default:
		task += ". Fetch, keyword_inventory, keyword_map, and schema_types are already attached. Visible copy is the source of truth. Customize body and chrome together. Do not invent rankings or Core Web Vitals. Do not publish."
	}
	if feedback != "" {
		task += " Human notes:\n" + feedback
	}
	return contract.Request{TaskDescription: task, ContextData: raw}, nil
}

func (s *Server) EnrichSEO(ctx context.Context, req contract.Request) contract.Request {
	if s == nil || s.store == nil {
		return req
	}
	data := contract.ContextObject(req.ContextData)
	slug := firstNonEmpty(strVal(data["project_slug"]), strVal(data["project"]))
	if nested, ok := data["project"].(map[string]any); ok && slug == "" {
		slug = strVal(nested["slug"])
	}
	if slug == "" {
		attachFetchedSEO(data, strVal(data["url"]))
		raw, _ := json.Marshal(data)
		req.ContextData = raw
		return s.withSEOSettings(ctx, req)
	}
	p, err := s.store.GetProjectBySlug(ctx, slug)
	if err != nil {
		return req
	}
	data["project"] = map[string]any{
		"id": p.ID, "name": p.Name, "slug": p.Slug, "one_liner": p.OneLiner, "url": p.URL, "git_repo": p.GitRepo,
	}
	if strVal(data["url"]) == "" && p.URL != "" {
		data["url"] = p.URL
	}
	briefs, _ := s.store.ListSeoBriefs(ctx, p.ID)
	data["walk"] = seoWalkContext(briefs)
	action := strVal(data["action"])
	guessed := seo.Infer(data, req.TaskDescription)
	if action == "walk" || action == "next" || (action == "" && guessed == "onsite" && !strings.Contains(strings.ToLower(req.TaskDescription), "onsite") && !strings.Contains(strings.ToLower(req.TaskDescription), "cursor")) {
		data["action"] = resolveSeoAction("walk", briefs)
	} else if action == "" {
		data["action"] = guessed
	}
	attachFetchedSEO(data, strVal(data["url"]))
	seed := buildSeoSeed(pagesFromContext(data), p, briefs)
	data["cite_seed"] = seed.Cites
	data["faq_seed"] = seedFAQs(seed.FAQs)
	data["schema_seed"] = seed.JSONLD
	if strVal(data["keyword"]) == "" && seed.Keyword != "" {
		data["keyword"] = seed.Keyword
	}
	raw, _ := json.Marshal(data)
	req.ContextData = raw
	req = s.withSEOSettings(ctx, req)
	return s.withSEODeliveryContext(ctx, req, p)
}

func (s *Server) withSEOSettings(ctx context.Context, req contract.Request) contract.Request {
	if s == nil || s.store == nil {
		return req
	}
	st, err := s.store.Settings(ctx)
	if err != nil {
		return req
	}
	data := contract.ContextObject(req.ContextData)
	if v := strings.TrimSpace(st["seo.gsc_site_url"]); v != "" {
		data["gsc_site_url"] = v
	}
	if v := strings.TrimSpace(st["seo.ga4_property_id"]); v != "" {
		data["ga4_property_id"] = v
	}
	if snap, err := s.store.GetSeoGA4(ctx); err == nil && snap != nil {
		data["ga4"] = snap
	}
	if snap, err := s.store.GetSeoGSC(ctx); err == nil && snap != nil {
		data["gsc"] = snap
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return req
	}
	req.ContextData = raw
	return req
}

func attachFetchedSEO(data map[string]any, pageURL string) {
	if data == nil {
		return
	}
	pages := pagesFromContext(data)
	if len(pages) == 0 && pageURL != "" {
		pages = fetchSitePages(pageURL)
		if len(pages) > 0 {
			data["fetched_pages"] = pages
		}
	}
	if len(pages) == 0 {
		return
	}
	if _, ok := data["keyword_inventory"]; !ok {
		inv := keywordInventory(pages)
		if len(inv) > 0 {
			data["keyword_inventory"] = inv
			data["keyword_map"] = keywordMap(pages, inv)
		}
	}
	data["keyword_method"] = "The visible page (H1, headings, body, CTAs) is the source of truth — that is what the designer and the user already saw. Title and meta are often unread by the designer. Isolate phrases from the visible copy. Then customize everything: visible body, headings, CTAs, AND title/meta, in the same voice and offers already on the page. Do not invent a hidden SEO layer. No volume. cite_seed, faq_seed, and schema_seed are already extracted from the page — use them; rewrite only for clarity."
	if _, ok := data["schema_types"]; !ok {
		if types := schemaTypesFromPages(pages); len(types) > 0 {
			data["schema_types"] = types
		}
	}
	if _, ok := data["robots"]; !ok {
		if robots := fetchRobotsNotes(firstNonEmpty(pageURL, pages[0].URL)); robots != nil {
			data["robots"] = robots
		}
	}
	if _, ok := data["pagespeed"]; !ok {
		if psi := pagespeedSnapshot(firstNonEmpty(pageURL, pages[0].URL)); psi != nil {
			data["pagespeed"] = psi
		}
	}
}

func (s *Server) applySeoResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	if !boolVal(resp.StructuredData["save_brief"]) {
		return
	}
	data := contract.ContextObject(req.ContextData)
	p, err := s.resolveSeoProject(ctx, data, resp.StructuredData)
	if err != nil {
		return
	}
	d, err := seoBriefFromStructured(resp.StructuredData, p.ID)
	if err != nil {
		return
	}
	briefs, _ := s.store.ListSeoBriefs(ctx, p.ID)
	mergeSeoBrief(&d, buildSeoSeed(pagesFromContext(data), p, briefs), seoAskForm{})
	saved, err := s.store.InsertSeoBrief(ctx, d)
	if err != nil {
		return
	}
	if inv := inventoryFromContext(data); len(inv) > 0 {
		_ = s.store.SetSeoInventory(ctx, saved.ID, inv)
	}
	if len(d.PlanItems) > 0 {
		_ = s.store.SetSeoPlanItems(ctx, saved.ID, d.PlanItems)
	}
	s.startSeoAutomation(ctx, saved.ID)
	resp.StructuredData["seo_brief_id"] = saved.ID
}

func (s *Server) resolveSeoProject(ctx context.Context, body, sd map[string]any) (Project, error) {
	if id := int64Val(firstNonEmptyAny(body["project_id"], sd["project_id"])); id > 0 {
		return s.store.GetProject(ctx, id)
	}
	slug := firstNonEmpty(strVal(body["project_slug"]), strVal(sd["project_slug"]))
	if slug == "" {
		if nested, ok := sd["project"].(map[string]any); ok {
			slug = strVal(nested["slug"])
			if id := int64Val(nested["id"]); id > 0 {
				return s.store.GetProject(ctx, id)
			}
		}
		if nested, ok := body["project"].(map[string]any); ok {
			slug = strVal(nested["slug"])
			if id := int64Val(nested["id"]); id > 0 {
				return s.store.GetProject(ctx, id)
			}
		}
	}
	if slug == "" {
		return Project{}, errors.New("project is required")
	}
	return s.store.GetProjectBySlug(ctx, slug)
}

func (s *Server) internalSeoBriefsGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var pid int64
	if raw := strings.TrimSpace(r.URL.Query().Get("project_id")); raw != "" {
		pid, _ = strconv.ParseInt(raw, 10, 64)
	}
	list, err := s.store.ListSeoBriefs(r.Context(), pid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, seoBriefJSON(d))
	}
	writeJSON(w, map[string]any{"briefs": out})
}

func (s *Server) internalSeoBriefsPOST(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	sd := body
	if nested, ok := body["structured_data"].(map[string]any); ok {
		sd = nested
	}
	p, err := s.resolveSeoProject(r.Context(), body, sd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d, err := seoBriefFromStructured(sd, p.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	saved, err := s.store.InsertSeoBrief(r.Context(), d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.linkJob(r.Context(), "seo", saved.Title, saved.Rationale, "seo_brief", saved.ID, map[string]any{"kind": saved.Kind})
	s.startSeoAutomation(r.Context(), saved.ID)
	if reloaded, err := s.store.GetSeoBrief(r.Context(), saved.ID); err == nil {
		saved = reloaded
	}
	writeJSON(w, map[string]any{"ok": true, "committed": false, "status": saved.Status, "brief": seoBriefJSON(saved)})
}

func seoBriefJSON(d SeoBrief) map[string]any {
	return map[string]any{
		"id":           d.ID,
		"project_id":   d.ProjectID,
		"project_name": d.ProjectName,
		"kind":         d.Kind,
		"url":          d.URL,
		"keyword":      d.Keyword,
		"title":        d.Title,
		"meta":         d.Meta,
		"h1":           d.H1,
		"body":         d.Body,
		"flags":        d.Flags,
		"status":       d.Status,
		"rationale":    d.Rationale,
		"pack_path":    d.PackPath(),
		"created_at":   d.CreatedAt,
		"updated_at":   d.UpdatedAt,
	}
}
