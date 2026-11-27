package growth

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// AdsManualStep is one click in the Google Ads or Meta UI after Accept.
type AdsManualStep struct {
	N     int    `json:"n"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func adsManualCampaign(slug, name, mode string) string {
	return AdsCampaignName(slug, name, mode)
}

func (p AdsPack) BookedHow(channel string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	for _, pl := range ClipAds(p).Placements {
		if pl.Channel != channel {
			continue
		}
		if pl.Mode == "manual" {
			return "manual"
		}
		if pl.Mode == "place" || pl.ID != "" {
			return "place"
		}
	}
	return ""
}

func AdsBookConflict(p AdsPack, channel, want string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	want = strings.ToLower(strings.TrimSpace(want))
	how := p.BookedHow(channel)
	if how == "" {
		return ""
	}
	if want == "place" && how == "manual" {
		return "This pack is already booked in the Ads UI. Do not Place the same cells or you pay twice."
	}
	if want == "manual" && how == "place" {
		return "n8n already placed this channel. Do not also create these campaigns in the Ads UI."
	}
	return ""
}

func adsManualCells(p AdsPack) []AdsCell {
	p = ClipAds(p)
	if len(p.Cells) > 0 {
		return p.Cells
	}
	return []AdsCell{{Geo: "Nigeria", Theme: "search"}}
}

func AdsManualGoogle(p AdsPack, name, slug, mode string) []AdsManualStep {
	p = ClipAds(p)
	mode = NormalizeAdsMode(mode)
	camp := adsManualCampaign(slug, name, mode)
	div := AdsReserveDivisor(mode)
	cells := adsManualCells(p)
	var cellLines []string
	for i, c := range cells {
		label := strings.TrimSpace(c.Geo)
		if c.Theme != "" {
			if label != "" {
				label += " · "
			}
			label += c.Theme
		}
		ag := camp + " · " + label
		track := AdsTrackURL(p.FinalURL, "google", camp, AdsCellContent(c))
		cellLines = append(cellLines, strconv.Itoa(i+1)+". Ad group name `"+ag+"`. Location is "+c.Geo+". Final URL is "+track+".")
	}
	var kw []string
	for _, k := range p.Keywords {
		kw = append(kw, "["+k+"]")
	}
	if len(kw) == 0 {
		kw = []string{"[keyword from the pack]"}
	}
	var heads []string
	for _, h := range p.Headlines {
		heads = append(heads, h.Text+" ("+strconv.Itoa(utf8.RuneCountInString(h.Text))+"/30)")
	}
	var descs []string
	for _, d := range p.Descriptions {
		descs = append(descs, d.Text+" ("+strconv.Itoa(utf8.RuneCountInString(d.Text))+"/90)")
	}
	return []AdsManualStep{
		{1, "Open the company account", "Open ads.google.com and the Google Ads customer on Defaults → Ads. Do not open a second company account."},
		{2, "Start a Search campaign", "Campaigns → New campaign. Skip Performance Max and Smart campaigns. Choose Search. Goal is website traffic (clicks), not a lead form Google invents."},
		{3, "Name the campaign for this spend mode", "Campaign name is `" + camp + "` exactly. Discovery spend is `{slug}-discovery`. Mature spend is `{slug}`. Do not reuse the discovery campaign for substantial spend. Prune and GA4 join on that name."},
		{4, "Google Search only", "Turn off Search partners and the Display Network. This office tests Google Search only."},
		{5, "Locations from the pack", "Add each named place below as a location (not a radius around you). Do not add age or gender. Nigeria stays in the list if a cell is only a theme."},
		{6, "Daily budget", adsManualBudgetBody("Google", div, mode)},
		{7, "Bidding", "Choose Maximize clicks. Set a CPC ceiling at that same daily amount. Do not turn on Target CPA or Maximize conversions unless a conversion already fired for this bet."},
		{8, "One ad group per cell", "Create one ad group per line below. Paste the final URL for that cell so UTM can tell the cells apart.\n" + strings.Join(cellLines, "\n")},
		{9, "Exact match keywords", "In each ad group, add these as Exact match only (square brackets). Do not add phrase or broad match.\n" + strings.Join(kw, "\n")},
		{10, "Responsive Search Ad", "Paste every headline (each stays ≤30 characters) and every description (each stays ≤90). Advertiser is TheMobileProf Technologies.\nHeadlines:\n" + strings.Join(heads, "\n") + "\nDescriptions:\n" + strings.Join(descs, "\n")},
		{11, "All Lagos hours for a week", "Leave ad schedule on all days and all hours for the first seven days so weekday versus weekend and morning versus evening can show. Do not narrow hours yourself on day one."},
		{12, "Start paused", "Set the campaign to Paused unless Defaults → Ads says ENABLED. Save. Then paste the campaign id (the digits after /campaigns/ in the URL) on this pack with I booked it myself, so prune can score it. Do not also click Place on Google Search."},
	}
}

func AdsManualMeta(p AdsPack, name, slug, mode string) []AdsManualStep {
	p = ClipAds(p)
	mode = NormalizeAdsMode(mode)
	camp := adsManualCampaign(slug, name, mode)
	div := AdsReserveDivisor(mode)
	cells := adsManualCells(p)
	var cellLines []string
	for i, c := range cells {
		label := strings.TrimSpace(c.Geo)
		if c.Theme != "" {
			if label != "" {
				label += " · "
			}
			label += c.Theme
		}
		track := AdsTrackURL(p.FinalURL, "meta", camp, AdsCellContent(c))
		cellLines = append(cellLines, strconv.Itoa(i+1)+". Ad set name `"+camp+" · "+label+"`. Geo is "+c.Geo+" (Nigeria if the cell has no city). Destination URL is "+track+".")
	}
	var copyLines []string
	for _, h := range p.Headlines {
		copyLines = append(copyLines, h.Text)
	}
	for _, d := range p.Descriptions {
		copyLines = append(copyLines, d.Text)
	}
	return []AdsManualStep{
		{1, "Open the company ad account", "Open adsmanager.facebook.com and the Meta ad account plus Page on Defaults → Ads. Do not create a second ad account."},
		{2, "Traffic campaign", "Create → Traffic (website). Skip Advantage+ shopping and leads campaigns Google or Meta upsell. Campaign name is `" + camp + "` exactly. Discovery spend is `{slug}-discovery`. Mature spend is `{slug}`."},
		{3, "One campaign budget", adsManualBudgetBody("Meta", div, mode)},
		{4, "One ad set per cell", "Create one ad set per line below so prune can pause a dead cell without killing the rest. Age and gender stay open. Do not add interest stacks this office did not put on the Card.\n" + strings.Join(cellLines, "\n")},
		{5, "Creative from the pack", "Use the Page on Defaults. Paste pack headlines and descriptions as primary text and headline. Search the Pexels queries yourself and `/file` stills under product — do not generate faces.\n" + strings.Join(copyLines, "\n")},
		{6, "All hours for a week", "Leave the ad set schedule on all hours for the first seven days. Start Paused unless Defaults → Ads says ENABLED. Save, then paste the campaign id on this pack with I booked it myself. Do not also click Place on Meta."},
	}
}

func AdsManualMarkdown(p AdsPack, name, slug, mode string) string {
	p = ClipAds(p)
	mode = NormalizeAdsMode(mode)
	div := AdsReserveDivisor(mode)
	camp := adsManualCampaign(slug, name, mode)
	var b strings.Builder
	b.WriteString("## After you Accept — pick one path per channel\n\n")
	b.WriteString("Chat still cannot spend. After Accept, either follow **Book it yourself** in the Google Ads or Meta UI, or click **Place** so n8n books the same campaign. Do not do both for the same channel, or you pay twice.\n\n")
	b.WriteString("Discovery spend and mature spend are different campaigns. Discovery is `" + AdsCampaignName(slug, name, AdsModeDiscovery) + "` (cheap probe, all cells, remaining/" + strconv.Itoa(AdsDiscoveryReserve) + "). Mature is `" + AdsCampaignName(slug, name, AdsModeMature) + "` (winning Search segments only, remaining/" + strconv.Itoa(AdsMatureReserve) + ") after a human marks the Card mature. This pack is **" + mode + "** (`" + camp + "`, remaining/" + strconv.Itoa(div) + ").\n\n")
	b.WriteString("Place (automated) talks to n8n. n8n holds Google Ads OAuth and Facebook Ads OAuth. Google sunset developer tokens on 9 September 2026; API access is the Cloud project that owns the Google Ads OAuth client. Enable the Google Ads API there and apply for Explorer (or higher) on the Google Ads API Overview page so production accounts work. Go never holds those tokens.\n\n")
	b.WriteString("### Book it yourself · Google Ads\n\n")
	for _, s := range AdsManualGoogle(p, name, slug, mode) {
		b.WriteString(strconv.Itoa(s.N) + ". **" + s.Title + ".** " + strings.ReplaceAll(s.Body, "\n", "\n   ") + "\n\n")
	}
	b.WriteString("### Book it yourself · Meta\n\n")
	for _, s := range AdsManualMeta(p, name, slug, mode) {
		b.WriteString(strconv.Itoa(s.N) + ". **" + s.Title + ".** " + strings.ReplaceAll(s.Body, "\n", "\n   ") + "\n\n")
	}
	b.WriteString("### Place automatically\n\n")
	b.WriteString("On the Growth book, click **Place on Google Search** or **Place on Meta**. n8n reads remaining funds. Discovery stays a platform-minimum probe (never more than remaining/" + strconv.Itoa(AdsDiscoveryReserve) + ") on every cell. Mature Place is refused until the Card lists winning geo plus keyword-theme segments, then uses remaining/" + strconv.Itoa(AdsMatureReserve) + " on those cells only. Exact match, Google Search only, one experiment per cell, all Lagos hours for a week. Campaigns start paused unless Defaults say ENABLED. After Place, do not also create those campaigns in the Ads UI.\n\n")
	return b.String()
}

func adsManualBudgetBody(vendor string, div int, mode string) string {
	if NormalizeAdsMode(mode) == AdsModeMature {
		return "Set a daily amount " + vendor + " will accept in this account currency. Never more than remaining funds divided by " + strconv.Itoa(div) + ". This is substantial spend on winning segments, not the discovery probe. If Defaults → Ads has a daily ceiling, do not exceed it."
	}
	return "Set the smallest daily amount " + vendor + " will accept in this account currency (often 1). Never more than remaining funds divided by " + strconv.Itoa(div) + ". This is discovery spend, not substantial scale. If Defaults → Ads has a daily ceiling, do not exceed it."
}
