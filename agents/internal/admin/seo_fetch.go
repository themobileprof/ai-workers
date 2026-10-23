package admin

import (
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
	URL   string `json:"url"`
	Title string `json:"title"`
	H1    string `json:"h1"`
	Meta  string `json:"meta"`
	Text  string `json:"text"`
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
		URL:   u.String(),
		Title: collapseSpace(stripTags(firstSub(reTitle, html))),
		H1:    collapseSpace(stripTags(firstSub(reH1, html))),
		Meta:  metaDescription(html),
		Text:  clipCRM(htmlToText(pageBodyHTML(html)), 4000),
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
