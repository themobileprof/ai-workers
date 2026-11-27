package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/growth"
)

func liveProposedAds(decks []GrowthDeck) GrowthDeck {
	for _, d := range decks {
		if d.Status == "proposed" && d.Kind == "ads" {
			return d
		}
	}
	return GrowthDeck{}
}

func adsDecks(decks []GrowthDeck) []GrowthDeck {
	var out []GrowthDeck
	for _, d := range decks {
		if d.Kind == "ads" {
			out = append(out, d)
		}
	}
	return out
}

func pitchDecks(decks []GrowthDeck) []GrowthDeck {
	var out []GrowthDeck
	for _, d := range decks {
		if d.Kind != "ads" {
			out = append(out, d)
		}
	}
	return out
}

func (s *Server) projectAds(w http.ResponseWriter, r *http.Request) {
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
	saved, opener, err := s.runAdsAsk(r.Context(), p, deckAskFromForm(r))
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "Growth did not save ads: "+err.Error(), ""))
		return
	}
	s.ensureDeckJob(r.Context(), saved, opener)
	http.Redirect(w, r, withQuery(withQuery(withQuery("/admin/projects/"+strconv.FormatInt(p.ID, 10), "ok", "drafted"), "tab", "ads"), "step", "ads"), http.StatusSeeOther)
}

func (s *Server) runAdsAsk(ctx context.Context, p Project, form deckAskForm) (GrowthDeck, string, error) {
	if !cardHasDeckFacts(p) {
		return GrowthDeck{}, "", errors.New("write the Card one-liner first")
	}
	if strings.TrimSpace(p.URL) == "" {
		return GrowthDeck{}, "", errors.New("write the Card public URL first")
	}
	decks, _ := s.store.ListGrowthDecks(ctx, p.ID)
	if live := liveProposedAds(decks); live.ID > 0 {
		return live, "", nil
	}
	pack := growth.SeedAds(p.URL)
	req, err := adsRequest(p, form.Feedback, pack)
	if err != nil {
		return GrowthDeck{}, "", err
	}
	d := GrowthDeck{
		ProjectID:    p.ID,
		ProjectName:  p.Name,
		Kind:         "ads",
		Audience:     strings.TrimSpace(form.Audience),
		Title:        p.Name + " Search ads",
		Rationale:    strings.TrimSpace(form.Feedback),
		Ads:          pack,
		CurrentSlide: 1,
		Body:         growth.AdsMarkdownFor(pack, p.Name, p.Slug, p.AdsMode),
	}
	opener := ""
	if s.workers != nil {
		if fn := s.workers["growth"]; fn != nil {
			resp, err := fn(ctx, req)
			if err == nil && resp.Status == contract.StatusSuccess {
				opener = strings.TrimSpace(resp.OutputText)
				if ads := growth.ParseAds(resp.StructuredData["ads"]); ads.Current > 0 || ads.FinalURL != "" {
					d.Ads = growth.MergeAds(pack, ads)
					d.CurrentSlide = d.Ads.Current
					d.Body = growth.AdsMarkdownFor(d.Ads, p.Name, p.Slug, p.AdsMode)
				}
				if t := strVal(resp.StructuredData["title"]); t != "" {
					d.Title = t
				}
			}
		}
	}
	saved, err := s.store.InsertGrowthDeck(ctx, d)
	if err != nil {
		return GrowthDeck{}, "", err
	}
	return saved, opener, nil
}

func adsRequest(p Project, feedback string, pack growth.AdsPack) (contract.Request, error) {
	ctx := map[string]any{
		"action":         "ads",
		"channel":        "desk",
		"cannot_send":    true,
		"cannot_spend":   true,
		"human_feedback": strings.TrimSpace(feedback),
		"ads":            pack,
		"current_step":   growth.ClampAdsStep(pack.Current),
		"ads_steps":      growth.AdsSteps(),
	}
	attachProductProject(ctx, p)
	raw, err := json.Marshal(ctx)
	if err != nil {
		return contract.Request{}, err
	}
	task := "Start a Google Search ads pack for " + p.Name + " (" + p.Slug + "). Work step " + strconv.Itoa(growth.ClampAdsStep(pack.Current)) + " first. Use only Card facts. Do not spend. Do not invent a city. Pexels queries are for a human to search."
	if feedback != "" {
		task += " Human notes:\n" + feedback
	}
	return contract.Request{TaskDescription: task, ContextData: raw}, nil
}

func (s *Server) attachAdsSession(ctx context.Context, data map[string]any, d GrowthDeck) {
	p, err := s.store.GetProject(ctx, d.ProjectID)
	if err == nil {
		attachProductProject(data, p)
	}
	data["action"] = "ads"
	data["ref_kind"] = "growth_ads"
	data["ref_id"] = d.ID
	data["growth_deck_id"] = d.ID
	data["ads"] = d.Ads
	data["current_step"] = growth.ClampAdsStep(d.Ads.Current)
	data["ads_steps"] = growth.AdsSteps()
	data["cannot_send"] = true
	data["cannot_spend"] = true
	data["pack_path"] = d.PackPath()
	s.attachAdsGA4Glance(ctx, data, p)
	if len(d.Ads.Looked) > 0 {
		looks, _ := data["looks"].([]string)
		data["looks"] = append(looks, d.Ads.Looked...)
	}
}

func (s *Server) applyAdsResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	if strVal(resp.StructuredData["task_type"]) != "ads" && strVal(resp.StructuredData["action"]) != "ads" {
		return
	}
	data := contract.ContextObject(req.ContextData)
	if strings.EqualFold(strVal(data["channel"]), "whatsapp") {
		return
	}
	var existing GrowthDeck
	id := int64Val(firstNonEmptyAny(data["growth_deck_id"], data["ref_id"], resp.StructuredData["growth_deck_id"]))
	if id > 0 {
		if d, err := s.store.GetGrowthDeck(ctx, id); err == nil && d.Status == "proposed" && d.Kind == "ads" {
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
		if live := liveProposedAds(mustListDecks(s, ctx, p.ID)); live.ID > 0 {
			existing = live
		}
	}
	pack := growth.MergeAds(existing.Ads, growth.ParseAds(resp.StructuredData["ads"]))
	if n := int(int64Val(resp.StructuredData["current_step"])); n > 0 {
		pack.Current = growth.ClampAdsStep(n)
	}
	pack, _ = growth.GuardAds(pack, growth.CardBlob(map[string]any{
		"one_liner": p.OneLiner, "hypothesis": p.Hypothesis, "mission": p.Mission,
		"notes": p.Notes, "funnel_steps": p.FunnelSteps, "url": p.URL, "name": p.Name, "slug": p.Slug,
	}))
	d := GrowthDeck{
		Kind:         "ads",
		ProjectID:    p.ID,
		ProjectName:  p.Name,
		Title:        firstNonEmpty(strVal(resp.StructuredData["title"]), firstNonEmpty(existing.Title, p.Name+" Search ads")),
		Audience:     firstNonEmpty(strVal(resp.StructuredData["audience"]), existing.Audience),
		Ads:          pack,
		CurrentSlide: pack.Current,
		Body:         growth.AdsMarkdownFor(pack, p.Name, p.Slug, p.AdsMode),
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
	resp.StructuredData["current_step"] = saved.Ads.Current
}

func (s *Server) attachLiveAds(ctx context.Context, data *pageData, decks []GrowthDeck) {
	if data == nil {
		return
	}
	live := liveProposedAds(decks)
	if live.ID == 0 {
		return
	}
	ld := live
	data.LiveAds = &ld
	if s == nil || s.store == nil {
		return
	}
	job, err := s.store.FindJobByRef(ctx, "growth", "growth_deck", live.ID)
	if err != nil {
		return
	}
	j := job
	data.AdsJob = &j
	msgs, _ := s.store.ListJobMessages(ctx, job.ID)
	data.AdsMessages = msgs
}

func (s *Server) growthPlace(w http.ResponseWriter, r *http.Request) {
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
	channel := strings.ToLower(strings.TrimSpace(r.FormValue("channel")))
	d, err := s.store.GetGrowthDeck(r.Context(), id)
	if err != nil || !d.IsAds() {
		http.NotFound(w, r)
		return
	}
	if !d.Approved() {
		s.placeAdsFail(w, r, u, "Accept the pack before Place. Chat never Places.")
		return
	}
	if gap := d.Ads.PlaceReady(); gap != "" {
		s.placeAdsFail(w, r, u, "Pack is not ready: "+gap+".")
		return
	}
	if msg := growth.AdsBookConflict(d.Ads, channel, "place"); msg != "" {
		s.placeAdsFail(w, r, u, msg)
		return
	}
	body, err := s.adsPlaceBody(r.Context(), d, channel)
	if err != nil {
		s.placeAdsFail(w, r, u, err.Error())
		return
	}
	if s.n8n == nil {
		s.placeAdsFail(w, r, u, "n8n is not configured. Place talks to n8n, not Google or Meta from this process.")
		return
	}
	out, err := s.n8n.post(r.Context(), n8nAdsPath, body)
	if err != nil {
		s.placeAdsFail(w, r, u, "Place failed: "+err.Error())
		return
	}
	audit := firstNonEmpty(extractField(out, "audit_text", "text"), "n8n placed the campaign.")
	placedID := firstNonEmpty(extractField(out, "id", "campaign_id", "resource_name"), "")
	if boolVal(outMap(out)["skip"]) || strings.EqualFold(extractField(out, "status"), "error") {
		s.placeAdsFail(w, r, u, audit)
		return
	}
	d.Ads = d.Ads.WithPlacement(growth.AdsPlacement{
		Channel: channel,
		ID:      placedID,
		Mode:    "place",
		Audit:   audit,
		At:      time.Now().UTC().Format(time.RFC3339),
	})
	if _, err := s.store.RecordAdsPlacement(r.Context(), d); err != nil {
		s.placeAdsFail(w, r, u, "Placed, but the pack did not save: "+err.Error())
		return
	}
	http.Redirect(w, r, growthNext(r, "/admin/growth?ok=placed"), http.StatusSeeOther)
}

func (s *Server) growthBookManual(w http.ResponseWriter, r *http.Request) {
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
	channel := strings.ToLower(strings.TrimSpace(r.FormValue("channel")))
	d, err := s.store.GetGrowthDeck(r.Context(), id)
	if err != nil || !d.IsAds() {
		http.NotFound(w, r)
		return
	}
	if !d.Approved() {
		s.placeAdsFail(w, r, u, "Accept the pack before you book it in the Ads UI. Chat never spends.")
		return
	}
	if gap := d.Ads.PlaceReady(); gap != "" {
		s.placeAdsFail(w, r, u, "Pack is not ready: "+gap+".")
		return
	}
	if msg := growth.AdsBookConflict(d.Ads, channel, "manual"); msg != "" {
		s.placeAdsFail(w, r, u, msg)
		return
	}
	if channel != "google" && channel != "meta" {
		s.placeAdsFail(w, r, u, "Pick Google Search or Meta.")
		return
	}
	campaignID := strings.TrimSpace(r.FormValue("campaign_id"))
	if campaignID == "" {
		campaignID = "manual"
	}
	audit := "Handler booked this in the Ads UI from the pack instructions. Do not Place the same cells."
	if campaignID != "manual" {
		audit += " Campaign id " + campaignID + "."
	} else {
		audit += " Paste the campaign id when you have it so prune can score this placement."
	}
	d.Ads = d.Ads.WithPlacement(growth.AdsPlacement{
		Channel: channel,
		ID:      campaignID,
		Mode:    "manual",
		Audit:   audit,
		At:      time.Now().UTC().Format(time.RFC3339),
	})
	if _, err := s.store.RecordAdsPlacement(r.Context(), d); err != nil {
		s.placeAdsFail(w, r, u, "Booked, but the pack did not save: "+err.Error())
		return
	}
	http.Redirect(w, r, growthNext(r, "/admin/growth?ok=booked"), http.StatusSeeOther)
}

func (s *Server) placeAdsFail(w http.ResponseWriter, r *http.Request, u *User, msg string) {
	decks, _ := s.store.ListGrowthDecks(r.Context(), 0)
	projects, _ := s.store.ListProjects(r.Context())
	data := pageData{
		Title:     "Growth",
		Nav:       "growth",
		User:      u,
		Decks:     decks,
		Projects:  projects,
		DeckKinds: growth.Specs(),
		CanGrowth: s.growthReady(),
		Error:     msg,
	}
	s.attachAdsGA4(r.Context(), &data)
	s.render(w, "growth", data)
}

func (s *Server) adsPlaceBody(ctx context.Context, d GrowthDeck, channel string) (map[string]any, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel != "google" && channel != "meta" {
		return nil, errors.New("pick Google Search or Meta")
	}
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return nil, err
	}
	cap := strings.TrimSpace(settings["ads.daily_cap"])
	switch channel {
	case "google":
		if strings.TrimSpace(settings["ads.google_customer_id"]) == "" {
			return nil, errors.New("paste the Google Ads customer id on Defaults → Ads")
		}
	case "meta":
		if strings.TrimSpace(settings["ads.meta_ad_account_id"]) == "" || strings.TrimSpace(settings["ads.meta_page_id"]) == "" {
			return nil, errors.New("paste the Meta ad account id and Page id on Defaults → Ads")
		}
	}
	p, err := s.store.GetProject(ctx, d.ProjectID)
	if err != nil {
		return nil, err
	}
	status := strings.ToUpper(strings.TrimSpace(settings["ads.campaign_status"]))
	if status != "ENABLED" {
		status = "PAUSED"
	}
	pack := growth.ClipAds(d.Ads)
	if d.AdsMode != "" {
		p.AdsMode = d.AdsMode
	}
	if d.AdsSegments != "" && p.AdsSegments == "" {
		p.AdsSegments = d.AdsSegments
	}
	mode := growth.NormalizeAdsMode(p.AdsMode)
	campaign := growth.AdsCampaignName(p.Slug, p.Name, mode)
	filtered, err := growth.FilterAdsCells(pack.Cells, mode, p.AdsSegments)
	if err != nil {
		return nil, err
	}
	heads, descs := []string{}, []string{}
	for _, h := range pack.Headlines {
		heads = append(heads, h.Text)
	}
	for _, x := range pack.Descriptions {
		descs = append(descs, x.Text)
	}
	cells := adsPlaceCells(channel, campaign, pack, filtered)
	return map[string]any{
		"action":                   "place",
		"channel":                  channel,
		"slug":                     p.Slug,
		"name":                     p.Name,
		"campaign":                 campaign,
		"ads_mode":                 mode,
		"reserve_div":              growth.AdsReserveDivisor(mode),
		"final_url":                pack.FinalURL,
		"headlines":                heads,
		"descriptions":             descs,
		"keywords":                 pack.Keywords,
		"cells":                    cells,
		"daily_cap":                cap,
		"campaign_status":          status,
		"google_customer_id":       strings.TrimSpace(settings["ads.google_customer_id"]),
		"google_login_customer_id": strings.TrimSpace(settings["ads.google_login_customer_id"]),
		"meta_ad_account_id":       strings.TrimSpace(settings["ads.meta_ad_account_id"]),
		"meta_page_id":             strings.TrimSpace(settings["ads.meta_page_id"]),
	}, nil
}

func adsPlaceCells(channel, campaign string, pack growth.AdsPack, src []growth.AdsCell) []map[string]any {
	if len(src) == 0 {
		src = []growth.AdsCell{{Geo: "Nigeria", Theme: "search"}}
	}
	out := make([]map[string]any, 0, len(src))
	for _, c := range src {
		out = append(out, map[string]any{
			"geo":   c.Geo,
			"theme": c.Theme,
			"why":   c.Why,
			"url":   growth.AdsTrackURL(pack.FinalURL, channel, campaign, growth.AdsCellContent(c)),
		})
	}
	return out
}

func outMap(v any) map[string]any {
	v = unwrapN8n(v)
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func (s *Server) internalAdsPlacementsGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	settings, err := s.store.Settings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	rows, err := s.store.ListAdsPlacements(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if rows == nil {
		rows = []AdsPlacementRow{}
	}
	writeJSON(w, map[string]any{
		"google_customer_id":       strings.TrimSpace(settings["ads.google_customer_id"]),
		"google_login_customer_id": strings.TrimSpace(settings["ads.google_login_customer_id"]),
		"meta_ad_account_id":       strings.TrimSpace(settings["ads.meta_ad_account_id"]),
		"placements":               rows,
	})
}

func (s *Server) internalAdsDecidePOST(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var in growth.AdsDecideIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	writeJSON(w, growth.AdsDecide(in))
}

func (s *Server) internalAdsPrunedPOST(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Channel string   `json:"channel"`
		ID      string   `json:"id"`
		Note    string   `json:"note"`
		Dropped []string `json:"dropped"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := s.store.RecordAdsPrune(r.Context(), body.Channel, body.ID, body.Note, body.Dropped); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) recordAdsTurnAway(ctx context.Context, id int64, reason string) {
	if s == nil || s.store == nil || id <= 0 {
		return
	}
	reason = clipRunes(strings.TrimSpace(reason), 200)
	if reason == "" {
		return
	}
	d, err := s.store.GetGrowthDeck(ctx, id)
	if err != nil || d.Kind != "ads" {
		return
	}
	d.Ads.Looked = append(d.Ads.Looked, "handler turned away: "+reason)
	d.Ads = growth.ClipAds(d.Ads)
	if _, err := s.store.RecordAdsPlacement(ctx, d); err != nil {
		return
	}
}
