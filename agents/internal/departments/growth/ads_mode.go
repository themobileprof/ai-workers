package growth

import (
	"errors"
	"strings"
)

const (
	AdsModeDiscovery = "discovery"
	AdsModeMature    = "mature"

	// AdsDiscoveryReserve spreads remaining funds over about one month of daily
	// budget. A discovery cell needs a full week of Lagos hours; a month is about
	// four weekly iterations. Nigerian Google Ads accounts are usually NGN, so
	// remaining and the platform-minimum daily are in naira, not dollars.
	AdsDiscoveryReserve = 30
	AdsMatureReserve    = 30
)

func NormalizeAdsMode(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case AdsModeMature, "scale", "substantial":
		return AdsModeMature
	default:
		return AdsModeDiscovery
	}
}

func AdsReserveDivisor(mode string) int {
	if NormalizeAdsMode(mode) == AdsModeMature {
		return AdsMatureReserve
	}
	return AdsDiscoveryReserve
}

// AdsCampaignName is the Google/Meta campaign and utm_campaign value.
// Discovery spend is always `{slug}-discovery`. Mature spend is `{slug}`.
func AdsCampaignName(slug, name, mode string) string {
	base := strings.TrimSpace(slug)
	if base == "" {
		base = strings.TrimSpace(name)
	}
	if base == "" {
		return "the bet slug on the Card"
	}
	if NormalizeAdsMode(mode) == AdsModeMature {
		return strings.TrimSuffix(base, "-discovery")
	}
	if strings.HasSuffix(base, "-discovery") {
		return base
	}
	return base + "-discovery"
}

func ParseAdsSegments(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ';' || r == ','
	})
	seen := map[string]bool{}
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ClipRunes(p, 80))
		if len(out) == 8 {
			break
		}
	}
	return out
}

func ClipAdsSegments(raw string) string {
	return strings.Join(ParseAdsSegments(raw), "\n")
}

func CellMatchesSegment(c AdsCell, segs []string) bool {
	content := strings.ToLower(AdsCellContent(c))
	geo := adsUTMToken(c.Geo, 80)
	theme := adsUTMToken(c.Theme, 80)
	label := strings.ToLower(strings.TrimSpace(c.Geo + " " + c.Theme))
	for _, raw := range segs {
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "" {
			continue
		}
		tok := adsUTMToken(s, 80)
		if tok != "" && (tok == content || strings.Contains(content, tok) || strings.Contains(tok, content)) {
			return true
		}
		if geo != "" && strings.Contains(tok, geo) && (theme == "" || strings.Contains(tok, theme)) {
			return true
		}
		if label != "" && (strings.Contains(label, s) || strings.Contains(s, label)) {
			return true
		}
	}
	return false
}

func FilterAdsCells(cells []AdsCell, mode, segments string) ([]AdsCell, error) {
	if len(cells) == 0 {
		cells = []AdsCell{{Geo: "Nigeria", Theme: "search"}}
	}
	if NormalizeAdsMode(mode) != AdsModeMature {
		return cells, nil
	}
	segs := ParseAdsSegments(segments)
	if len(segs) == 0 {
		return nil, errors.New("mature spend needs winning Search segments on the Card (geo plus keyword theme). Discovery Place does not.")
	}
	var out []AdsCell
	for _, c := range cells {
		if CellMatchesSegment(c, segs) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("none of the Card winning segments match this pack’s cells. Edit the segments or the pack before substantial Place.")
	}
	return out, nil
}
