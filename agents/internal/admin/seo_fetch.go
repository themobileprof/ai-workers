package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const seoFetchMaxPages = 8
const seoFetchMaxBytes = 400 << 10

type fetchedPage struct {
	URL    string        `json:"url"`
	Title  string        `json:"title"`
	H1     string        `json:"h1"`
	Meta   string        `json:"meta"`
	Text   string        `json:"text"`
	JSONLD []jsonLDBlock `json:"json_ld,omitempty"`
}

type jsonLDBlock struct {
	Types []string `json:"types"`
	Raw   string   `json:"raw"`
}

var (
	seoFetchClient = &http.Client{Timeout: 8 * time.Second}
	reScript       = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle        = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reTitle        = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reH1           = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	reBody         = regexp.MustCompile(`(?is)<body[^>]*>(.*)</body>`)
	reMetaDesc     = regexp.MustCompile(`(?is)<meta\s[^>]*name\s*=\s*["']description["'][^>]*>`)
	reMetaContent  = regexp.MustCompile(`(?is)content\s*=\s*["']([^"']*)["']`)
	reHref         = regexp.MustCompile(`(?is)<a\s[^>]*href\s*=\s*["']([^"']+)["']`)
	reJSONLD       = regexp.MustCompile(`(?is)<script\b[^>]*type\s*=\s*["']application/ld\+json["'][^>]*>(.*?)</script>`)
	reTag          = regexp.MustCompile(`<[^>]+>`)
)

func fetchSitePages(start string) []fetchedPage {
	start = strings.TrimSpace(start)
	if start == "" {
		return nil
	}
	base, err := url.Parse(start)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil
	}
	seen := map[string]bool{}
	queue := []string{base.String()}
	var out []fetchedPage
	for len(queue) > 0 && len(out) < seoFetchMaxPages {
		u := queue[0]
		queue = queue[1:]
		if seen[u] {
			continue
		}
		seen[u] = true
		page, hrefs, ok := fetchOnePage(base, u)
		if !ok {
			continue
		}
		out = append(out, page)
		if len(out) == 1 {
			for _, h := range hrefs {
				if !seen[h] {
					queue = append(queue, h)
				}
			}
		}
	}
	return out
}

func fetchHomePage(start string) []fetchedPage {
	start = strings.TrimSpace(start)
	if start == "" {
		return nil
	}
	base, err := url.Parse(start)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return nil
	}
	page, _, ok := fetchOnePage(base, base.String())
	if !ok {
		return nil
	}
	return []fetchedPage{page}
}

func fetchOnePage(origin *url.URL, raw string) (fetchedPage, []string, bool) {
	u, err := origin.Parse(raw)
	if err != nil {
		return fetchedPage{}, nil, false
	}
	if !sameSEOHost(origin, u) || !seoCrawlable(u) {
		return fetchedPage{}, nil, false
	}
	u.Fragment = ""
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return fetchedPage{}, nil, false
	}
	req.Header.Set("User-Agent", "TheMobileProfSEO/1.0 (own-site review)")
	resp, err := seoFetchClient.Do(req)
	if err != nil {
		return fetchedPage{}, nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fetchedPage{}, nil, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, seoFetchMaxBytes))
	if err != nil {
		return fetchedPage{}, nil, false
	}
	html := string(body)
	page := fetchedPage{
		URL:    u.String(),
		Title:  collapseSpace(stripTags(firstSub(reTitle, html))),
		H1:     collapseSpace(stripTags(firstSub(reH1, html))),
		Meta:   metaDescription(html),
		Text:   clipCRM(htmlToText(pageBodyHTML(html)), 4000),
		JSONLD: extractJSONLD(html),
	}
	var hrefs []string
	for _, m := range reHref.FindAllStringSubmatch(html, 40) {
		if len(m) < 2 {
			continue
		}
		nu, err := u.Parse(strings.TrimSpace(m[1]))
		if err != nil || !sameSEOHost(origin, nu) || !seoCrawlable(nu) {
			continue
		}
		nu.Fragment = ""
		hrefs = append(hrefs, nu.String())
	}
	return page, hrefs, true
}

func sameSEOHost(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Hostname(), b.Hostname())
}

func seoCrawlable(u *url.URL) bool {
	path := strings.ToLower(u.Path)
	for _, ext := range []string{".pdf", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".zip", ".css", ".js", ".mp4", ".woff", ".woff2"} {
		if strings.HasSuffix(path, ext) {
			return false
		}
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func pageBodyHTML(html string) string {
	if m := reBody.FindStringSubmatch(html); len(m) >= 2 {
		return m[1]
	}
	html = reTitle.ReplaceAllString(html, " ")
	html = reMetaDesc.ReplaceAllString(html, " ")
	return html
}

func htmlToText(html string) string {
	html = reScript.ReplaceAllString(html, " ")
	html = reStyle.ReplaceAllString(html, " ")
	return collapseSpace(stripTags(html))
}

func stripTags(s string) string {
	return reTag.ReplaceAllString(s, " ")
}

func collapseSpace(s string) string {
	var b strings.Builder
	space := true
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		space = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func firstSub(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func metaDescription(html string) string {
	tag := reMetaDesc.FindString(html)
	if tag == "" {
		return ""
	}
	return collapseSpace(firstSub(reMetaContent, tag))
}

func pagespeedSnapshot(pageURL string) map[string]any {
	key := strings.TrimSpace(os.Getenv("PAGESPEED_API_KEY"))
	if key == "" || pageURL == "" {
		return nil
	}
	u := "https://www.googleapis.com/pagespeedonline/v5/runPagespeed?strategy=mobile&url=" + url.QueryEscape(pageURL) + "&key=" + url.QueryEscape(key)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	resp, err := seoFetchClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil
	}
	// Keep a small, honest slice — never invent scores if Google omitted them.
	body := string(raw)
	out := map[string]any{"source": "pagespeed_insights", "url": pageURL}
	if i := strings.Index(body, `"score":`); i >= 0 && i < 200000 {
		out["raw_excerpt"] = clipCRM(body[i:min(len(body), i+400)], 400)
	}
	return out
}

func extractJSONLD(html string) []jsonLDBlock {
	var out []jsonLDBlock
	for _, m := range reJSONLD.FindAllStringSubmatch(html, 8) {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		if raw == "" {
			continue
		}
		out = append(out, jsonLDBlock{Types: jsonLDTypes(raw), Raw: clipCRM(raw, 1500)})
	}
	return out
}

func jsonLDTypes(raw string) []string {
	var v any
	if json.Unmarshal([]byte(raw), &v) != nil {
		return nil
	}
	seen := map[string]bool{}
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case []any:
			for _, item := range t {
				walk(item)
			}
		case map[string]any:
			if g, ok := t["@graph"]; ok {
				walk(g)
			}
			switch typ := t["@type"].(type) {
			case string:
				if typ != "" {
					seen[typ] = true
				}
			case []any:
				for _, item := range typ {
					if s, ok := item.(string); ok && s != "" {
						seen[s] = true
					}
				}
			}
		}
	}
	walk(v)
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	return out
}

func schemaTypesFromPages(pages []fetchedPage) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range pages {
		for _, block := range p.JSONLD {
			for _, t := range block.Types {
				if t == "" || seen[t] {
					continue
				}
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	return out
}

func fetchRobotsNotes(pageURL string) map[string]any {
	u, err := url.Parse(pageURL)
	if err != nil || u.Host == "" {
		return nil
	}
	robots, err := u.Parse("/robots.txt")
	if err != nil {
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, robots.String(), nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "TheMobileProfSEO/1.0 (own-site review)")
	resp, err := seoFetchClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if err != nil {
		return nil
	}
	body := string(raw)
	out := map[string]any{"url": robots.String(), "excerpt": clipCRM(collapseSpace(body), 800)}
	var blocked []string
	for _, bot := range []string{"GPTBot", "Google-Extended", "ChatGPT-User", "PerplexityBot", "ClaudeBot"} {
		if robotsBlocks(body, bot) {
			blocked = append(blocked, bot)
		}
	}
	if len(blocked) > 0 {
		out["blocked_ai"] = blocked
	}
	return out
}

func robotsBlocks(body, bot string) bool {
	lower := strings.ToLower(body)
	needle := "user-agent: " + strings.ToLower(bot)
	i := strings.Index(lower, needle)
	if i < 0 {
		return false
	}
	rest := lower[i:]
	if j := strings.Index(rest[len(needle):], "user-agent:"); j >= 0 {
		rest = rest[:len(needle)+j]
	}
	return strings.Contains(rest, "disallow: /")
}
