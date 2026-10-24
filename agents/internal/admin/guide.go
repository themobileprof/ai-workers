package admin

import (
	"context"
	"embed"
	"encoding/json"
	"html"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/samuel/ai-workers/agents/internal/llm"
)

//go:embed guide/*.md
var guideFS embed.FS

type GuideAsk struct {
	ID        int64
	UserID    int64
	UserName  string
	Question  string
	Answer    string
	Page      string
	Unsure    bool
	CreatedAt time.Time
}

type GuideReply struct {
	Answer string      `json:"answer"`
	Hrefs  []string    `json:"hrefs"`
	Unsure bool        `json:"unsure"`
	Source string      `json:"source,omitempty"`
	Draft  *GuideDraft `json:"draft,omitempty"`
}

type guideInput struct {
	User     User
	Question string
	Page     string
	History  []llm.Message
	Prior    []GuideAsk
	Draft    *GuideDraft
	World    formWorld
}

type faqEntry struct {
	Q string
	A string
}

var (
	htmlBlockRe = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlTagRe   = regexp.MustCompile(`<[^>]+>`)
	spaceRe     = regexp.MustCompile(`\s+`)
	tokenRe     = regexp.MustCompile(`[a-z0-9]+`)
)

func mustGuideFiles() error {
	for _, name := range []string{"guide/handbook.md", "guide/faq.md"} {
		if _, err := guideFS.ReadFile(name); err != nil {
			return err
		}
	}
	return nil
}

func readGuideFile(name string) string {
	raw, err := guideFS.ReadFile("guide/" + name)
	if err != nil {
		return ""
	}
	return string(raw)
}

func parseFAQ(raw string) []faqEntry {
	var out []faqEntry
	for _, block := range strings.Split(raw, "\n## ") {
		block = strings.TrimSpace(block)
		if block == "" || strings.HasPrefix(block, "# ") {
			continue
		}
		q, a, ok := strings.Cut(block, "\n")
		if !ok {
			continue
		}
		q = strings.TrimSpace(strings.TrimPrefix(q, "## "))
		a = strings.TrimSpace(a)
		if q == "" || a == "" {
			continue
		}
		out = append(out, faqEntry{Q: q, A: a})
	}
	return out
}

func stripHTML(s string) string {
	s = htmlBlockRe.ReplaceAllString(s, " ")
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

func tokens(s string) []string {
	s = strings.ToLower(s)
	raw := tokenRe.FindAllString(s, -1)
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, w := range raw {
		if len(w) < 2 || stopWord[w] {
			continue
		}
		if len(w) > 4 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
			w = strings.TrimSuffix(w, "s")
		}
		if seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}

var stopWord = map[string]bool{
	"the": true, "and": true, "for": true, "you": true, "your": true, "this": true,
	"that": true, "with": true, "from": true, "are": true, "was": true, "how": true,
	"do": true, "does": true, "what": true, "why": true, "can": true, "i": true,
	"we": true, "to": true, "of": true, "in": true, "on": true, "or": true, "is": true,
	"it": true, "an": true, "a": true, "be": true, "my": true, "me": true, "at": true,
}

func overlap(query, doc []string) int {
	have := map[string]bool{}
	for _, w := range doc {
		have[w] = true
	}
	n := 0
	for _, w := range query {
		if have[w] {
			n++
		}
	}
	return n
}

func bestFAQ(question string, entries []faqEntry) (faqEntry, int) {
	qtok := tokens(question)
	best := faqEntry{}
	bestN := 0
	for _, e := range entries {
		n := overlap(qtok, tokens(e.Q+" "+e.A))
		// Short asks like "what is stamp" lose stop-words; if every leftover
		// token is in the question title, treat it as a hit.
		if len(qtok) > 0 && overlap(qtok, tokens(e.Q)) == len(qtok) {
			n += 2
		}
		if n > bestN {
			bestN = n
			best = e
		}
	}
	return best, bestN
}

type playbookHit struct {
	Meta DocMeta
	Body string
	N    int
}

func pickPlaybooks(question, page string, limit int) []playbookHit {
	qtok := tokens(question + " " + page)
	var hits []playbookHit
	for _, meta := range docCatalog {
		raw, err := playbookFS.ReadFile("playbook/" + meta.Slug + ".html")
		if err != nil {
			continue
		}
		body := stripHTML(string(raw))
		blob := meta.Title + " " + meta.Summary + " " + strings.Join(meta.Prefixes, " ") + " " + body
		n := overlap(qtok, tokens(blob))
		if slugFromPage(page) == meta.Slug {
			n += 4
		}
		if n == 0 {
			continue
		}
		if len(body) > 2800 {
			body = body[:2800] + "…"
		}
		hits = append(hits, playbookHit{Meta: meta, Body: body, N: n})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].N > hits[j].N })
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func slugFromPage(page string) string {
	page = strings.ToLower(strings.TrimSpace(page))
	for _, p := range []string{"/admin/docs/", "/admin/desks/", "/admin/"} {
		if strings.HasPrefix(page, p) {
			rest := strings.TrimPrefix(page, p)
			rest = strings.Trim(rest, "/")
			if i := strings.IndexAny(rest, "/?#"); i >= 0 {
				rest = rest[:i]
			}
			return rest
		}
	}
	return strings.Trim(page, "/")
}

func runtimeDeskMap() string {
	var b strings.Builder
	b.WriteString("Workers (Ask files a job; Accept is your yes):\n")
	for _, d := range AllDesks() {
		b.WriteString("- ")
		b.WriteString(d.Title)
		b.WriteString(" prefix ")
		b.WriteString(d.Prefix)
		b.WriteString(" desk ")
		b.WriteString(d.Path())
		if d.BookPath != "" {
			b.WriteString(" book ")
			b.WriteString(d.BookPath)
		}
		b.WriteString(" — ")
		b.WriteString(d.Help)
		b.WriteString("\n")
	}
	b.WriteString("\nStage caps (open tasks): parked/killed 0, idea 1, testing 3, selling 5, fundable 4.\n")
	b.WriteString("Docs: /admin/docs then /admin/docs/{slug}. Player sheets first (you, owner, bdm, project-face, field, handler). Owner/BDM: Help? log is Docs → Asked at /admin/guide.\n")
	b.WriteString("Office menu (owner/BDM books): CRM, Files, Grants, Defaults. n8n is under the signed-in name, not Office.\n")
	return b.String()
}

func buildGuideContext(in guideInput) (string, []string) {
	var b strings.Builder
	var hrefs []string
	seen := map[string]bool{}
	addHref := func(h string) {
		h = strings.TrimSpace(h)
		if !strings.HasPrefix(h, "/admin/") || seen[h] {
			return
		}
		seen[h] = true
		hrefs = append(hrefs, h)
	}

	b.WriteString("# Person\n")
	b.WriteString("Name: ")
	b.WriteString(in.User.Name)
	b.WriteString("\nRole: ")
	b.WriteString(in.User.Role)
	b.WriteString("\nDesks: ")
	b.WriteString(strings.Join(in.User.Desks, ", "))
	b.WriteString("\nPage they are on: ")
	b.WriteString(in.Page)
	b.WriteString("\n\n# Fillable forms\n")
	b.WriteString(catalogPrompt(allowedCatalog(in.User, in.Page)))
	if wp := worldPrompt(in.World); wp != "" {
		b.WriteString("\n")
		b.WriteString(wp)
	}
	b.WriteString("\n# Open form\n")
	b.WriteString(draftPrompt(in.Draft))
	b.WriteString("\n\n# Map\n")
	b.WriteString(runtimeDeskMap())
	b.WriteString("\n# Handbook\n")
	b.WriteString(readGuideFile("handbook.md"))
	b.WriteString("\n\n# FAQ\n")
	faqRaw := readGuideFile("faq.md")
	b.WriteString(faqRaw)
	faqHit, n := bestFAQ(in.Question, parseFAQ(faqRaw))
	if n >= 2 {
		b.WriteString("\n\n# Closest FAQ\nQ: ")
		b.WriteString(faqHit.Q)
		b.WriteString("\nA: ")
		b.WriteString(faqHit.A)
	}
	hits := pickPlaybooks(in.Question, in.Page, 3)
	if len(hits) > 0 {
		b.WriteString("\n\n# Playbook excerpts\n")
		for _, h := range hits {
			b.WriteString("## ")
			b.WriteString(h.Meta.Title)
			b.WriteString(" (")
			b.WriteString(h.Meta.Href())
			b.WriteString(")\n")
			b.WriteString(h.Body)
			b.WriteString("\n\n")
			addHref(h.Meta.Href())
		}
	}
	if len(in.Prior) > 0 {
		b.WriteString("\n# Recent answered questions in this office\n")
		for _, a := range in.Prior {
			if strings.TrimSpace(a.Answer) == "" {
				continue
			}
			b.WriteString("Q: ")
			b.WriteString(a.Question)
			b.WriteString("\nA: ")
			b.WriteString(a.Answer)
			b.WriteString("\n\n")
		}
	}
	pack := b.String()
	if len(pack) > 24000 {
		pack = pack[:24000] + "\n…"
	}
	return pack, hrefs
}

func parseGuideLLM(raw string) guideLLM {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			var out guideLLM
			if json.Unmarshal([]byte(s[i:j+1]), &out) == nil {
				out.Hrefs = cleanHrefs(out.Hrefs)
				out.Recipe = strings.TrimSpace(out.Recipe)
				return out
			}
		}
	}
	return guideLLM{Answer: s}
}

func parseGuideModel(raw string) GuideReply {
	m := parseGuideLLM(raw)
	if strings.TrimSpace(m.Answer) == "" {
		return GuideReply{Answer: strings.TrimSpace(raw)}
	}
	return GuideReply{Answer: m.Answer, Hrefs: m.Hrefs, Unsure: m.Unsure}
}

func cleanHrefs(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, h := range in {
		h = strings.TrimSpace(h)
		if !strings.HasPrefix(h, "/admin/") {
			continue
		}
		for i, r := range h {
			if unicode.IsSpace(r) || r == ')' || r == '"' || r == '\'' {
				h = h[:i]
				break
			}
		}
		if seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, h)
	}
	return out
}

func faqFallback(question string) GuideReply {
	hit, n := bestFAQ(question, parseFAQ(readGuideFile("faq.md")))
	if n < 2 {
		return GuideReply{
			Answer: "I do not have a firm answer in the handbook. Open Docs (`/admin/docs`) — start at How you fit — or ask the owner. This question is stored so the guide can learn it.",
			Unsure: true,
			Hrefs:  []string{"/admin/docs", "/admin/docs/you"},
			Source: "none",
		}
	}
	return GuideReply{Answer: hit.A, Hrefs: []string{"/admin/docs"}, Source: "faq"}
}

func (s *Server) answerGuide(ctx context.Context, in guideInput) GuideReply {
	if s != nil && s.store != nil && len(in.Prior) == 0 {
		in.Prior, _ = s.store.RecentGuideQA(ctx, 12)
	}
	in.World = s.loadFormWorld(ctx)
	pack, extraHrefs := buildGuideContext(in)
	if s == nil || s.guide == nil {
		if wiz, ok := continueDraft(in, in.World); ok {
			wiz.Hrefs = cleanHrefs(append(wiz.Hrefs, extraHrefs...))
			return wiz
		}
		rep := faqFallback(in.Question)
		rep.Hrefs = cleanHrefs(append(rep.Hrefs, extraHrefs...))
		return rep
	}
	sys := `You are the desk guide for TheMobileProf Technologies, one Lagos company. Signed-in people ask what to do and how on this website.
Answer only from the supplied handbook, FAQ, map, playbook excerpts, and fillable-form catalog. If it is not there, set unsure true and say so — do not invent stages, vendors, screens, or recipe ids.
If the user wants to accomplish a desk form (add a date, task, person, Ask a worker, …), set recipe to that catalog id and put extracted field values in fields. Dates must be YYYY-MM-DD. Compare their intent to the catalog — do not use keyword lists of your own outside that catalog. New forms appear in the catalog; you pick by meaning.
If they are continuing an open form, keep that recipe and fill more fields. If they want a different form, switch recipe. If they cancel, set cancel true and recipe "".
If they only want an explanation, leave recipe "".
Never Accept for the user. Never send email. Never publish. Never hire. Never pick a form that Accepts, rejects, deletes, or marks Done. /approve on Telegram only sends a pending info@ draft. Recruiting is closed. Stamp is an old word for Accept (the Decide tab). Submit in Help? posts the page form; you only collect fields. The brass button is Help? (it used to say How?).
Point at /admin/... paths. Keep the answer short. Nigerian English is fine. No emoji.
Reply with a JSON object only: {"answer":"...","unsure":false,"hrefs":["/admin/docs/delivery"],"recipe":"","fields":{},"cancel":false}`
	msgs := make([]llm.Message, 0, len(in.History)+2)
	for _, m := range in.History {
		role := m.Role
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if len(content) > 1500 {
			content = content[:1500]
		}
		msgs = append(msgs, llm.Message{Role: role, Content: content})
	}
	if len(msgs) > 8 {
		msgs = msgs[len(msgs)-8:]
	}
	msgs = append(msgs, llm.Message{Role: "user", Content: "Knowledge:\n" + pack + "\n\nQuestion:\n" + in.Question})
	raw, err := s.guide.Complete(ctx, llm.Request{System: sys, Messages: msgs, Plain: true})
	if err != nil || strings.TrimSpace(raw) == "" {
		if wiz, ok := continueDraft(in, in.World); ok {
			wiz.Hrefs = cleanHrefs(append(wiz.Hrefs, extraHrefs...))
			return wiz
		}
		rep := faqFallback(in.Question)
		rep.Hrefs = cleanHrefs(append(rep.Hrefs, extraHrefs...))
		return rep
	}
	model := parseGuideLLM(raw)
	if wiz, ok := applyLLMForm(in, model, in.World); ok {
		wiz.Hrefs = cleanHrefs(append(wiz.Hrefs, extraHrefs...))
		if wiz.Source == "" {
			wiz.Source = "llm"
		}
		return wiz
	}
	rep := GuideReply{Answer: model.Answer, Hrefs: cleanHrefs(append(model.Hrefs, extraHrefs...)), Unsure: model.Unsure, Source: "llm"}
	if strings.TrimSpace(rep.Answer) == "" {
		rep = faqFallback(in.Question)
		rep.Hrefs = cleanHrefs(append(rep.Hrefs, extraHrefs...))
	}
	return rep
}
