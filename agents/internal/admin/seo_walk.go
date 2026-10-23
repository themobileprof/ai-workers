package admin

import (
	"strings"

	"github.com/samuel/ai-workers/agents/internal/departments/seo"
)

const seoCompanyName = "TheMobileProf Technologies"

func seoWalkIDs() []string {
	return []string{"cite", "faq", "schema", "onsite", "offsite"}
}

func liveSeoKinds(briefs []SeoBrief) map[string]bool {
	out := map[string]bool{}
	for _, d := range briefs {
		if d.Status == "rejected" || d.Kind == "" {
			continue
		}
		out[d.Kind] = true
	}
	return out
}

func nextSeoWalkAction(briefs []SeoBrief) string {
	live := liveSeoKinds(briefs)
	for _, id := range seoWalkIDs() {
		if !live[id] {
			return id
		}
	}
	return ""
}

func nextSeoWalkLabel(after string) string {
	ids := seoWalkIDs()
	for i, id := range ids {
		if id == after && i+1 < len(ids) {
			if spec, ok := seo.Lookup(ids[i+1]); ok {
				return spec.Label
			}
			return ids[i+1]
		}
	}
	return ""
}

func resolveSeoAction(action string, briefs []SeoBrief) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" || action == "walk" || action == "next" {
		if next := nextSeoWalkAction(briefs); next != "" {
			return next
		}
		return "onsite"
	}
	if _, ok := seo.Lookup(action); ok {
		return action
	}
	if next := nextSeoWalkAction(briefs); next != "" {
		return next
	}
	return "onsite"
}

func seoWalk(briefs []SeoBrief, pageURL string) []RunStep {
	live := liveSeoKinds(briefs)
	nowSet := false
	var steps []RunStep
	for _, id := range seoWalkIDs() {
		spec, _ := seo.Lookup(id)
		st := RunStep{
			ID:     id,
			Kind:   "Walk",
			Title:  spec.Label,
			Why:    spec.Help,
			How:    seoWalkHow(id),
			Button: "Ask this step",
			Help:   "Ask SEO " + spec.Label,
			Done:   live[id],
		}
		if strings.TrimSpace(pageURL) == "" && !st.Done {
			st.Detail = "Skip until there is a public URL on Card."
		}
		if st.Done {
			st.Stamp = "done"
		} else if strings.TrimSpace(pageURL) == "" {
			st.Stamp = "later"
		} else if !nowSet {
			st.Now = true
			st.Stamp = "now"
			nowSet = true
		} else {
			st.Stamp = "later"
		}
		steps = append(steps, st)
	}
	return steps
}

func seoWalkHow(id string) []string {
	switch id {
	case "cite":
		return []string{
			"Facts below are copied from the live page. Edit a line if it is wrong.",
			"Ask this step to file the pack. Turn away if a fact is invented.",
			"Next is FAQ, written only from those facts.",
		}
	case "faq":
		return []string{
			"Q&A below is from the page (and the facts step). Visible first.",
			"Ask this step. FAQPage JSON-LD must match that block.",
			"Next is Schema for the rest of the page.",
		}
	case "schema":
		return []string{
			"JSON-LD below is what is already on the page, plus Organization if missing.",
			"Ask this step. Do not invent NAP or ratings.",
			"Next is the onsite pack Cursor implements.",
		}
	case "onsite":
		return []string{
			"Title, H1, and meta below are the designed page. Edit, then Ask.",
			"The pack weaves facts, FAQ, and schema from earlier steps.",
			"Next is offsite listings you already have.",
		}
	default:
		return []string{
			"Tasks below are listings already named on the pages.",
			"Ask this step. Delivery writes after the delay unless you Turn away.",
		}
	}
}

func latestLiveSeo(briefs []SeoBrief, kind string) SeoBrief {
	for _, d := range briefs {
		if d.Kind == kind && d.Status != "rejected" {
			return d
		}
	}
	return SeoBrief{}
}

func seoWalkContext(briefs []SeoBrief) map[string]any {
	out := map[string]any{
		"order":   seoWalkIDs(),
		"next":    nextSeoWalkAction(briefs),
		"company": seoCompanyName,
	}
	for _, id := range []string{"cite", "faq", "schema"} {
		if d := latestLiveSeo(briefs, id); d.ID > 0 {
			out[id] = map[string]any{
				"id":    d.ID,
				"title": d.Title,
				"body":  d.Body,
			}
		}
	}
	return out
}

func walkBodyFromStructured(sd map[string]any, existing string) string {
	var b strings.Builder
	if strings.TrimSpace(existing) != "" {
		b.WriteString(strings.TrimSpace(existing))
		b.WriteString("\n\n")
	}
	if cites := stringSlice(sd["cites"]); len(cites) > 0 {
		b.WriteString("## Quotable facts\n\n")
		for _, c := range cites {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			b.WriteString("- " + c + "\n")
		}
		b.WriteString("\n")
	}
	if faqs := parseSeoFAQs(sd["faqs"]); len(faqs) > 0 {
		b.WriteString("## Visible FAQ\n\n")
		for _, f := range faqs {
			b.WriteString("**Q:** " + f.Q + "\n\n")
			b.WriteString("**A:** " + f.A + "\n\n")
		}
	}
	if raw := strings.TrimSpace(firstNonEmpty(strVal(sd["json_ld"]), strVal(sd["schema"]))); raw != "" {
		b.WriteString("## Proposed JSON-LD\n\n```json\n")
		b.WriteString(raw)
		b.WriteString("\n```\n\n")
	}
	return strings.TrimSpace(b.String())
}

type seoFAQ struct {
	Q string
	A string
}

func parseSeoFAQs(raw any) []seoFAQ {
	var rows []any
	switch t := raw.(type) {
	case []any:
		rows = t
	case []map[string]any:
		for _, m := range t {
			rows = append(rows, m)
		}
	default:
		return nil
	}
	var out []seoFAQ
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		f := seoFAQ{
			Q: firstNonEmpty(strVal(m["q"]), strVal(m["question"])),
			A: firstNonEmpty(strVal(m["a"]), strVal(m["answer"])),
		}
		if f.Q != "" && f.A != "" {
			out = append(out, f)
		}
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func walkHasContent(sd map[string]any) bool {
	return len(stringSlice(sd["cites"])) > 0 || len(parseSeoFAQs(sd["faqs"])) > 0 ||
		strings.TrimSpace(firstNonEmpty(strVal(sd["json_ld"]), strVal(sd["schema"]))) != ""
}
