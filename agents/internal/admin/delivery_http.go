package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func deliveryFlash(ok string) string {
	switch ok {
	case "saved":
		return "Task saved. A parked or killed project cannot hold work."
	case "done":
		return "Marked done."
	case "skipped":
		return "Skipped."
	case "plan":
		return "Plan proposed. It writes itself after the delay unless you Turn away. Accept still works now."
	case "queued":
		return "Held on the backlog. Promote when this stage has a free slot."
	case "pace":
		return "Pace saved. Delivery will not assign more hours than this."
	default:
		return ""
	}
}

func (s *Server) deliveryActor(ctx context.Context, from, channel string) (User, bool) {
	if s == nil || s.store == nil {
		return User{}, false
	}
	from = strings.TrimSpace(from)
	if from != "" {
		if u, ok, _ := s.store.LookupUserByPhone(ctx, from); ok {
			return u, true
		}
	}
	if officeDeliveryChannel(channel) {
		users, _ := s.store.ListUsers(ctx)
		for _, cand := range users {
			if cand.Active && cand.Role == "owner" {
				return cand, true
			}
		}
	}
	return User{}, false
}

func (s *Server) EnrichDelivery(ctx context.Context, req contract.Request) contract.Request {
	if s == nil || s.store == nil {
		return req
	}
	data := contract.ContextObject(req.ContextData)
	from := strVal(data["from"])
	slug := firstNonEmpty(strVal(data["project_slug"]), strVal(data["project"]))
	u, ok := s.deliveryActor(ctx, from, strVal(data["channel"]))
	if !ok {
		data["found"] = false
		data["access_denied"] = true
		raw, _ := json.Marshal(data)
		req.ContextData = raw
		return req
	}
	if slug == "" {
		title := strVal(data["meeting_title"])
		if title == "" {
			title = strVal(data["calendar_title"])
		}
		if title == "" {
			title = strVal(data["title"])
		}
		projects, _ := s.store.ListProjects(ctx)
		var visible []Project
		for _, p := range projects {
			if canSeeBet(u, p.Members) {
				visible = append(visible, p)
			}
		}
		slug = slugFromMeetingTitle(title, visible)
	}
	overlay := s.store.deliveryOverlay(ctx, &u, slug)
	for k, v := range data {
		if _, exists := overlay[k]; !exists {
			overlay[k] = v
		}
	}
	overlay["found"] = true
	overlay["actor_id"] = u.ID
	raw, _ := json.Marshal(overlay)
	req.ContextData = raw
	return req
}

func (s *Server) requireDelivery(w http.ResponseWriter, r *http.Request) *User {
	u := s.requireAdmin(w, r)
	if u == nil {
		return nil
	}
	if canOffice(u) || Handles(*u, "delivery") {
		return u
	}
	http.Error(w, "not assigned to Delivery", http.StatusForbidden)
	return nil
}

func (s *Server) deliveryGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireDelivery(w, r)
	if u == nil {
		return
	}
	s.render(w, "delivery", s.deliveryView(r, u, "", deliveryFlash(r.URL.Query().Get("ok"))))
}

func (s *Server) deliveryView(r *http.Request, u *User, errMsg, flash string) pageData {
	items, _ := s.store.ListWorkItems(r.Context(), 0, 0)
	if !canOffice(u) {
		var mine []WorkItem
		for _, it := range items {
			if it.AssigneeID == u.ID {
				mine = append(mine, it)
			}
		}
		items = mine
	}
	projects, _ := s.store.ListProjects(r.Context())
	users, _ := s.store.ListUsers(r.Context())
	for i := range users {
		users[i].HoursPerWeek = s.store.UserPace(r.Context(), users[i].ID)
	}
	var bets []DeliveryBet
	for _, p := range projects {
		eng := EngagementFor(p.Stage)
		open, _ := s.store.CountOpenWork(r.Context(), p.ID)
		slots := eng.MaxOpenTasks - open
		if slots < 0 {
			slots = 0
		}
		bets = append(bets, DeliveryBet{Project: p, Engagement: eng, Open: open, Slots: slots})
	}
	events, _ := s.store.ListProjectEvents(r.Context(), 0, true)
	return pageData{
		Title:        "Delivery",
		Nav:          "delivery",
		User:         u,
		WorkItems:    items,
		Projects:     projects,
		Users:        users,
		DeliveryBets: bets,
		Events:       events,
		CanAsk:       s.workers != nil && s.workers["delivery"] != nil,
		Error:        errMsg,
		Flash:        flash,
	}
}

func (s *Server) deliveryItemPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDelivery(w, r)
	if u == nil {
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	aid, _ := strconv.ParseInt(r.FormValue("assignee_id"), 10, 64)
	p, err := s.store.GetProject(r.Context(), pid)
	if err != nil {
		s.render(w, "delivery", s.deliveryView(r, u, "pick a project", ""))
		return
	}
	eng := EngagementFor(p.Stage)
	open, _ := s.store.CountOpenWork(r.Context(), p.ID)
	if eng.MaxOpenTasks <= 0 {
		s.render(w, "delivery", s.deliveryView(r, u, p.Name+" is "+p.Stage+" — no tasks until you Accept a new stage on Decide", ""))
		return
	}
	status := "open"
	if open >= eng.MaxOpenTasks {
		status = "queued"
	}
	if aid > 0 {
		who, err := s.store.GetUser(r.Context(), aid)
		if err != nil || !canAssign(who, p.ID, p.Members) {
			s.render(w, "delivery", s.deliveryView(r, u, "that person is not seated on this bet (office may take any bet)", ""))
			return
		}
	}
	_, err = s.store.InsertWorkItem(r.Context(), WorkItem{
		ProjectID:  pid,
		AssigneeID: aid,
		Title:      r.FormValue("title"),
		Hours:      parsePace(r.FormValue("hours")),
		Due:        strings.TrimSpace(r.FormValue("due")),
		Kind:       "task",
		Status:     status,
	})
	if err != nil {
		s.render(w, "delivery", s.deliveryView(r, u, err.Error(), ""))
		return
	}
	ok := "saved"
	if status == "queued" {
		ok = "queued"
	}
	http.Redirect(w, r, "/admin/delivery?ok="+ok, http.StatusSeeOther)
}

func (s *Server) deliveryItemStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireDelivery(w, r)
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		it, err := s.store.GetWorkItem(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if !canOffice(u) && it.AssigneeID != u.ID {
			http.Error(w, "not your task", http.StatusForbidden)
			return
		}
		var err2 error
		if status == "open" {
			err2 = s.store.PromoteWorkItem(r.Context(), id)
		} else {
			err2 = s.store.SetWorkItemStatus(r.Context(), id, status)
		}
		if err2 != nil {
			s.render(w, "delivery", s.deliveryView(r, u, err2.Error(), ""))
			return
		}
		ok := "done"
		if status == "skipped" {
			ok = "skipped"
		}
		if status == "open" {
			ok = "saved"
		}
		http.Redirect(w, r, adminNext(r, "/admin/delivery?ok="+ok), http.StatusSeeOther)
	}
}

func (s *Server) deliveryPacePOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requirePeople(w, r)
	if u == nil {
		return
	}
	_ = r.ParseForm()
	id, err := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.SetUserPace(r.Context(), id, parsePace(r.FormValue("hours_per_week"))); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/users/"+strconv.FormatInt(id, 10)+"?ok=pace", http.StatusSeeOther)
}

func (s *Server) deliveryAskPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDelivery(w, r)
	if u == nil {
		return
	}
	_ = r.ParseForm()
	task := strings.TrimSpace(r.FormValue("task"))
	if task == "" {
		s.render(w, "delivery", s.deliveryView(r, u, "Ask needs a sentence", ""))
		return
	}
	fn := s.workers["delivery"]
	if fn == nil {
		s.render(w, "delivery", s.deliveryView(r, u, "Delivery worker is not wired", ""))
		return
	}
	slug := strings.TrimSpace(r.FormValue("project"))
	overlay := s.store.deliveryOverlay(r.Context(), u, slug)
	overlay["channel"] = "desk"
	overlay["department"] = "delivery"
	overlay["action"] = "plan"
	raw, _ := json.Marshal(overlay)
	resp, err := fn(r.Context(), contract.Request{TaskDescription: task, ContextData: raw})
	if err != nil || resp.Status != contract.StatusSuccess {
		msg := resp.OutputText
		if msg == "" && err != nil {
			msg = err.Error()
		}
		s.render(w, "delivery", s.deliveryView(r, u, "Worker: "+msg, ""))
		return
	}
	job, err := s.store.InsertDeskJob(r.Context(), jobFromWorker("delivery", "desk", task, resp.StructuredData, resp.OutputText))
	if err != nil {
		s.render(w, "delivery", s.deliveryView(r, u, err.Error(), ""))
		return
	}
	_, _ = s.store.InsertJobMessage(r.Context(), JobMessage{JobID: job.ID, AuthorKind: "handler", UserID: u.ID, Body: task})
	if resp.OutputText != "" {
		_, _ = s.store.InsertJobMessage(r.Context(), JobMessage{JobID: job.ID, AuthorKind: "worker", Body: resp.OutputText})
	}
	http.Redirect(w, r, "/admin/desks/delivery?job="+strconv.FormatInt(job.ID, 10)+"&ok=asked", http.StatusSeeOther)
}

func (s *Server) internalDeliveryContext(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	slug := strings.TrimSpace(r.URL.Query().Get("project"))
	u, ok := s.deliveryActor(r.Context(), from, r.URL.Query().Get("channel"))
	if !ok {
		writeJSON(w, map[string]any{"found": false})
		return
	}
	if pin := strings.TrimSpace(r.URL.Query().Get("pin")); pin != "" {
		if p, err := s.store.GetProjectBySlug(r.Context(), pin); err == nil && canSeeBet(u, p.Members) {
			_ = s.store.PinProject(r.Context(), u.ID, p.ID)
			slug = p.Slug
		}
	}
	if slug == "" {
		if id := s.store.PinnedProject(r.Context(), u.ID); id > 0 {
			if p, err := s.store.GetProject(r.Context(), id); err == nil {
				slug = p.Slug
			}
		}
	}
	ctx := s.store.deliveryOverlay(r.Context(), &u, slug)
	ctx["found"] = true
	writeJSON(w, ctx)
}

func (s *Server) internalDeliveryDigest(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]any{"people": s.store.MorningDigest(r.Context())})
}

func (s *Server) internalDeliveryCheckin(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		From    string `json:"from"`
		Done    []any  `json:"done_ids"`
		Skipped []any  `json:"skipped_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	u, ok, err := s.store.LookupUserByPhone(r.Context(), body.From)
	if err != nil || !ok {
		u, ok, err = s.store.LookupUserByTelegram(r.Context(), body.From)
	}
	if err != nil || !ok {
		http.Error(w, "unknown staff", http.StatusForbidden)
		return
	}
	n, _ := s.store.ApplyCheckins(r.Context(), u, intIDs(body.Done), "done")
	k, _ := s.store.ApplyCheckins(r.Context(), u, intIDs(body.Skipped), "skipped")
	writeJSON(w, map[string]any{"done": n, "skipped": k})
}

func (s *Server) applyDeliveryResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	sd := resp.StructuredData
	data := contract.ContextObject(req.ContextData)
	from := strVal(data["from"])
	u, ok, _ := s.store.LookupUserByPhone(ctx, from)
	if !ok {
		u, ok, _ = s.store.LookupUserByTelegram(ctx, from)
	}
	if !ok {
		if id := int64Val(data["actor_id"]); id > 0 {
			if got, err := s.store.GetUser(ctx, id); err == nil {
				u, ok = got, true
			}
		}
	}
	if !ok {
		return
	}
	if pin := int64Val(sd["pin_project_id"]); pin > 0 {
		_ = s.store.PinProject(ctx, u.ID, pin)
	}
	if boolVal(sd["apply_checkin"]) {
		_, _ = s.store.ApplyCheckins(ctx, u, intIDs(sd["done_ids"]), "done")
		_, _ = s.store.ApplyCheckins(ctx, u, intIDs(sd["skipped_ids"]), "skipped")
	}
}

func (s *Server) projectWorkPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	aid, _ := strconv.ParseInt(r.FormValue("assignee_id"), 10, 64)
	eng := EngagementFor(p.Stage)
	open, _ := s.store.CountOpenWork(r.Context(), p.ID)
	status := "open"
	if eng.MaxOpenTasks <= 0 || open >= eng.MaxOpenTasks {
		status = "queued"
	}
	_, _ = s.store.InsertWorkItem(r.Context(), WorkItem{
		ProjectID:  p.ID,
		AssigneeID: aid,
		Title:      r.FormValue("title"),
		Hours:      parsePace(r.FormValue("hours")),
		Due:        strings.TrimSpace(r.FormValue("due")),
		Status:     status,
	})
	http.Redirect(w, r, "/admin/projects/"+strconv.FormatInt(p.ID, 10)+"?tab=work", http.StatusSeeOther)
}

type DeliveryBet struct {
	Project
	Engagement StageEngagement
	Open       int
	Slots      int
}
