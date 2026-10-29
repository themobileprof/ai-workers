package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/growth"
)

type deckAskForm struct {
	Action   string
	Audience string
	Beats    string
	Title    string
	Feedback string
}

func deckAskFromForm(r *http.Request) deckAskForm {
	return deckAskForm{
		Action:   r.FormValue("action"),
		Audience: r.FormValue("audience"),
		Beats:    r.FormValue("beats"),
		Title:    r.FormValue("title"),
		Feedback: r.FormValue("human_feedback"),
	}
}

func (s *Server) growthReady() bool {
	if s == nil || s.workers == nil {
		return false
	}
	return s.workers["growth"] != nil
}

func growthFlash(ok, step string) string {
	switch ok {
	case "drafted":
		return "Session started. Chat the copy for this slide. The pack URL updates every turn. Chat never Accepts."
	case "chat":
		return "Worker replied. The pack updated. Chat a tighter line, say next, or skip. Accept when the ten slides are right."
	case "deck_ok":
		return "Accepted. The pack was already live. The clerk does not email the deck."
	case "deck_no":
		return "Turned away. The pack URL is unpublished."
	default:
		return ""
	}
}

func (s *Server) growthGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireDesk(w, r, "growth")
	if u == nil {
		return
	}
	decks, err := s.store.ListGrowthDecks(r.Context(), 0)
	projects, _ := s.store.ListProjects(r.Context())
	data := pageData{
		Title:     "Growth",
		Nav:       "growth",
		User:      u,
		Decks:     decks,
		Projects:  projects,
		DeckKinds: growth.Specs(),
		CanGrowth: true,
		Flash:     growthFlash(r.URL.Query().Get("ok"), r.URL.Query().Get("step")),
	}
	if pid, _ := strconv.ParseInt(r.URL.Query().Get("project_id"), 10, 64); pid > 0 {
		if p, err := s.store.GetProject(r.Context(), pid); err == nil {
			data.Project = &p
			owned, _ := s.store.ListGrowthDecks(r.Context(), p.ID)
			data.DeckWalk = deckWalk(owned, p)
			data.DeckSeed = loadDeckSeed(p, owned)
			s.attachLiveDeck(r.Context(), &data, owned)
		}
	}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "growth", data)
}

func (s *Server) growthStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireDesk(w, r, "growth")
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.store.SetGrowthDeckStatus(r.Context(), id, status); err != nil {
			decks, _ := s.store.ListGrowthDecks(r.Context(), 0)
			projects, _ := s.store.ListProjects(r.Context())
			s.render(w, "growth", pageData{
				Title:     "Growth",
				Nav:       "growth",
				User:      u,
				Decks:     decks,
				Projects:  projects,
				DeckKinds: growth.Specs(),
				CanGrowth: s.growthReady(),
				Error:     err.Error(),
			})
			return
		}
		mapped := "accepted"
		if status == "rejected" {
			mapped = "rejected"
		}
		_ = s.store.StampJobsByRef(r.Context(), "growth_deck", id, mapped)
		ok := "deck_ok"
		if status == "rejected" {
			ok = "deck_no"
		}
		http.Redirect(w, r, growthNext(r, "/admin/growth?ok="+ok), http.StatusSeeOther)
	}
}

func growthNext(r *http.Request, fallback string) string {
	_ = r.ParseForm()
	out := fallback
	if next := strings.TrimSpace(r.FormValue("next")); strings.HasPrefix(next, "/admin/") {
		out = next
		if strings.Contains(fallback, "ok=deck_no") {
			out = withQuery(out, "ok", "deck_no")
		} else if strings.Contains(fallback, "ok=deck_ok") {
			out = withQuery(out, "ok", "deck_ok")
		}
	}
	if strings.Contains(out, "/admin/projects/") {
		return withQuery(out, "tab", "deck")
	}
	if strings.Contains(fallback, "ok=deck_no") {
		return withQuery(out, "tab", "rejected")
	}
	if strings.Contains(fallback, "ok=deck_ok") {
		return withQuery(out, "tab", "accepted")
	}
	return out
}

func (s *Server) growthAskPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "growth")
	if u == nil {
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	p, err := s.store.GetProject(r.Context(), pid)
	if err != nil {
		decks, _ := s.store.ListGrowthDecks(r.Context(), 0)
		projects, _ := s.store.ListProjects(r.Context())
		s.render(w, "growth", pageData{Title: "Growth", Nav: "growth", User: u, Decks: decks, Projects: projects, DeckKinds: growth.Specs(), CanGrowth: true, Error: "pick a project"})
		return
	}
	saved, opener, err := s.runDeckAsk(r.Context(), p, deckAskFromForm(r))
	if err != nil {
		decks, _ := s.store.ListGrowthDecks(r.Context(), 0)
		projects, _ := s.store.ListProjects(r.Context())
		s.render(w, "growth", pageData{Title: "Growth", Nav: "growth", User: u, Decks: decks, Projects: projects, DeckKinds: growth.Specs(), CanGrowth: true, Error: err.Error()})
		return
	}
	s.ensureDeckJob(r.Context(), saved, opener)
	http.Redirect(w, r, withQuery(withQuery("/admin/growth?ok=drafted", "project_id", strconv.FormatInt(p.ID, 10)), "step", saved.Kind), http.StatusSeeOther)
}

func (s *Server) projectDeck(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "growth")
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	saved, opener, err := s.runDeckAsk(r.Context(), p, deckAskFromForm(r))
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "Growth did not save a deck: "+err.Error(), ""))
		return
	}
	s.ensureDeckJob(r.Context(), saved, opener)
	http.Redirect(w, r, withQuery(withQuery(withQuery("/admin/projects/"+strconv.FormatInt(p.ID, 10), "ok", "drafted"), "tab", "deck"), "step", saved.Kind), http.StatusSeeOther)
}

func (s *Server) runDeckAsk(ctx context.Context, p Project, form deckAskForm) (GrowthDeck, string, error) {
	if !cardHasDeckFacts(p) {
		return GrowthDeck{}, "", errors.New("write the Card one-liner first")
	}
	decks, _ := s.store.ListGrowthDecks(ctx, p.ID)
	if live := liveProposedSession(decks); live.ID > 0 && resolveDeckAction(form.Action, decks) != "plan" {
		return live, "", nil
	}
	action := resolveDeckAction(form.Action, decks)
	seed := buildDeckSeed(p, decks, form)
	slides := seedSessionSlides(p)
	req, err := deckRequest(p, action, seed, form.Feedback, decks, slides, 1)
	if err != nil {
		return GrowthDeck{}, "", err
	}
	var d GrowthDeck
	opener := ""
	if s.workers != nil {
		if fn := s.workers["growth"]; fn != nil {
			resp, err := fn(ctx, req)
			if err == nil && resp.Status == contract.StatusSuccess {
				opener = strings.TrimSpace(resp.OutputText)
				d, err = growthDeckFromStructured(resp.StructuredData, p.ID, slides)
				if err != nil {
					d = GrowthDeck{}
				}
			}
		}
	}
	if d.Kind == "" {
		d = deckFromSeed(p, action, seed, form, slides)
	} else {
		mergeGrowthDeck(&d, seed, form)
		if len(d.Slides) == 0 {
			d.Slides = slides
		}
	}
	if strings.TrimSpace(d.Title) == "" && strings.TrimSpace(d.Body) == "" && len(d.Slides) == 0 {
		return GrowthDeck{}, "", errors.New("nothing on the Card to file — write the one-liner")
	}
	saved, err := s.store.InsertGrowthDeck(ctx, d)
	if err != nil {
		return GrowthDeck{}, "", err
	}
	return saved, opener, nil
}

func deckRequest(p Project, action string, seed deckSeed, feedback string, decks []GrowthDeck, slides []growth.DeckSlide, current int) (contract.Request, error) {
	action = resolveDeckAction(action, decks)
	ctx := map[string]any{
		"action":         action,
		"channel":        "desk",
		"cannot_send":    true,
		"human_feedback": strings.TrimSpace(feedback),
		"audience_seed":  seed.Audience,
		"kawasaki":       growth.KawasakiSlides(),
		"slides":         slides,
		"current_slide":  growth.ClampSlide(current),
	}
	attachProductProject(ctx, p)
	ctx["walk"] = deckWalkContext(decks)
	raw, err := json.Marshal(ctx)
	if err != nil {
		return contract.Request{}, err
	}
	task := "Start a Kawasaki 10-slide session for " + p.Name + " (" + p.Slug + "). Work slide " + strconv.Itoa(growth.ClampSlide(current)) + " first. Use only Card facts. Do not invent traction. Do not send."
	if action == "plan" {
		task = "Write a longer business-plan outline for " + p.Name + " with the same ten Kawasaki headings. Not a CAC filing. Do not send."
	}
	if feedback != "" {
		task += " Human notes:\n" + feedback
	}
	return contract.Request{TaskDescription: task, ContextData: raw}, nil
}

func deckFromSeed(p Project, action string, seed deckSeed, form deckAskForm, slides []growth.DeckSlide) GrowthDeck {
	action = resolveDeckAction(action, nil)
	if len(slides) == 0 {
		slides = seedSessionSlides(p)
	}
	return GrowthDeck{
		ProjectID:    p.ID,
		Kind:         action,
		Audience:     firstNonEmpty(form.Audience, seed.Audience),
		Title:        firstNonEmpty(form.Title, seed.Title),
		Body:         growth.SlidesMarkdown(slides),
		Rationale:    strings.TrimSpace(form.Feedback),
		Slides:       slides,
		CurrentSlide: 1,
	}
}

func mergeGrowthDeck(d *GrowthDeck, seed deckSeed, form deckAskForm) {
	if d.Audience == "" {
		d.Audience = firstNonEmpty(form.Audience, seed.Audience)
	}
	if d.Title == "" {
		d.Title = seed.Title
	}
	if d.Body == "" {
		if d.Kind == "audience" {
			d.Body = firstNonEmpty(form.Audience, seed.Audience)
		} else {
			d.Body = firstNonEmpty(form.Beats, seed.Beats)
		}
	}
	if extra := strings.TrimSpace(form.Feedback); extra != "" && d.Rationale == "" {
		d.Rationale = extra
	}
}

func growthDeckFromStructured(sd map[string]any, projectID int64, base []growth.DeckSlide) (GrowthDeck, error) {
	kind := strVal(sd["action"])
	if kind == "" {
		kind = strVal(sd["kind"])
	}
	if _, ok := growth.Lookup(kind); !ok {
		kind = "session"
	}
	slides := growth.MergeSlides(base, growth.ParseSlides(sd["slides"]))
	current := int(int64Val(sd["current_slide"]))
	if current < 1 {
		current = 1
	}
	body := strVal(sd["body"])
	if body == "" {
		body = growth.SlidesMarkdown(slides)
	}
	d := GrowthDeck{
		ProjectID:    projectID,
		Kind:         kind,
		Audience:     strVal(sd["audience"]),
		Title:        strVal(sd["title"]),
		Body:         body,
		Rationale:    strVal(sd["rationale"]),
		Slides:       slides,
		CurrentSlide: growth.ClampSlide(current),
	}
	if strVal(sd["title"]) == "" && strVal(sd["audience"]) == "" && strVal(sd["body"]) == "" && !slidesHaveCopy(slides) {
		return GrowthDeck{}, errors.New("worker returned no audience, slides, or pack")
	}
	if d.Kind == "audience" && strVal(sd["body"]) == "" && d.Audience != "" {
		d.Body = d.Audience
	}
	if d.Title == "" {
		d.Title = d.KindLabel()
	}
	return d, nil
}

func slidesHaveCopy(slides []growth.DeckSlide) bool {
	for _, s := range slides {
		if strings.TrimSpace(s.Copy) != "" {
			return true
		}
	}
	return false
}

func (s *Server) EnrichGrowth(ctx context.Context, req contract.Request) contract.Request {
	if s == nil || s.store == nil {
		return req
	}
	data := contract.ContextObject(req.ContextData)
	if id := int64Val(data["ref_id"]); id > 0 && strings.EqualFold(strVal(data["ref_kind"]), "growth_deck") {
		if d, err := s.store.GetGrowthDeck(ctx, id); err == nil {
			s.attachDeckSession(ctx, data, d)
			raw, _ := json.Marshal(data)
			req.ContextData = raw
			return req
		}
	}
	projects, _ := s.store.ListProjects(ctx)
	p, ok := matchProject(projects, data, req.TaskDescription)
	if !ok {
		return req
	}
	decks, _ := s.store.ListGrowthDecks(ctx, p.ID)
	live := liveProposedSession(decks)
	if !growth.LooksLike(data, req.TaskDescription) && live.ID == 0 {
		return req
	}
	if live.ID > 0 {
		s.attachDeckSession(ctx, data, live)
	} else {
		attachProductProject(data, p)
		data["action"] = "session"
		data["slides"] = seedSessionSlides(p)
		data["current_slide"] = 1
		data["kawasaki"] = growth.KawasakiSlides()
		data["cannot_send"] = true
	}
	raw, _ := json.Marshal(data)
	req.ContextData = raw
	return req
}

func (s *Server) attachDeckSession(ctx context.Context, data map[string]any, d GrowthDeck) {
	p, err := s.store.GetProject(ctx, d.ProjectID)
	if err == nil {
		attachProductProject(data, p)
	}
	data["action"] = "session"
	data["ref_kind"] = "growth_deck"
	data["ref_id"] = d.ID
	data["growth_deck_id"] = d.ID
	data["slides"] = d.Slides
	data["current_slide"] = growth.ClampSlide(d.CurrentSlide)
	data["audience"] = d.Audience
	data["kawasaki"] = growth.KawasakiSlides()
	data["cannot_send"] = true
	data["pack_path"] = d.PackPath()
}

func (s *Server) applyDeckResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	if !boolVal(resp.StructuredData["save_deck"]) && growth.ParseSlides(resp.StructuredData["slides"]) == nil {
		return
	}
	data := contract.ContextObject(req.ContextData)
	if strings.EqualFold(strVal(data["channel"]), "whatsapp") {
		return
	}
	var existing GrowthDeck
	id := int64Val(firstNonEmptyAny(data["growth_deck_id"], data["ref_id"], resp.StructuredData["growth_deck_id"]))
	if id > 0 {
		if d, err := s.store.GetGrowthDeck(ctx, id); err == nil && d.Status == "proposed" {
			existing = d
		}
	}
	p, err := s.resolveSeoProject(ctx, data, resp.StructuredData)
	if err != nil && existing.ProjectID > 0 {
		p, err = s.store.GetProject(ctx, existing.ProjectID)
	}
	if err != nil {
		return
	}
	if existing.ID == 0 {
		if live := liveProposedSession(mustListDecks(s, ctx, p.ID)); live.ID > 0 {
			existing = live
		}
	}
	d, err := growthDeckFromStructured(resp.StructuredData, p.ID, existing.Slides)
	if err != nil {
		return
	}
	var saved GrowthDeck
	if existing.ID > 0 {
		d.ID = existing.ID
		saved, err = s.store.UpdateGrowthDeckDraft(ctx, d)
	} else {
		saved, err = s.store.InsertGrowthDeck(ctx, d)
	}
	if err != nil {
		return
	}
	resp.StructuredData["growth_deck_id"] = saved.ID
	resp.StructuredData["pack_path"] = saved.PackPath()
	resp.StructuredData["current_slide"] = saved.CurrentSlide
}

func mustListDecks(s *Server, ctx context.Context, projectID int64) []GrowthDeck {
	list, _ := s.store.ListGrowthDecks(ctx, projectID)
	return list
}

func (s *Server) ensureDeckJob(ctx context.Context, d GrowthDeck, opener string) {
	if d.ID <= 0 {
		return
	}
	if _, err := s.store.FindJobByRef(ctx, "growth", "growth_deck", d.ID); err == nil {
		return
	}
	s.linkJob(ctx, "growth", d.Title, deckOpener(d, opener), "growth_deck", d.ID, map[string]any{
		"kind":          d.Kind,
		"audience":      d.Audience,
		"current_slide": d.CurrentSlide,
	})
}

func deckOpener(d GrowthDeck, output string) string {
	if s := strings.TrimSpace(output); s != "" {
		return s
	}
	cur := d.Current()
	var b strings.Builder
	b.WriteString(d.SlideLabel() + "\n\n")
	if copy := strings.TrimSpace(cur.Copy); copy != "" {
		b.WriteString(copy + "\n\n")
	}
	b.WriteString("Tighter, next, or skip? Chat never Accepts.")
	return b.String()
}

func (s *Server) attachLiveDeck(ctx context.Context, data *pageData, decks []GrowthDeck) {
	if data == nil {
		return
	}
	live := liveProposedSession(decks)
	if live.ID == 0 {
		return
	}
	ld := live
	data.LiveDeck = &ld
	if s == nil || s.store == nil {
		return
	}
	job, err := s.store.FindJobByRef(ctx, "growth", "growth_deck", live.ID)
	if err != nil {
		return
	}
	j := job
	data.DeckJob = &j
	msgs, _ := s.store.ListJobMessages(ctx, job.ID)
	data.Messages = msgs
}
