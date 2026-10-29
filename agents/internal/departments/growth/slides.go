package growth

import (
	"fmt"
	"strings"
)

// DeckSlide is one Kawasaki slide in a live session.
type DeckSlide struct {
	N      int    `json:"n"`
	Title  string `json:"title"`
	Copy   string `json:"copy"`
	Notes  string `json:"notes"`
	Status string `json:"status"` // draft | agreed | skipped
}

func EmptySlides() []DeckSlide {
	out := make([]DeckSlide, 0, 10)
	for _, s := range KawasakiSlides() {
		out = append(out, DeckSlide{N: s.N, Title: s.Title, Status: "draft"})
	}
	return out
}

func ParseSlides(raw any) []DeckSlide {
	var rows []any
	switch t := raw.(type) {
	case []any:
		rows = t
	case []map[string]any:
		for _, m := range t {
			rows = append(rows, m)
		}
	case []DeckSlide:
		return MergeSlides(EmptySlides(), t)
	default:
		return nil
	}
	var updates []DeckSlide
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		s := DeckSlide{
			N:      intVal(m["n"]),
			Title:  str(m["title"]),
			Copy:   firstStr(m["copy"], m["body"]),
			Notes:  str(m["notes"]),
			Status: str(m["status"]),
		}
		if s.N < 1 || s.N > 10 {
			continue
		}
		if s.Title == "" {
			s.Title = KawasakiSlides()[s.N-1].Title
		}
		updates = append(updates, s)
	}
	if len(updates) == 0 {
		return nil
	}
	return updates
}

func MergeSlides(base, updates []DeckSlide) []DeckSlide {
	out := EmptySlides()
	apply := func(u DeckSlide) {
		if u.N < 1 || u.N > 10 {
			return
		}
		i := u.N - 1
		if u.Title != "" {
			out[i].Title = u.Title
		}
		if strings.TrimSpace(u.Copy) != "" {
			out[i].Copy = u.Copy
		}
		if strings.TrimSpace(u.Notes) != "" {
			out[i].Notes = u.Notes
		}
		switch u.Status {
		case "agreed", "skipped":
			out[i].Status = u.Status
		case "draft":
			if out[i].Status == "" {
				out[i].Status = "draft"
			}
		}
	}
	if len(base) == 10 {
		copy(out, base)
		for i := range out {
			if out[i].Title == "" {
				out[i].Title = KawasakiSlides()[i].Title
			}
			if out[i].N == 0 {
				out[i].N = i + 1
			}
			if out[i].Status == "" {
				out[i].Status = "draft"
			}
		}
	} else {
		for _, u := range base {
			apply(u)
		}
	}
	for _, u := range updates {
		apply(u)
	}
	return out
}

func (s DeckSlide) StampClass() string {
	switch s.Status {
	case "agreed":
		return "ok"
	case "skipped":
		return "dark"
	default:
		return "brass"
	}
}

func (s DeckSlide) StampLabel() string {
	switch s.Status {
	case "agreed":
		return "agreed"
	case "skipped":
		return "skipped"
	default:
		return "draft"
	}
}

func SlidesMarkdown(slides []DeckSlide) string {
	if len(slides) == 0 {
		slides = EmptySlides()
	}
	var b strings.Builder
	for _, s := range slides {
		b.WriteString(fmt.Sprintf("## %d. %s\n\n", s.N, s.Title))
		copy := strings.TrimSpace(s.Copy)
		if copy == "" {
			copy = "[TO BE COMPLETED]"
		}
		b.WriteString(copy + "\n\n")
		if notes := strings.TrimSpace(s.Notes); notes != "" {
			b.WriteString("_Speaker notes:_ " + notes + "\n\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func ClampSlide(n int) int {
	if n < 1 {
		return 1
	}
	if n > 10 {
		return 10
	}
	return n
}

func firstStr(vals ...any) string {
	for _, v := range vals {
		if s := strings.TrimSpace(str(v)); s != "" {
			return s
		}
	}
	return ""
}

func intVal(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n := 0
		for _, r := range t {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		return n
	default:
		return 0
	}
}
