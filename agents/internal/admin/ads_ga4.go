package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/departments/growth"
)

const adsGA4PaidCap = 30
const adsGA4FunnelCap = 15

type AdsGA4PaidRow struct {
	SourceMedium string  `json:"source_medium"`
	Campaign     string  `json:"campaign"`
	Content      string  `json:"content"`
	Sessions     float64 `json:"sessions"`
	Engaged      float64 `json:"engaged"`
	BounceRate   float64 `json:"bounce_rate"`
	KeyEvents    float64 `json:"key_events"`
}

type AdsGA4FunnelRow struct {
	Name     string  `json:"name"`
	Users    float64 `json:"users"`
	Keep     float64 `json:"keep"`
	ShowKeep bool    `json:"show_keep,omitempty"`
}

func (r AdsGA4FunnelRow) KeepLabel() string {
	return fmt.Sprintf("%.0f%%", r.Keep*100)
}

func (r AdsGA4PaidRow) BounceLabel() string {
	return fmt.Sprintf("%.0f%%", r.BounceRate*100)
}

type AdsGA4FunnelCampaign struct {
	Campaign string            `json:"campaign"`
	Steps    []AdsGA4FunnelRow `json:"steps"`
}

type AdsGA4View struct {
	Found      bool                   `json:"found"`
	PropertyID string                 `json:"property_id"`
	Range      string                 `json:"range"`
	FetchedAt  string                 `json:"fetched_at"`
	Note       string                 `json:"note"`
	Steps      []growth.AdsFunnelStep `json:"steps"`
	Paid       []AdsGA4PaidRow        `json:"paid"`
	Funnel     []AdsGA4FunnelRow      `json:"funnel"`
	ByCampaign []AdsGA4FunnelCampaign `json:"by_campaign"`
}

func (v AdsGA4View) ForSlug(slug string) AdsGA4View {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" || !v.Found {
		return v
	}
	var paid []AdsGA4PaidRow
	for _, r := range v.Paid {
		if strings.Contains(strings.ToLower(r.Campaign), slug) || strings.Contains(strings.ToLower(r.Content), slug) {
			paid = append(paid, r)
		}
	}
	var by []AdsGA4FunnelCampaign
	for _, c := range v.ByCampaign {
		if strings.Contains(strings.ToLower(c.Campaign), slug) {
			by = append(by, c)
		}
	}
	out := v
	out.Paid = paid
	out.ByCampaign = by
	if len(paid) == 0 && len(by) == 0 {
		out.Note = strings.TrimSpace(out.Note + " No paid sessions tagged with this bet slug yet. Discovery Place stamps utm_campaign to `{slug}-discovery`; mature Place uses `{slug}`.")
	}
	return out
}

func (s *Store) migrateAdsGA4(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS ads_ga4 (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  property_id TEXT NOT NULL DEFAULT '',
  snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}

func (s *Store) UpsertAdsGA4(ctx context.Context, snap AdsGA4View) error {
	if s == nil || s.pool == nil {
		return errors.New("store")
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	fetched := time.Now().UTC()
	if ts := strings.TrimSpace(snap.FetchedAt); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			fetched = t
		}
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO ads_ga4 (id, property_id, snapshot, fetched_at)
VALUES (1, $1, $2::jsonb, $3)
ON CONFLICT (id) DO UPDATE SET property_id = EXCLUDED.property_id, snapshot = EXCLUDED.snapshot, fetched_at = EXCLUDED.fetched_at
`, strings.TrimSpace(snap.PropertyID), raw, fetched)
	return err
}

func (s *Store) GetAdsGA4(ctx context.Context) (AdsGA4View, error) {
	if s == nil || s.pool == nil {
		return AdsGA4View{}, nil
	}
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT snapshot FROM ads_ga4 WHERE id = 1`).Scan(&raw)
	if err != nil {
		return AdsGA4View{}, nil
	}
	var snap AdsGA4View
	if err := json.Unmarshal(raw, &snap); err != nil {
		return AdsGA4View{}, nil
	}
	return snap, nil
}

func (s *Server) attachAdsGA4(ctx context.Context, data *pageData) {
	if s == nil || s.store == nil || data == nil {
		return
	}
	snap, err := s.store.GetAdsGA4(ctx)
	if err != nil || !snap.Found {
		return
	}
	if data.Project != nil {
		snap = snap.ForSlug(data.Project.Slug)
		if line := strings.TrimSpace(data.Project.FunnelSteps); line != "" {
			snap.Note = strings.TrimSpace("Card funnel: " + line + ". " + snap.Note)
		}
	}
	data.AdsGA4 = &snap
}

func parseAdsGA4Snapshot(body map[string]any, steps []growth.AdsFunnelStep) (AdsGA4View, error) {
	if body == nil {
		return AdsGA4View{}, errors.New("empty GA4 body")
	}
	if _, bad := body["error"]; bad {
		return AdsGA4View{}, errors.New("GA4 error payload")
	}
	if strings.TrimSpace(strVal(body["source"])) != "ga4_data_api" {
		return AdsGA4View{}, errors.New("source must be ga4_data_api")
	}
	pid := ga4PropertyID(strVal(body["property_id"]))
	if pid == "" {
		return AdsGA4View{}, errors.New("property_id required")
	}
	if len(steps) < 2 {
		steps = growth.DefaultAdsFunnelSteps()
	}
	out := AdsGA4View{
		Found:      true,
		PropertyID: pid,
		Range:      firstNonEmpty(strVal(body["range"]), "7daysAgo:yesterday"),
		Steps:      steps,
		Paid:       parseAdsGA4Paid(body["paid"]),
		Note:       strings.TrimSpace(strVal(body["note"])),
	}
	funnelRaw, _ := body["funnel"].(map[string]any)
	out.Funnel, out.ByCampaign = parseAdsGA4Funnel(funnelRaw, steps)
	if ts := strings.TrimSpace(strVal(body["fetched_at"])); ts != "" {
		if _, err := time.Parse(time.RFC3339, ts); err == nil {
			out.FetchedAt = ts
		}
	}
	if out.FetchedAt == "" {
		out.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if len(out.Funnel) == 0 && funnelHasError(funnelRaw) {
		out.Note = strings.TrimSpace(out.Note + " Funnel report did not return. Paid sessions still land. Mark key events in GA4 if you need a third step.")
	}
	return out, nil
}

func funnelHasError(raw map[string]any) bool {
	if raw == nil {
		return true
	}
	if _, ok := raw["error"]; ok {
		return true
	}
	return false
}

func parseAdsGA4Paid(v any) []AdsGA4PaidRow {
	m, _ := v.(map[string]any)
	if m == nil {
		return nil
	}
	rows := ga4TableRows(m)
	out := make([]AdsGA4PaidRow, 0, len(rows))
	for _, r := range rows {
		camp := strings.TrimSpace(r.dim["sessionCampaignName"])
		if camp == "" {
			camp = "(not set)"
		}
		out = append(out, AdsGA4PaidRow{
			SourceMedium: firstNonEmpty(r.dim["sessionSourceMedium"], "(not set)"),
			Campaign:     camp,
			Content:      r.dim["sessionManualAdContent"],
			Sessions:     r.metric["sessions"],
			Engaged:      r.metric["engagedSessions"],
			BounceRate:   r.metric["bounceRate"],
			KeyEvents:    r.metric["keyEvents"],
		})
		if len(out) >= adsGA4PaidCap {
			break
		}
	}
	return out
}

func parseAdsGA4Funnel(raw map[string]any, steps []growth.AdsFunnelStep) ([]AdsGA4FunnelRow, []AdsGA4FunnelCampaign) {
	if raw == nil {
		return nil, nil
	}
	table, _ := raw["funnelTable"].(map[string]any)
	if table == nil {
		table = raw
	}
	rows := ga4TableRows(table)
	if len(rows) == 0 {
		return nil, nil
	}
	totals := map[string]float64{}
	by := map[string]map[string]float64{}
	for _, r := range rows {
		step := firstNonEmpty(firstNonEmpty(r.dim["funnelStepName"], r.dim["funnelStep"]), r.dim["step"])
		camp := strings.TrimSpace(firstNonEmpty(r.dim["sessionCampaignName"], r.dim["campaign"]))
		users := r.metric["activeUsers"]
		if users == 0 {
			users = r.metric["completions"]
		}
		if users == 0 {
			for _, v := range r.metric {
				if v > users {
					users = v
				}
			}
		}
		if step == "" {
			continue
		}
		totals[step] += users
		if camp == "" {
			camp = "(not set)"
		}
		if by[camp] == nil {
			by[camp] = map[string]float64{}
		}
		by[camp][step] += users
	}
	overall := funnelRows(steps, totals)
	var camps []AdsGA4FunnelCampaign
	for camp, m := range by {
		camps = append(camps, AdsGA4FunnelCampaign{Campaign: camp, Steps: funnelRows(steps, m)})
	}
	sort.Slice(camps, func(i, j int) bool { return camps[i].Campaign < camps[j].Campaign })
	if len(camps) > adsGA4FunnelCap {
		camps = camps[:adsGA4FunnelCap]
	}
	return overall, camps
}

func funnelRows(steps []growth.AdsFunnelStep, users map[string]float64) []AdsGA4FunnelRow {
	var out []AdsGA4FunnelRow
	var prev float64
	for i, s := range steps {
		n := users[s.Name]
		row := AdsGA4FunnelRow{Name: s.Name, Users: n, Keep: 1, ShowKeep: i > 0}
		if i > 0 && prev > 0 {
			row.Keep = n / prev
		}
		if i > 0 && prev <= 0 {
			row.Keep = 0
		}
		out = append(out, row)
		prev = n
	}
	return out
}

type ga4Row struct {
	dim    map[string]string
	metric map[string]float64
}

func ga4TableRows(raw map[string]any) []ga4Row {
	if raw == nil {
		return nil
	}
	dimH := headerNames(raw["dimensionHeaders"])
	metH := headerNames(raw["metricHeaders"])
	list, _ := raw["rows"].([]any)
	out := make([]ga4Row, 0, len(list))
	for _, row := range list {
		m, _ := row.(map[string]any)
		if m == nil {
			continue
		}
		item := ga4Row{dim: map[string]string{}, metric: map[string]float64{}}
		for i, dv := range anyList(m["dimensionValues"]) {
			name := ""
			if i < len(dimH) {
				name = dimH[i]
			}
			vm, _ := dv.(map[string]any)
			if name != "" {
				item.dim[name] = strVal(vm["value"])
			}
		}
		for i, mv := range anyList(m["metricValues"]) {
			name := ""
			if i < len(metH) {
				name = metH[i]
			}
			vm, _ := mv.(map[string]any)
			if name != "" {
				item.metric[name] = jsonFloat(vm["value"])
			}
		}
		out = append(out, item)
	}
	return out
}

func headerNames(v any) []string {
	var out []string
	for _, h := range anyList(v) {
		m, _ := h.(map[string]any)
		name := strVal(m["name"])
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func anyList(v any) []any {
	list, _ := v.([]any)
	return list
}

func (s *Server) internalAdsGA4QueryGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.store == nil {
		http.Error(w, "store", http.StatusBadGateway)
		return
	}
	st, err := s.store.Settings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	pid := ga4PropertyID(st["seo.ga4_property_id"])
	if pid == "" {
		writeJSON(w, map[string]any{"skip": true, "reason": "seo.ga4_property_id is empty on Defaults → Traffic"})
		return
	}
	steps := growth.ParseAdsFunnelSteps(st["ads.funnel_steps"])
	base := "https://analyticsdata.googleapis.com/v1beta/properties/" + pid
	writeJSON(w, map[string]any{
		"skip":            false,
		"property_id":     pid,
		"range":           "7daysAgo:yesterday",
		"steps":           steps,
		"project_funnels": s.adsProjectFunnels(r.Context()),
		"paid_url":        base + ":runReport",
		"paid_body":       growth.AdsPaidReportBody(),
		"funnel_url":      base + ":runFunnelReport",
		"funnel_body":     growth.AdsFunnelReportBody(steps),
	})
}

func (s *Server) adsProjectFunnels(ctx context.Context) []map[string]any {
	if s == nil || s.store == nil {
		return nil
	}
	list, err := s.store.ListProjects(ctx)
	if err != nil {
		return nil
	}
	var out []map[string]any
	for _, p := range list {
		if !growth.AdsFunnelLineOK(p.FunnelSteps) {
			continue
		}
		out = append(out, map[string]any{
			"slug":  p.Slug,
			"raw":   strings.TrimSpace(p.FunnelSteps),
			"steps": growth.ParseAdsFunnelSteps(p.FunnelSteps),
		})
	}
	return out
}

func (s *Server) internalAdsGA4POST(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.store == nil {
		http.Error(w, "store", http.StatusBadGateway)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	st, err := s.store.Settings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	want := ga4PropertyID(st["seo.ga4_property_id"])
	if want == "" {
		http.Error(w, "seo.ga4_property_id is empty on Defaults", http.StatusBadRequest)
		return
	}
	steps := growth.ParseAdsFunnelSteps(st["ads.funnel_steps"])
	snap, err := parseAdsGA4Snapshot(body, steps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if snap.PropertyID != want {
		http.Error(w, "property_id does not match Defaults", http.StatusBadRequest)
		return
	}
	if err := s.store.UpsertAdsGA4(r.Context(), snap); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.fileAdsGlanceJob(r.Context(), snap)
	s.filePaidSenseJobs(r.Context(), snap)
	writeJSON(w, map[string]any{"ok": true, "ga4": snap})
}

func (s *Server) attachAdsGA4Glance(ctx context.Context, data map[string]any, p Project) {
	if s == nil || s.store == nil || data == nil {
		return
	}
	looks := []string{"card"}
	snap, err := s.store.GetAdsGA4(ctx)
	if err == nil && snap.Found {
		if strings.TrimSpace(p.Slug) != "" {
			snap = snap.ForSlug(p.Slug)
		}
		data["ads_ga4"] = compactAdsGA4(snap)
		looks = append(looks, "ga4")
	}
	data["looks"] = looks
	data["cannot_spend"] = true
	data["cannot_send"] = true
}

func compactAdsGA4(v AdsGA4View) map[string]any {
	paid := make([]any, 0, len(v.Paid))
	for i, r := range v.Paid {
		if i == 8 {
			break
		}
		paid = append(paid, map[string]any{
			"campaign": r.Campaign, "content": r.Content,
			"sessions": r.Sessions, "engaged": r.Engaged,
		})
	}
	funnel := make([]any, 0, len(v.Funnel))
	for _, r := range v.Funnel {
		funnel = append(funnel, map[string]any{"name": r.Name, "users": r.Users})
	}
	return map[string]any{
		"found":  v.Found,
		"note":   v.Note,
		"range":  v.Range,
		"paid":   paid,
		"funnel": funnel,
	}
}

func adsGlanceHasPaid(snap AdsGA4View) bool {
	for _, r := range snap.Paid {
		if r.Sessions > 0 {
			return true
		}
	}
	return false
}

func glanceDayID(now time.Time) int64 {
	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		loc = time.FixedZone("WAT", 3600)
	}
	t := now.In(loc)
	return int64(t.Year()*10000 + int(t.Month())*100 + t.Day())
}

func (s *Server) fileAdsGlanceJob(ctx context.Context, snap AdsGA4View) {
	if s == nil || s.store == nil || !snap.Found || !adsGlanceHasPaid(snap) {
		return
	}
	body := strings.TrimSpace(growth.FormatAdsGlance(compactAdsGA4(snap)))
	if body == "" {
		return
	}
	sum := body
	if len(sum) > 800 {
		sum = sum[:797] + "…"
	}
	day := glanceDayID(time.Now())
	if existing, err := s.store.FindJobByRef(ctx, "growth", "ads_ga4_glance", day); err == nil && existing.ID > 0 {
		if strings.TrimSpace(existing.Summary) == sum {
			return
		}
		_ = s.store.UpdateDeskJobSummary(ctx, existing.ID, sum)
		_, _ = s.store.InsertJobMessage(ctx, JobMessage{JobID: existing.ID, AuthorKind: "worker", Body: body})
		return
	}
	job, err := s.store.InsertDeskJob(ctx, DeskJob{
		Desk:    "growth",
		Title:   "Paid traffic glance",
		Status:  "proposed",
		Source:  "ga4",
		Summary: sum,
		RefKind: "ads_ga4_glance",
		RefID:   day,
		Payload: map[string]any{"task_type": "ads_glance", "cannot_spend": true, "cannot_send": true, "save_deck": false},
	})
	if err != nil {
		return
	}
	_, _ = s.store.InsertJobMessage(ctx, JobMessage{JobID: job.ID, AuthorKind: "worker", Body: body})
}

func (s *Server) filePaidSenseJobs(ctx context.Context, snap AdsGA4View) {
	if s == nil || s.store == nil || !snap.Found || !adsGlanceHasPaid(snap) {
		return
	}
	projects, err := s.store.ListProjects(ctx)
	if err != nil {
		return
	}
	for _, p := range projects {
		if strings.TrimSpace(p.Slug) == "" {
			continue
		}
		part := snap.ForSlug(p.Slug)
		if !adsGlanceHasPaid(part) {
			continue
		}
		s.filePaidSenseJob(ctx, p, part)
	}
}

func (s *Server) filePaidSenseJob(ctx context.Context, p Project, snap AdsGA4View) {
	evidence, segments, body := formatPaidSense(p, snap)
	if body == "" {
		return
	}
	sum := body
	if len(sum) > 800 {
		sum = sum[:797] + "…"
	}
	payload := map[string]any{
		"task_type":          "product_validation",
		"action":             "analyse_paid_cells",
		"save_memory":        true,
		"needs_human":        true,
		"cannot_spend":       true,
		"cannot_send":        true,
		"cannot_write_stamp": true,
		"project_id":         p.ID,
		"project_slug":       p.Slug,
		"project_name":       p.Name,
		"card_evidence":      evidence,
		"ads_segments":       segments,
		"decision":           "CONTINUE_TESTING",
		"title":              "Paid cells · " + p.Name,
	}
	if existing, err := s.store.FindJobByRef(ctx, "product-dev", "ads_paid_sense", p.ID); err == nil && existing.ID > 0 {
		if strings.TrimSpace(existing.Summary) == sum {
			return
		}
		_ = s.store.UpdateDeskJobContent(ctx, existing.ID, sum, payload)
		_, _ = s.store.InsertJobMessage(ctx, JobMessage{JobID: existing.ID, AuthorKind: "worker", Body: body})
		return
	}
	job, err := s.store.InsertDeskJob(ctx, DeskJob{
		Desk:    "product-dev",
		Title:   "Paid cells · " + p.Name,
		Status:  "proposed",
		Source:  "ga4",
		Summary: sum,
		RefKind: "ads_paid_sense",
		RefID:   p.ID,
		Payload: payload,
	})
	if err != nil {
		return
	}
	_, _ = s.store.InsertJobMessage(ctx, JobMessage{JobID: job.ID, AuthorKind: "worker", Body: body})
}

func winningPaidCells(rows []AdsGA4PaidRow, n int) []AdsGA4PaidRow {
	var hits []AdsGA4PaidRow
	for _, r := range rows {
		if r.Sessions <= 0 {
			continue
		}
		hits = append(hits, r)
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Sessions != hits[j].Sessions {
			return hits[i].Sessions > hits[j].Sessions
		}
		return hits[i].Engaged > hits[j].Engaged
	})
	seen := map[string]bool{}
	var out []AdsGA4PaidRow
	for _, r := range hits {
		key := strings.ToLower(strings.TrimSpace(r.Content))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(r.Campaign))
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
		if n > 0 && len(out) >= n {
			break
		}
	}
	return out
}

func formatPaidSense(p Project, snap AdsGA4View) (evidence, segments, body string) {
	winners := winningPaidCells(snap.Paid, 3)
	if len(winners) == 0 {
		return "", "", ""
	}
	var segLines []string
	var evParts []string
	for _, r := range winners {
		label := strings.TrimSpace(r.Content)
		if label == "" {
			label = strings.TrimSpace(r.Campaign)
		}
		if label == "" {
			continue
		}
		segLines = append(segLines, label)
		evParts = append(evParts, fmt.Sprintf("%s (%.0f sessions, %.0f engaged)", label, r.Sessions, r.Engaged))
	}
	if len(segLines) == 0 {
		return "", "", ""
	}
	segments = strings.Join(segLines, "\n")
	evidence = "Paid Search cells for " + firstNonEmpty(p.Name, p.Slug) + " last 7 days: " + strings.Join(evParts, "; ") + ". These are geo plus keyword-theme segments, not age or gender. Accept writes them as proposed winning segments. You still mark the Card mature before substantial Place. This job does not Place and does not kill the bet."
	var b strings.Builder
	b.WriteString("Paid Search cells for " + firstNonEmpty(p.Name, p.Slug) + " (last 7 days). This is evidence, not Place. Chat cannot spend and cannot mark the Card mature.\n")
	if n := strings.TrimSpace(snap.Note); n != "" {
		b.WriteString(n + "\n")
	}
	b.WriteString("\nWinning cells (geo plus keyword theme):\n")
	for _, r := range winners {
		cell := strings.TrimSpace(r.Content)
		if cell == "" {
			cell = "(no cell)"
		}
		camp := strings.TrimSpace(r.Campaign)
		if camp == "" {
			camp = "(no campaign)"
		}
		b.WriteString(fmt.Sprintf("- %s / %s · %.0f sessions, %.0f engaged\n", camp, cell, r.Sessions, r.Engaged))
	}
	b.WriteString("\nAccept writes the evidence and these segments onto the Card. Discovery spend stays on `" + growth.AdsCampaignName(p.Slug, p.Name, growth.AdsModeDiscovery) + "`. Substantial spend waits until you set Ads spend to Mature on the Card. Sessions are not payment. Do not KILL from this snapshot.")
	return evidence, segments, b.String()
}

func (s *Server) internalAdsGA4GET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.store == nil {
		http.Error(w, "store", http.StatusBadGateway)
		return
	}
	snap, err := s.store.GetAdsGA4(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if !snap.Found {
		writeJSON(w, map[string]any{"found": false})
		return
	}
	writeJSON(w, map[string]any{"found": true, "ga4": snap})
}
