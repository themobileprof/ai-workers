package admin

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"
)

// keywordPhrase is one phrase taken from this bet’s own pages.
// Visible copy (body, H1) is what the designer and user already saw.
// Title/meta are chrome and may never have been designed. Count is on-site
// visible occurrences — never search volume.
type keywordPhrase struct {
	Phrase    string   `json:"phrase"`
	Count     int      `json:"count"`
	BodyCount int      `json:"body_count"`
	InBody    bool     `json:"in_body"`
	InTitle   bool     `json:"in_title"`
	InH1      bool     `json:"in_h1"`
	InMeta    bool     `json:"in_meta"`
	Pages     []string `json:"pages"`
	BodyPages []string `json:"body_pages,omitempty"`
	Role      string   `json:"role"`
}

type pageKeywordRow struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	BodyPrimary string `json:"body_primary"`
	TitleClaim  string `json:"title_claim"`
	Mismatch    bool   `json:"title_lags_body"`
}

var seoStop = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true, "if": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true, "by": true,
	"from": true, "with": true, "without": true, "as": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true, "it": true,
	"this": true, "that": true, "these": true, "those": true, "we": true, "you": true,
	"your": true, "our": true, "their": true, "they": true, "them": true, "us": true,
	"not": true, "no": true, "yes": true, "can": true, "will": true, "just": true,
	"about": true, "into": true, "over": true, "after": true, "before": true, "than": true,
	"then": true, "so": true, "too": true, "very": true, "more": true, "most": true,
	"other": true, "some": true, "any": true, "all": true, "each": true, "here": true,
	"there": true, "when": true, "where": true, "what": true, "which": true, "who": true,
	"how": true, "why": true, "also": true, "only": true, "own": true, "same": true,
	"such": true, "via": true, "per": true, "click": true, "learn": true, "read": true,
	"home": true, "page": true, "privacy": true, "policy": true, "terms": true, "cookie": true,
	"cookies": true, "login": true, "signup": true, "sign": true, "contact": true,
	"menu": true, "skip": true, "content": true, "next": true, "back": true, "submit": true,
	"welcome": true, "hello": true, "please": true, "using": true, "use": true, "used": true,
}

func keywordInventory(pages []fetchedPage) []keywordPhrase {
	type acc struct {
		phrase  string
		body    int
		inTitle map[string]bool
		inH1    map[string]bool
		inMeta  map[string]bool
		bodyOn  map[string]bool
		pages   map[string]bool
	}
	got := map[string]*acc{}
	touch := func(raw, page, src string) *acc {
		phrase := collapseSpace(strings.ToLower(raw))
		if !usableSEOPhrase(phrase) {
			return nil
		}
		a := got[phrase]
		if a == nil {
			a = &acc{
				phrase:  phrase,
				inTitle: map[string]bool{},
				inH1:    map[string]bool{},
				inMeta:  map[string]bool{},
				bodyOn:  map[string]bool{},
				pages:   map[string]bool{},
			}
			got[phrase] = a
		}
		a.pages[page] = true
		switch src {
		case "title":
			a.inTitle[page] = true
		case "h1":
			a.inH1[page] = true
		case "meta":
			a.inMeta[page] = true
		case "body":
			a.body++
			a.bodyOn[page] = true
		}
		return a
	}
	for _, p := range pages {
		page := strings.TrimSpace(p.URL)
		if page == "" {
			continue
		}
		for _, g := range phraseGrams(p.Title, 8) {
			touch(g, page, "title")
		}
		for _, g := range phraseGrams(p.H1, 8) {
			touch(g, page, "h1")
		}
		for _, g := range phraseGrams(p.Meta, 8) {
			touch(g, page, "meta")
		}
		for _, g := range phraseGrams(p.Text, 5) {
			touch(g, page, "body")
		}
	}
	out := make([]keywordPhrase, 0, len(got))
	for _, a := range got {
		words := strings.Fields(a.phrase)
		claimed := len(a.inTitle)+len(a.inH1)+len(a.inMeta) > 0
		switch {
		case a.body >= 2:
		case a.body >= 1 && len(words) >= 3:
		case a.body == 0 && claimed:
		default:
			continue
		}
		k := keywordPhrase{
			Phrase:    a.phrase,
			Count:     a.body,
			BodyCount: a.body,
			InBody:    a.body > 0,
			InTitle:   len(a.inTitle) > 0,
			InH1:      len(a.inH1) > 0,
			InMeta:    len(a.inMeta) > 0,
			Pages:     mapKeys(a.pages),
			BodyPages: mapKeys(a.bodyOn),
		}
		k.Role = phraseRole(k)
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BodyCount != out[j].BodyCount {
			return out[i].BodyCount > out[j].BodyCount
		}
		ri, rj := roleRank(out[i].Role), roleRank(out[j].Role)
		if ri != rj {
			return ri < rj
		}
		return out[i].Phrase < out[j].Phrase
	})
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func phraseRole(k keywordPhrase) string {
	if k.InBody && len(k.BodyPages) >= 2 && k.BodyCount >= 2 {
		return "cannibal"
	}
	if k.InBody && !k.InTitle {
		return "gap"
	}
	if k.InBody && k.InTitle {
		return "aligned"
	}
	return "brochure"
}

func roleRank(role string) int {
	switch role {
	case "gap":
		return 0
	case "aligned":
		return 1
	case "cannibal":
		return 2
	default:
		return 3
	}
}

func keywordMap(pages []fetchedPage, inv []keywordPhrase) []pageKeywordRow {
	out := make([]pageKeywordRow, 0, len(pages))
	for _, p := range pages {
		row := pageKeywordRow{URL: p.URL, Title: p.Title, TitleClaim: collapseSpace(strings.ToLower(p.Title))}
		best := ""
		bestN := -1
		bestRole := 99
		for _, k := range inv {
			if !k.InBody || !pageIn(k.BodyPages, p.URL) {
				continue
			}
			r := roleRank(k.Role)
			if k.BodyCount > bestN || (k.BodyCount == bestN && r < bestRole) {
				best = k.Phrase
				bestN = k.BodyCount
				bestRole = r
			}
		}
		row.BodyPrimary = best
		if best != "" && row.TitleClaim != "" && !strings.Contains(row.TitleClaim, best) && !strings.Contains(best, row.TitleClaim) {
			row.Mismatch = true
		}
		out = append(out, row)
	}
	return out
}

func pageIn(pages []string, u string) bool {
	for _, p := range pages {
		if p == u {
			return true
		}
	}
	return false
}

func phraseGrams(s string, maxWords int) []string {
	words := seoWords(s)
	if len(words) == 0 {
		return nil
	}
	var out []string
	if n := len(words); n >= 2 && n <= maxWords {
		out = append(out, strings.Join(words, " "))
	}
	top := 5
	if maxWords < top {
		top = maxWords
	}
	for n := 2; n <= top && n <= len(words); n++ {
		for i := 0; i+n <= len(words); i++ {
			out = append(out, strings.Join(words[i:i+n], " "))
		}
	}
	return out
}

func seoWords(s string) []string {
	var b strings.Builder
	var out []string
	flush := func() {
		w := strings.ToLower(strings.TrimSpace(b.String()))
		b.Reset()
		if w == "" || seoStop[w] || len(w) < 3 {
			return
		}
		out = append(out, w)
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()
	return out
}

func usableSEOPhrase(phrase string) bool {
	if phrase == "" || len(phrase) > 80 {
		return false
	}
	words := strings.Fields(phrase)
	if len(words) < 2 || len(words) > 8 {
		return false
	}
	for _, w := range words {
		if !seoStop[w] {
			return true
		}
	}
	return false
}

func mapKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func inventoryFromContext(data map[string]any) []keywordPhrase {
	if data == nil {
		return nil
	}
	raw, _ := json.Marshal(data["keyword_inventory"])
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var out []keywordPhrase
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}
