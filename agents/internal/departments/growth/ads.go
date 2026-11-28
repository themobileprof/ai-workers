package growth

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const adsHeadlineLimit = 30
const adsDescLimit = 90
const adsHeadlineCap = 15
const adsDescCap = 4

const adsOverlay = `You are the Growth clerk for TheMobileProf Technologies (Lagos), the only CAC-registered company in this office.
You draft a Google Search text-ad pack (Responsive Search Ads) from the Card. You cannot spend, publish, or log into Google Ads or Meta.
A human Accepts on the desk, then either follows the pack instructions in the Ads UI or clicks Place. n8n holds the tokens. Chat never Accepts.
Do not invent waitlist, MAU, GMV, CPC, or a city that is not already on the Card or the public URL.
Keywords and claims must already appear on the Card one-liner, hypothesis, mission, notes, funnel_steps, or project.url host/path words. If Ikeja is not there, do not put Ikeja in a keyword or Pexels query. Funnel events come from Card funnel_steps (TRAFFIC.md) — do not invent generate_lead if it is not there.
context_data.looks and context_data.ads_ga4 are already fetched this turn. Quote only those numbers. Do not invent CPC or sessions. Go will drop geo/keywords that are not on the Card.
If context_data.lessons.turned_away is set, do not repeat those mistakes. If lessons.accepted is set, stay consistent with those packs. Do not invent facts from them.
Search tests are geo (named place on the Card) plus a query theme (words on the Card). Do not propose age or gender cells — Search will not isolate those.
Discovery Place is a cheap probe on all cells (campaign {slug}-discovery, remaining/30 — about one month of daily budget, enough for four weekly iterations). Substantial spend is a different campaign ({slug}, remaining/30) after a human marks the Card mature and lists winning geo plus keyword-theme segments. Nigerian Google Ads accounts are usually NGN. You cannot mark mature, Place, or spend.
Pexels: suggest 3–5 search phrases a human will type on pexels.com. Objects and places only. Never a generated face. Never "happy African woman laptop". Include one avoid line.
Work ONE step at a time. context_data.ads is the live pack. context_data.current_step is 1–6.
1. Final URL — Card public URL only. Else [TO BE COMPLETED].
2. Test cells — 3–5 {geo, theme, why}. Named places only.
3. Keywords — phrases on the Card/page. No competitor names unless already on the Card.
4. Headlines — each ≤30 characters, up to 15. Advertiser is TheMobileProf Technologies.
5. Descriptions — each ≤90 characters, up to 4.
6. Pexels queries — {query, avoid}. Human searches and /file under product. You do not fetch images.
cannot_send is always true. cannot_spend is always true. Customer WhatsApp is sales, not this session.
Nigerian/West African commercial English. Phone-short. No markdown tables.
task_type must be "ads".`

type AdsStep struct {
	N     int    `json:"n"`
	Title string `json:"title"`
	Card  string `json:"card"`
}

func AdsSteps() []AdsStep {
	return []AdsStep{
		{1, "Final URL", "Card public URL only. Do not invent a landing page."},
		{2, "Test cells", "3–5 Search tests: geo (named place on the Card) + query theme. Not age or gender."},
		{3, "Keywords", "Phrases already on the Card or landing page. No competitor scrape."},
		{4, "Headlines", "Google Search RSA. Each ≤30 characters. Up to 15."},
		{5, "Descriptions", "Google Search RSA. Each ≤90 characters. Up to 4."},
		{6, "Pexels queries", "3–5 phrases a human types on Pexels. Objects and places. Not generated faces."},
	}
}

// AdsWalkStep is one of the six Search-ads steps as shown on the desk.
type AdsWalkStep struct {
	N      int    `json:"n"`
	Title  string `json:"title"`
	Card   string `json:"card"`
	Status string `json:"status"`
	Body   string `json:"body"`
}

func (p AdsPack) Walk() []AdsWalkStep {
	p = ClipAds(p)
	cur := ClampAdsStep(p.Current)
	var out []AdsWalkStep
	for _, s := range AdsSteps() {
		item := AdsWalkStep{N: s.N, Title: s.Title, Card: s.Card, Body: p.stepPreview(s.N), Status: "draft"}
		if strings.TrimSpace(item.Body) != "" && s.N != cur {
			item.Status = "agreed"
		}
		out = append(out, item)
	}
	return out
}

func (p AdsPack) stepPreview(n int) string {
	switch n {
	case 1:
		return strings.TrimSpace(p.FinalURL)
	case 2:
		var lines []string
		for _, c := range p.Cells {
			line := strings.TrimSpace(c.Geo)
			if c.Theme != "" {
				if line != "" {
					line += " · "
				}
				line += c.Theme
			}
			if line != "" {
				lines = append(lines, line)
			}
		}
		return strings.Join(lines, "\n")
	case 3:
		return strings.Join(p.Keywords, "\n")
	case 4:
		var lines []string
		for _, h := range p.Headlines {
			if t := strings.TrimSpace(h.Text); t != "" {
				lines = append(lines, t)
			}
		}
		return strings.Join(lines, "\n")
	case 5:
		var lines []string
		for _, d := range p.Descriptions {
			if t := strings.TrimSpace(d.Text); t != "" {
				lines = append(lines, t)
			}
		}
		return strings.Join(lines, "\n")
	case 6:
		var lines []string
		for _, q := range p.Pexels {
			if t := strings.TrimSpace(q.Query); t != "" {
				lines = append(lines, t)
			}
		}
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

func (s AdsWalkStep) StampClass() string {
	if s.Status == "agreed" {
		return "ok"
	}
	return "brass"
}

func (s AdsWalkStep) StampLabel() string {
	if s.Status == "agreed" {
		return "agreed"
	}
	return "draft"
}

type AdsCell struct {
	Geo    string `json:"geo"`
	Theme  string `json:"theme"`
	Why    string `json:"why"`
	Status string `json:"status"`
}

type AdsLine struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

type PexelsQuery struct {
	Query string `json:"query"`
	Avoid string `json:"avoid"`
}

func (q PexelsQuery) SearchURL() string {
	qstr := strings.TrimSpace(q.Query)
	if qstr == "" {
		return "https://www.pexels.com/"
	}
	return "https://www.pexels.com/search/" + url.PathEscape(qstr)
}

type AdsPlacement struct {
	Channel string   `json:"channel"`
	ID      string   `json:"id"`
	Mode    string   `json:"mode,omitempty"`
	Audit   string   `json:"audit"`
	At      string   `json:"at"`
	Dropped []string `json:"dropped,omitempty"`
}

type AdsPack struct {
	Current      int            `json:"current"`
	FinalURL     string         `json:"final_url"`
	Cells        []AdsCell      `json:"cells"`
	Keywords     []string       `json:"keywords"`
	Headlines    []AdsLine      `json:"headlines"`
	Descriptions []AdsLine      `json:"descriptions"`
	Pexels       []PexelsQuery  `json:"pexels"`
	Avoid        string         `json:"avoid"`
	Looked       []string       `json:"looked,omitempty"`
	Placements   []AdsPlacement `json:"placements"`
}

func ClampAdsStep(n int) int {
	if n < 1 {
		return 1
	}
	if n > 6 {
		return 6
	}
	return n
}

func ClipRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}

func ClipAds(p AdsPack) AdsPack {
	p.Current = ClampAdsStep(p.Current)
	p.FinalURL = strings.TrimSpace(p.FinalURL)
	p.Avoid = strings.TrimSpace(p.Avoid)
	var cells []AdsCell
	for _, c := range p.Cells {
		c.Geo, c.Theme, c.Why = strings.TrimSpace(c.Geo), strings.TrimSpace(c.Theme), strings.TrimSpace(c.Why)
		if c.Geo == "" && c.Theme == "" {
			continue
		}
		cells = append(cells, c)
	}
	p.Cells = cells
	var kw []string
	seen := map[string]bool{}
	for _, k := range p.Keywords {
		k = strings.TrimSpace(k)
		low := strings.ToLower(k)
		if k == "" || seen[low] {
			continue
		}
		seen[low] = true
		kw = append(kw, k)
	}
	p.Keywords = kw
	var heads []AdsLine
	for _, h := range p.Headlines {
		h.Text = ClipRunes(h.Text, adsHeadlineLimit)
		if h.Text == "" {
			continue
		}
		heads = append(heads, h)
		if len(heads) == adsHeadlineCap {
			break
		}
	}
	p.Headlines = heads
	var descs []AdsLine
	for _, d := range p.Descriptions {
		d.Text = ClipRunes(d.Text, adsDescLimit)
		if d.Text == "" {
			continue
		}
		descs = append(descs, d)
		if len(descs) == adsDescCap {
			break
		}
	}
	p.Descriptions = descs
	var px []PexelsQuery
	for _, q := range p.Pexels {
		q.Query, q.Avoid = strings.TrimSpace(q.Query), strings.TrimSpace(q.Avoid)
		if q.Query == "" {
			continue
		}
		px = append(px, q)
		if len(px) == 5 {
			break
		}
	}
	p.Pexels = px
	var places []AdsPlacement
	for _, pl := range p.Placements {
		pl.Channel = strings.ToLower(strings.TrimSpace(pl.Channel))
		pl.Mode = strings.ToLower(strings.TrimSpace(pl.Mode))
		if pl.Mode != "manual" && pl.Mode != "place" {
			pl.Mode = ""
		}
		pl.ID, pl.Audit, pl.At = strings.TrimSpace(pl.ID), strings.TrimSpace(pl.Audit), strings.TrimSpace(pl.At)
		pl.Dropped = uniqStrings(pl.Dropped)
		if pl.Channel == "" {
			continue
		}
		places = append(places, pl)
	}
	p.Placements = places
	var looked []string
	for _, line := range uniqStrings(p.Looked) {
		line = ClipRunes(strings.TrimSpace(line), 80)
		if line == "" {
			continue
		}
		looked = append(looked, line)
		if len(looked) == 12 {
			break
		}
	}
	p.Looked = looked
	return p
}

func AdsCampaignIDs(channel, resource string) (customerID, campaignID string) {
	_ = channel
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return "", ""
	}
	parts := strings.Split(strings.ReplaceAll(resource, "act_", ""), "/")
	for i, p := range parts {
		if i+1 >= len(parts) {
			continue
		}
		switch p {
		case "customers":
			customerID = adsDigits(parts[i+1])
		case "campaigns":
			campaignID = adsDigits(parts[i+1])
		}
	}
	if campaignID == "" {
		campaignID = adsDigits(resource)
	}
	return customerID, campaignID
}

func adsDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (p AdsPack) PlaceReady() string {
	p = ClipAds(p)
	if p.FinalURL == "" {
		return "need a final URL from the Card"
	}
	if len(p.Keywords) == 0 {
		return "need at least one keyword"
	}
	if len(p.Headlines) < 3 {
		return "need at least three headlines"
	}
	if len(p.Descriptions) < 2 {
		return "need at least two descriptions"
	}
	return ""
}

func (p AdsPack) PlacedOn(channel string) bool {
	channel = strings.ToLower(strings.TrimSpace(channel))
	for _, pl := range p.Placements {
		if pl.Channel == channel && pl.ID != "" {
			return true
		}
	}
	return false
}

func (p AdsPack) WithPlacement(pl AdsPlacement) AdsPack {
	p = ClipAds(p)
	pl.Channel = strings.ToLower(strings.TrimSpace(pl.Channel))
	if pl.Channel == "" {
		return p
	}
	var out []AdsPlacement
	for _, old := range p.Placements {
		if old.Channel != pl.Channel {
			out = append(out, old)
		}
	}
	out = append(out, ClipAds(AdsPack{Placements: []AdsPlacement{pl}}).Placements...)
	if len(out) == 0 {
		out = append(out, pl)
	}
	p.Placements = out
	return ClipAds(p)
}

func (p AdsPack) WithPruneNote(channel, note string) AdsPack {
	return p.WithPrune(channel, note, nil)
}

func (p AdsPack) WithPrune(channel, note string, dropped []string) AdsPack {
	p = ClipAds(p)
	channel = strings.ToLower(strings.TrimSpace(channel))
	note = strings.TrimSpace(note)
	if channel == "" {
		return p
	}
	for i, pl := range p.Placements {
		if pl.Channel != channel {
			continue
		}
		if note != "" {
			if pl.Audit != "" && !strings.Contains(pl.Audit, note) {
				pl.Audit = strings.TrimSpace(pl.Audit + " Cut: " + note)
			} else if pl.Audit == "" {
				pl.Audit = "Cut: " + note
			}
		}
		if dropped != nil {
			pl.Dropped = uniqStrings(dropped)
		}
		p.Placements[i] = pl
	}
	return ClipAds(p)
}

func SeedAds(url string) AdsPack {
	return ClipAds(AdsPack{Current: 1, FinalURL: strings.TrimSpace(url)})
}

func ParseAds(raw any) AdsPack {
	if raw == nil {
		return AdsPack{Current: 1}
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return AdsPack{Current: 1}
	}
	var p AdsPack
	if json.Unmarshal(b, &p) != nil {
		return AdsPack{Current: 1}
	}
	return ClipAds(p)
}

func MergeAds(base, extra AdsPack) AdsPack {
	out := ClipAds(base)
	ex := ClipAds(extra)
	if ex.FinalURL != "" {
		out.FinalURL = ex.FinalURL
	}
	if len(ex.Cells) > 0 {
		out.Cells = ex.Cells
	}
	if len(ex.Keywords) > 0 {
		out.Keywords = ex.Keywords
	}
	if len(ex.Headlines) > 0 {
		out.Headlines = ex.Headlines
	}
	if len(ex.Descriptions) > 0 {
		out.Descriptions = ex.Descriptions
	}
	if len(ex.Pexels) > 0 {
		out.Pexels = ex.Pexels
	}
	if ex.Avoid != "" {
		out.Avoid = ex.Avoid
	}
	if len(ex.Placements) > 0 {
		out.Placements = ex.Placements
	}
	if extra.Current > 0 {
		out.Current = ClampAdsStep(extra.Current)
	}
	if len(ex.Looked) > 0 {
		out.Looked = uniqStrings(append(out.Looked, ex.Looked...))
	}
	return ClipAds(out)
}

func AdsMarkdown(p AdsPack, name, slug string) string {
	return AdsMarkdownFor(p, name, slug, AdsModeDiscovery)
}

func AdsMarkdownFor(p AdsPack, name, slug, mode string) string {
	p = ClipAds(p)
	mode = NormalizeAdsMode(mode)
	var b strings.Builder
	b.WriteString("# Search ads pack\n\n")
	if name != "" {
		b.WriteString("Bet: **" + name + "**")
		if slug != "" {
			b.WriteString(" (`" + slug + "`)")
		}
		b.WriteString("\n\n")
	}
	b.WriteString("Legal name: **TheMobileProf Technologies**. One Google Ads account. Discovery spend uses campaign `" + AdsCampaignName(slug, name, AdsModeDiscovery) + "`. Mature spend uses `" + AdsCampaignName(slug, name, AdsModeMature) + "` after a human marks the Card mature.\n\n")
	b.WriteString("A human Accepts, then either follows **Book it yourself** in the Google Ads or Meta UI, or clicks **Place** on the Growth book. Do not do both for the same channel. Discovery Place is a cheap probe: platform-minimum daily in the account currency (usually NGN for Nigerian businesses), never more than remaining/30, all test cells. Mature Place is refused until the Card lists winning geo plus keyword-theme segments, then uses remaining/30 on those cells only, in a separate campaign. Defaults ceiling still clips. Each geo/keyword cell is its own experiment. RSA already rotates headlines. Place stamps utm_campaign to that campaign name and utm_content to the cell so GA4 can score paid sessions. Place runs all Lagos hours so weekday vs weekend and morning/midday/evening/night can show. A 07:00 Lagos job waits a full week, then pauses a cell that spent with 0 clicks, and turns off dayparts that spent with 0 clicks while another daypart got clicks. Conversions wait two weeks, and only if some cell already converted. The Growth book shows GA4 last 7 days (paid sessions and a funnel of events that already fire). Google Search RSA, exact match only, Google Search network only; Meta uses one campaign budget. Campaigns start paused unless Defaults say ENABLED. Pexels stills: search the queries yourself, download, `/file` under **product**. Do not generate faces.\n\n")
	if len(p.Looked) > 0 {
		b.WriteString("## Looked\n\n")
		for _, line := range p.Looked {
			b.WriteString("- " + line + "\n")
		}
		b.WriteString("\nThis office fetched these; the clerk did not call Google. Chat cannot spend.\n\n")
	}
	b.WriteString("## 1. Final URL\n\n")
	if p.FinalURL == "" {
		b.WriteString("[TO BE COMPLETED]\n\n")
	} else {
		b.WriteString(p.FinalURL + "\n\n")
	}
	b.WriteString("## 2. Test cells\n\n")
	if len(p.Cells) == 0 {
		b.WriteString("[TO BE COMPLETED]\n\n")
	} else {
		for i, c := range p.Cells {
			b.WriteString(strconv.Itoa(i+1) + ". **" + c.Geo + "** · " + c.Theme)
			if c.Why != "" {
				b.WriteString(" — " + c.Why)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## 3. Keywords\n\n")
	if len(p.Keywords) == 0 {
		b.WriteString("[TO BE COMPLETED]\n\n")
	} else {
		for _, k := range p.Keywords {
			b.WriteString("- " + k + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## 4. Headlines (≤30)\n\n")
	if len(p.Headlines) == 0 {
		b.WriteString("[TO BE COMPLETED]\n\n")
	} else {
		for _, h := range p.Headlines {
			b.WriteString("- " + h.Text + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## 5. Descriptions (≤90)\n\n")
	if len(p.Descriptions) == 0 {
		b.WriteString("[TO BE COMPLETED]\n\n")
	} else {
		for _, d := range p.Descriptions {
			b.WriteString("- " + d.Text + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## 6. Pexels (human search)\n\n")
	if p.Avoid != "" {
		b.WriteString("Do not download: " + p.Avoid + "\n\n")
	}
	if len(p.Pexels) == 0 {
		b.WriteString("[TO BE COMPLETED]\n\n")
	} else {
		for _, q := range p.Pexels {
			b.WriteString("- [" + q.Query + "](" + q.SearchURL() + ")")
			if q.Avoid != "" {
				b.WriteString(" — skip: " + q.Avoid)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(p.Placements) > 0 {
		b.WriteString("## Placed\n\n")
		for _, pl := range p.Placements {
			b.WriteString("- **" + pl.Channel + "**")
			if pl.Mode != "" {
				b.WriteString(" · " + pl.Mode)
			}
			if pl.ID != "" {
				b.WriteString(" `" + pl.ID + "`")
			}
			if pl.Audit != "" {
				b.WriteString(" — " + pl.Audit)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(AdsManualMarkdown(p, name, slug, mode))
	return b.String()
}

func LooksLikeAds(data map[string]any, task string) bool {
	if CustomerChannel(data) {
		return false
	}
	if strings.EqualFold(str(data["ref_kind"]), "growth_ads") {
		return true
	}
	if strings.EqualFold(str(data["task_type"]), "ads") {
		return true
	}
	a := strings.ToLower(strings.TrimSpace(str(data["action"])))
	if a == "ads" || a == "search-ads" || a == "adwords" {
		return true
	}
	if _, ok := data["ads"]; ok {
		return true
	}
	lower := strings.ToLower(task)
	for _, k := range []string{"google ads", "google ad", "adwords", "search ad", "search ads", "rsa headline", "pexels"} {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func handleAds(ctx context.Context, c llm.Completer, req contract.Request, data map[string]any) (contract.Response, error) {
	pack := ParseAds(data["ads"])
	if pack.Current < 1 {
		pack.Current = 1
	}
	system := adsOverlay + "\n\nAction: ads\n\n" +
		`structured_data MUST include:
- action: ads
- project_slug (from context if present)
- current_step: 1–6
- ads: {current, final_url, cells:[{geo,theme,why,status}], keywords:[], headlines:[{text,status}], descriptions:[{text,status}], pexels:[{query,avoid}], avoid}
- save_deck: true when context has a project
- cannot_send: true
- cannot_spend: true
- record_lead: false
- needs_human: true if they asked you to publish, spend, or fetch Pexels
output_text: the current step’s proposed copy, then ONE question. Do not dump the whole pack.`

	resp, err := departments.Run(ctx, c, system, departments.UserPrompt(req.TaskDescription, data))
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "ads"
	resp.StructuredData["action"] = "ads"
	resp.StructuredData["cannot_send"] = true
	resp.StructuredData["cannot_spend"] = true
	resp.StructuredData["record_lead"] = false
	merged := MergeAds(pack, ParseAds(resp.StructuredData["ads"]))
	if n := int(intFromMap(resp.StructuredData["current_step"])); n > 0 {
		merged.Current = ClampAdsStep(n)
	}
	merged, notes := GuardAds(merged, CardBlob(data["project"]))
	merged.Looked = uniqStrings(append(looksFrom(data), append(merged.Looked, notes...)...))
	resp.StructuredData["ads"] = merged
	resp.StructuredData["current_step"] = merged.Current
	resp.StructuredData["looks"] = merged.Looked
	if projectID := intFrom(data); projectID > 0 || str(nestedSlug(data)) != "" {
		resp.StructuredData["save_deck"] = true
	}
	return resp, err
}

func intFromMap(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}
