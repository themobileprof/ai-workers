package admin

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

type seoSeed struct {
	URL      string
	Keyword  string
	Title    string
	Meta     string
	H1       string
	Cites    []string
	FAQs     []seoFAQ
	JSONLD   string
	Types    []string
	Offsite  []string
	FromPage bool
}

type seoAskForm struct {
	Action, URL, Keyword, Feedback string
	Title, Meta, H1, Body, JSONLD  string
	CitesRaw, FAQsRaw, OffsiteRaw  string
}

var reSentenceSplit = regexp.MustCompile(`(?s)([^.!?]+[.!?]|[^.!?]+$)`)

func (s seoSeed) CitesText() string { return strings.Join(s.Cites, "\n") }

func (s seoSeed) FAQsText() string {
	var b strings.Builder
	for _, f := range s.FAQs {
		b.WriteString("Q: " + f.Q + "\nA: " + f.A + "\n\n")
	}
	return strings.TrimSpace(b.String())
}

func (s seoSeed) OffsiteText() string { return strings.Join(s.Offsite, "\n") }

func seoAskFromForm(r interface{ FormValue(string) string }) seoAskForm {
	return seoAskForm{
		Action:     r.FormValue("action"),
		URL:        r.FormValue("url"),
		Keyword:    r.FormValue("keyword"),
		Feedback:   r.FormValue("human_feedback"),
		Title:      r.FormValue("title"),
		Meta:       r.FormValue("meta"),
		H1:         r.FormValue("h1"),
		Body:       r.FormValue("body"),
		JSONLD:     r.FormValue("json_ld"),
		CitesRaw:   r.FormValue("cites"),
		FAQsRaw:    r.FormValue("faqs"),
		OffsiteRaw: r.FormValue("offsite"),
	}
}

func loadSeoSeed(p Project, briefs []SeoBrief) seoSeed {
	pageURL := strings.TrimSpace(p.URL)
	if pageURL == "" {
		return seoSeed{}
	}
	return buildSeoSeed(fetchHomePage(pageURL), p, briefs)
}

func buildSeoSeed(pages []fetchedPage, p Project, briefs []SeoBrief) seoSeed {
	seed := seoSeed{URL: strings.TrimSpace(p.URL), FromPage: len(pages) > 0}
	if len(pages) > 0 {
		home := pages[0]
		seed.URL = firstNonEmpty(home.URL, seed.URL)
		seed.Title = home.Title
		seed.Meta = home.Meta
		seed.H1 = home.H1
	}
	inv := keywordInventory(pages)
	if km := keywordMap(pages, inv); len(km) > 0 {
		seed.Keyword = km[0].BodyPrimary
	}
	if seed.Keyword == "" && p.OneLiner != "" {
		grams := phraseGrams(p.OneLiner, 5)
		if len(grams) > 0 {
			seed.Keyword = grams[0]
		}
	}
	if seed.Keyword == "" {
		seed.Keyword = collapseSpace(strings.ToLower(firstNonEmpty(seed.H1, seed.Title)))
	}
	if d := latestLiveSeo(briefs, "cite"); d.Body != "" {
		seed.Cites = citesFromBody(d.Body)
	}
	if len(seed.Cites) == 0 {
		seed.Cites = extractCites(pages, p)
	}
	if d := latestLiveSeo(briefs, "faq"); d.Body != "" {
		seed.FAQs = faqsFromBody(d.Body)
	}
	if len(seed.FAQs) == 0 {
		seed.FAQs = extractFAQs(pages, seed.Cites, p)
	}
	seed.Types = schemaTypesFromPages(pages)
	if d := latestLiveSeo(briefs, "schema"); strings.TrimSpace(d.Body) != "" {
		if raw := jsonFromBody(d.Body); raw != "" {
			seed.JSONLD = raw
		}
	}
	if seed.JSONLD == "" {
		seed.JSONLD = proposeSchemaJSON(pages, p, seed.FAQs)
	}
	seed.Offsite = extractOffsiteHints(pages, p)
	if seed.Title == "" {
		if d := latestLiveSeo(briefs, "onsite"); d.Title != "" {
			seed.Title = d.Title
		}
	}
	return seed
}

func applyAskForm(seed seoSeed, form seoAskForm) seoSeed {
	if v := strings.TrimSpace(form.URL); v != "" {
		seed.URL = v
	}
	if v := strings.TrimSpace(form.Keyword); v != "" {
		seed.Keyword = v
	}
	if v := strings.TrimSpace(form.Title); v != "" {
		seed.Title = v
	}
	if v := strings.TrimSpace(form.Meta); v != "" {
		seed.Meta = v
	}
	if v := strings.TrimSpace(form.H1); v != "" {
		seed.H1 = v
	}
	if v := strings.TrimSpace(form.JSONLD); v != "" {
		seed.JSONLD = v
	}
	if cites := parseCiteLines(form.CitesRaw); len(cites) > 0 {
		seed.Cites = cites
	}
	if faqs := parseFAQText(form.FAQsRaw); len(faqs) > 0 {
		seed.FAQs = faqs
	}
	if items := parseCiteLines(form.OffsiteRaw); len(items) > 0 {
		seed.Offsite = items
	}
	return seed
}

func extractCites(pages []fetchedPage, p Project) []string {
	names := citeNames(p, pages)
	var scored []struct {
		s string
		n int
	}
	seen := map[string]bool{}
	for _, page := range pages {
		for _, sent := range splitSentences(page.Text) {
			sent = collapseSpace(sent)
			if !usableCite(sent) || seen[strings.ToLower(sent)] {
				continue
			}
			seen[strings.ToLower(sent)] = true
			n := citeScore(sent, names)
			if n <= 0 {
				continue
			}
			scored = append(scored, struct {
				s string
				n int
			}{sent, n})
		}
	}
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].n > scored[i].n {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	out := make([]string, 0, 6)
	for _, row := range scored {
		out = append(out, row.s)
		if len(out) >= 6 {
			break
		}
	}
	return out
}

func citeNames(p Project, pages []fetchedPage) []string {
	names := []string{seoCompanyName, p.Name, p.Slug, "Lagos", "Ikeja", "Nigeria", "Academy"}
	for _, page := range pages {
		if page.H1 != "" {
			names = append(names, page.H1)
		}
	}
	var out []string
	for _, n := range names {
		n = collapseSpace(n)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

func usableCite(s string) bool {
	s = collapseSpace(s)
	if len(s) < 40 || len(s) > 180 {
		return false
	}
	letters := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	return letters >= 24
}

func citeScore(s string, names []string) int {
	lower := strings.ToLower(s)
	n := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			n += 2
			break
		}
	}
	if strings.Contains(s, "₦") || strings.Contains(lower, "ngn") || strings.Contains(lower, "naira") {
		n += 2
	}
	for _, name := range names {
		if name != "" && strings.Contains(lower, strings.ToLower(name)) {
			n += 3
		}
	}
	if strings.Contains(lower, "lagos") || strings.Contains(lower, "ikeja") {
		n += 1
	}
	return n
}

func extractFAQs(pages []fetchedPage, cites []string, p Project) []seoFAQ {
	var out []seoFAQ
	seen := map[string]bool{}
	add := func(q, a string) {
		q, a = collapseSpace(q), collapseSpace(a)
		if q == "" || a == "" {
			return
		}
		if !strings.HasSuffix(q, "?") {
			q += "?"
		}
		key := strings.ToLower(q)
		if seen[key] || len(out) >= 6 {
			return
		}
		seen[key] = true
		out = append(out, seoFAQ{Q: q, A: a})
	}
	for _, page := range pages {
		sents := splitSentences(page.Text)
		for i, sent := range sents {
			sent = collapseSpace(sent)
			if strings.HasSuffix(sent, "?") && i+1 < len(sents) {
				add(sent, sents[i+1])
			}
		}
	}
	if len(pages) > 0 && pages[0].H1 != "" && len(cites) > 0 {
		add("What is "+pages[0].H1+"?", cites[0])
	}
	if p.Name != "" && p.OneLiner != "" {
		add("What does "+p.Name+" do?", p.OneLiner)
	}
	for _, sent := range cites {
		lower := strings.ToLower(sent)
		if strings.Contains(lower, "lagos") || strings.Contains(lower, "ikeja") {
			add("Where is "+firstNonEmpty(p.Name, "this")+"?", sent)
			break
		}
	}
	return out
}

func extractOffsiteHints(pages []fetchedPage, p Project) []string {
	blob := strings.ToLower(p.OneLiner)
	for _, page := range pages {
		blob += " " + strings.ToLower(page.Text+" "+page.Title)
	}
	var out []string
	if strings.Contains(blob, "google") && (strings.Contains(blob, "business") || strings.Contains(blob, "maps") || strings.Contains(blob, "listing")) {
		out = append(out, "Confirm NAP on Google Business for "+firstNonEmpty(p.Name, seoCompanyName))
	}
	if strings.Contains(blob, "partner") || strings.Contains(blob, "listed") || strings.Contains(blob, "directory") {
		out = append(out, "Check the listing already named on the "+firstNonEmpty(p.Name, "page")+" site")
	}
	return out
}

func proposeSchemaJSON(pages []fetchedPage, p Project, faqs []seoFAQ) string {
	var graph []map[string]any
	seen := map[string]bool{}
	for _, page := range pages {
		for _, block := range page.JSONLD {
			for _, t := range block.Types {
				seen[t] = true
			}
			var v any
			if json.Unmarshal([]byte(block.Raw), &v) == nil {
				switch t := v.(type) {
				case []any:
					for _, item := range t {
						if m, ok := item.(map[string]any); ok {
							graph = append(graph, m)
						}
					}
				case map[string]any:
					if g, ok := t["@graph"].([]any); ok {
						for _, item := range g {
							if m, ok := item.(map[string]any); ok {
								graph = append(graph, m)
							}
						}
					} else {
						graph = append(graph, t)
					}
				}
			}
		}
	}
	origin := ""
	if len(pages) > 0 {
		origin = pages[0].URL
	} else {
		origin = p.URL
	}
	if !seen["Organization"] {
		org := map[string]any{
			"@type": "Organization",
			"name":  seoCompanyName,
		}
		if origin != "" {
			org["url"] = originHost(origin)
		}
		graph = append(graph, org)
	}
	if len(faqs) > 0 && !seen["FAQPage"] {
		var ents []map[string]any
		for _, f := range faqs {
			ents = append(ents, map[string]any{
				"@type": "Question",
				"name":  f.Q,
				"acceptedAnswer": map[string]any{
					"@type": "Answer",
					"text":  f.A,
				},
			})
		}
		graph = append(graph, map[string]any{"@type": "FAQPage", "mainEntity": ents})
	}
	if len(graph) == 0 {
		return ""
	}
	raw, err := json.MarshalIndent(map[string]any{
		"@context": "https://schema.org",
		"@graph":   graph,
	}, "", "  ")
	if err != nil {
		return ""
	}
	return string(raw)
}

func originHost(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil || u.Host == "" {
		return pageURL
	}
	u.Path, u.RawQuery, u.Fragment = "", "", ""
	return u.String()
}

func splitSentences(s string) []string {
	s = collapseSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, m := range reSentenceSplit.FindAllString(s, 40) {
		m = collapseSpace(m)
		if m != "" {
			out = append(out, m)
		}
	}
	return out
}

func parseCiteLines(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if line != "" {
			out = append(out, line)
		}
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func parseFAQText(raw string) []seoFAQ {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []seoFAQ
	var q, a string
	flush := func() {
		if q != "" && a != "" {
			out = append(out, seoFAQ{Q: collapseSpace(q), A: collapseSpace(a)})
		}
		q, a = "", ""
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(strings.ToLower(line), "q:"):
			flush()
			q = strings.TrimSpace(line[2:])
		case strings.HasPrefix(strings.ToLower(line), "a:"):
			a = strings.TrimSpace(line[2:])
		case strings.HasPrefix(line, "**Q:**"):
			flush()
			q = strings.TrimSpace(strings.TrimPrefix(line, "**Q:**"))
		case strings.HasPrefix(line, "**A:**"):
			a = strings.TrimSpace(strings.TrimPrefix(line, "**A:**"))
		default:
			if q != "" && a == "" && strings.HasSuffix(q, "?") {
				a = line
			}
		}
	}
	flush()
	return out
}

func citesFromBody(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
		if len(out) >= 6 {
			break
		}
	}
	return out
}

func faqsFromBody(body string) []seoFAQ {
	if got := parseFAQText(body); len(got) > 0 {
		return got
	}
	return nil
}

func jsonFromBody(body string) string {
	i := strings.Index(body, "```json")
	if i < 0 {
		i = strings.Index(body, "{")
		if i < 0 {
			return ""
		}
		return strings.TrimSpace(body[i:])
	}
	rest := body[i+7:]
	if j := strings.Index(rest, "```"); j >= 0 {
		return strings.TrimSpace(rest[:j])
	}
	return strings.TrimSpace(rest)
}

func seedAsStructured(action string, seed seoSeed) map[string]any {
	sd := map[string]any{
		"action":         action,
		"url":            seed.URL,
		"keyword":        seed.Keyword,
		"title":          firstNonEmpty(seed.Title, seed.H1),
		"meta":           seed.Meta,
		"h1":             seed.H1,
		"save_brief":     true,
		"cannot_publish": true,
		"cannot_send":    true,
	}
	switch action {
	case "cite":
		sd["cites"] = seed.Cites
	case "faq":
		sd["faqs"] = seedFAQs(seed.FAQs)
		sd["json_ld"] = seed.JSONLD
	case "schema":
		sd["json_ld"] = seed.JSONLD
	case "onsite":
		sd["cites"] = seed.Cites
		sd["faqs"] = seedFAQs(seed.FAQs)
		sd["json_ld"] = seed.JSONLD
	}
	if len(seed.Offsite) > 0 && (action == "offsite" || action == "onsite") {
		var items []any
		for _, t := range seed.Offsite {
			items = append(items, map[string]any{"title": t, "hours": 1, "kind": "seo_offsite"})
		}
		sd["items"] = items
	}
	return sd
}

func seedFAQs(in []seoFAQ) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, f := range in {
		out = append(out, map[string]any{"q": f.Q, "a": f.A})
	}
	return out
}

func mergeSeoBrief(d *SeoBrief, seed seoSeed, form seoAskForm) {
	if d.URL == "" {
		d.URL = firstNonEmpty(form.URL, seed.URL)
	}
	if d.Keyword == "" {
		d.Keyword = firstNonEmpty(form.Keyword, seed.Keyword)
	}
	if d.Title == "" {
		d.Title = firstNonEmpty(form.Title, firstNonEmpty(seed.Title, seed.H1))
	}
	if d.Meta == "" {
		d.Meta = firstNonEmpty(form.Meta, seed.Meta)
	}
	if d.H1 == "" {
		d.H1 = firstNonEmpty(form.H1, seed.H1)
	}
	extra := walkBodyFromStructured(seedAsStructured(d.Kind, seed), form.Body)
	if d.Body == "" {
		d.Body = extra
	} else if extra != "" && !strings.Contains(d.Body, extra) && d.Kind != "onsite" {
		// keep LLM onsite prose; seed already in walk sections via siblings
	} else if extra != "" && d.Body == "" {
		d.Body = extra
	}
	if d.Kind == "cite" && len(seed.Cites) > 0 && !strings.Contains(d.Body, seed.Cites[0]) {
		d.Body = walkBodyFromStructured(map[string]any{"cites": seed.Cites}, d.Body)
	}
	if d.Kind == "faq" && len(seed.FAQs) > 0 && !strings.Contains(d.Body, seed.FAQs[0].Q) {
		d.Body = walkBodyFromStructured(map[string]any{"faqs": seedFAQs(seed.FAQs)}, d.Body)
	}
	if (d.Kind == "schema" || d.Kind == "faq") && seed.JSONLD != "" && !strings.Contains(d.Body, `"@type"`) {
		d.Body = walkBodyFromStructured(map[string]any{"json_ld": seed.JSONLD}, d.Body)
	}
}

func briefFromSeed(p Project, action string, seed seoSeed, form seoAskForm) SeoBrief {
	sd := seedAsStructured(action, applyAskForm(seed, form))
	d, err := seoBriefFromStructured(sd, p.ID)
	if err != nil {
		d = SeoBrief{ProjectID: p.ID, Kind: action, URL: seed.URL, Keyword: seed.Keyword, Title: seed.Title, Meta: seed.Meta, H1: seed.H1, Body: walkBodyFromStructured(sd, form.Body)}
	}
	return d
}

func pagesFromContext(data map[string]any) []fetchedPage {
	if data == nil {
		return nil
	}
	raw, err := json.Marshal(data["fetched_pages"])
	if err != nil {
		return nil
	}
	var pages []fetchedPage
	if json.Unmarshal(raw, &pages) != nil {
		return nil
	}
	return pages
}
