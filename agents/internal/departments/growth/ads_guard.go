package growth

import (
	"net/url"
	"strings"
	"unicode"
)

var adsStopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "for": true, "from": true,
	"with": true, "your": true, "you": true, "our": true, "to": true, "of": true, "in": true,
	"on": true, "at": true, "is": true, "are": true, "get": true, "got": true, "buy": true,
	"now": true, "new": true, "best": true, "www": true, "com": true, "ng": true, "http": true,
	"https": true,
}

var adsClaimWords = []string{"waitlist", "mau", "gmv", "cpc", "ltv"}

var adsFaceWords = []string{"woman", "women", "man", "men", "girl", "boy", "face", "portrait", "smile", "smiling"}

// CardBlob is the lowercase Card text GuardAds checks against.
func CardBlob(v any) string {
	m, _ := v.(map[string]any)
	if m == nil {
		return ""
	}
	var parts []string
	for _, k := range []string{"one_liner", "hypothesis", "mission", "notes", "funnel_steps", "url", "name", "slug"} {
		parts = append(parts, str(m[k]))
	}
	if raw := str(m["url"]); raw != "" {
		if u, err := url.Parse(raw); err == nil {
			parts = append(parts, strings.ReplaceAll(u.Host, ".", " "))
			parts = append(parts, strings.ReplaceAll(u.Path, "/", " "))
		}
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func CardAllows(blob, phrase string) bool {
	blob = strings.ToLower(strings.TrimSpace(blob))
	phrase = strings.TrimSpace(phrase)
	if phrase == "" {
		return true
	}
	if blob == "" {
		return false
	}
	ok := false
	for _, tok := range adsTokens(phrase) {
		if adsStopWords[tok] {
			continue
		}
		ok = true
		if !strings.Contains(blob, tok) {
			return false
		}
	}
	return ok
}

// GuardAds drops geo, keywords, and Pexels queries whose content words are not on the Card.
// Headlines still ClipAds length; invented waitlist/MAU/CPC claims are dropped unless the Card says them.
func GuardAds(p AdsPack, blob string) (AdsPack, []string) {
	p = ClipAds(p)
	blob = strings.ToLower(strings.TrimSpace(blob))
	if blob == "" {
		return p, nil
	}
	var notes []string
	var cells []AdsCell
	for _, c := range p.Cells {
		if c.Geo != "" && !CardAllows(blob, c.Geo) {
			notes = append(notes, "dropped geo "+c.Geo+" (not on Card)")
			continue
		}
		if c.Theme != "" && !CardAllows(blob, c.Theme) {
			notes = append(notes, "dropped theme "+c.Theme+" (not on Card)")
			continue
		}
		cells = append(cells, c)
	}
	p.Cells = cells
	var kw []string
	for _, k := range p.Keywords {
		if !CardAllows(blob, k) {
			notes = append(notes, "dropped keyword "+k+" (not on Card)")
			continue
		}
		kw = append(kw, k)
	}
	p.Keywords = kw
	var heads []AdsLine
	for _, h := range p.Headlines {
		if claim := adsForbiddenClaim(blob, h.Text); claim != "" {
			notes = append(notes, "dropped headline claim "+claim)
			continue
		}
		heads = append(heads, h)
	}
	p.Headlines = heads
	var descs []AdsLine
	for _, d := range p.Descriptions {
		if claim := adsForbiddenClaim(blob, d.Text); claim != "" {
			notes = append(notes, "dropped description claim "+claim)
			continue
		}
		descs = append(descs, d)
	}
	p.Descriptions = descs
	var px []PexelsQuery
	for _, q := range p.Pexels {
		if adsHasFaceWord(q.Query) {
			notes = append(notes, "dropped Pexels face "+q.Query)
			continue
		}
		if !CardAllows(blob, q.Query) {
			notes = append(notes, "dropped Pexels "+q.Query+" (not on Card)")
			continue
		}
		px = append(px, q)
	}
	p.Pexels = px
	p.Looked = uniqStrings(append(p.Looked, notes...))
	return ClipAds(p), notes
}

func adsForbiddenClaim(blob, text string) string {
	low := strings.ToLower(text)
	for _, w := range adsClaimWords {
		if strings.Contains(low, w) && !strings.Contains(blob, w) {
			return w
		}
	}
	return ""
}

func adsHasFaceWord(phrase string) bool {
	low := strings.ToLower(phrase)
	for _, w := range adsFaceWords {
		if strings.Contains(low, w) {
			return true
		}
	}
	return false
}

func looksFrom(data map[string]any) []string {
	switch v := data["looks"].(type) {
	case []string:
		return v
	case []any:
		var out []string
		for _, x := range v {
			s := strings.TrimSpace(str(x))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func adsTokens(phrase string) []string {
	var b strings.Builder
	var out []string
	flush := func() {
		t := strings.ToLower(strings.TrimSpace(b.String()))
		b.Reset()
		if t == "" {
			return
		}
		if len([]rune(t)) < 3 {
			return
		}
		out = append(out, t)
	}
	for _, r := range phrase {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}
