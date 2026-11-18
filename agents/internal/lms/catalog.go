// Package lms is the TheMobileProf Academy catalog and (when LMS_EMAIL is set) paid lesson bodies.
// Another student's paid/unpaid status is still not on this client.
package lms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const DefaultAPIBase = "https://api.themobileprof.com"

// Course is one published (or listed) LMS course. Prices are as the API sent them.
type Course struct {
	ID            string
	Title         string
	Format        string
	Topic         string
	Price         string
	Duration      string
	Difficulty    string
	Description   string
	Objectives    string
	Prerequisites string
	Syllabus      string
	Published     bool
}

// Path is a Mini bundle of Micros.
type Path struct {
	Title       string
	Slug        string
	Description string
	Micros      []string
}

// Catalog is the public list. Do not invent a course that is not here.
type Catalog struct {
	Fetched time.Time
	Courses []Course
	Paths   []Path
}

type Client struct {
	base        string
	http        *http.Client
	mu          sync.Mutex
	ttl         time.Duration
	got         Catalog
	email       string
	pass        string
	token       string
	tokenAt     time.Time
	lessons     map[string]cachedLessons
	workshops   map[string]cachedWorkshop
	scholarship cachedSeason
}

type cachedSeason struct {
	at   time.Time
	item Season
	ok   bool
}

type cachedWorkshop struct {
	at   time.Time
	item Workshop
	ok   bool
}

type cachedLessons struct {
	at    time.Time
	items []Lesson
}

func NewClient(base string) *Client {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("LMS_API_BASE")), "/")
	}
	if base == "" {
		base = DefaultAPIBase
	}
	base = strings.TrimSuffix(base, "/api")
	email := firstEnv("LMS_EMAIL", "LMS_BOT_EMAIL")
	pass := firstEnv("LMS_PASSWORD", "LMS_BOT_PASSWORD")
	return &Client{
		base:      base,
		http:      &http.Client{Timeout: 20 * time.Second},
		ttl:       15 * time.Minute,
		email:     email,
		pass:      pass,
		lessons:   map[string]cachedLessons{},
		workshops: map[string]cachedWorkshop{},
	}
}

func firstEnv(names ...string) string {
	for _, n := range names {
		if v := strings.TrimSpace(os.Getenv(n)); v != "" {
			return v
		}
	}
	return ""
}

func (c *Client) HasAuth() bool {
	return c != nil && c.email != "" && c.pass != ""
}

func (c *Client) Get(ctx context.Context) (Catalog, error) {
	if c == nil {
		c = NewClient("")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.got.Fetched.IsZero() && time.Since(c.got.Fetched) < c.ttl && len(c.got.Courses) > 0 {
		return c.got, nil
	}
	got, err := c.fetch(ctx)
	if err != nil {
		if len(c.got.Courses) > 0 {
			return c.got, nil
		}
		return Catalog{}, err
	}
	c.got = got
	return got, nil
}

func (c *Client) fetch(ctx context.Context) (Catalog, error) {
	out := Catalog{Fetched: time.Now().UTC()}
	courses, err := c.getJSON(ctx, "/api/courses")
	if err != nil {
		return Catalog{}, err
	}
	for _, raw := range asList(courses, "courses") {
		co := parseCourse(raw)
		if co.Title == "" {
			continue
		}
		out.Courses = append(out.Courses, co)
	}
	minis, err := c.getJSON(ctx, "/api/mini-courses")
	if err == nil {
		for _, raw := range asList(minis, "miniCourses") {
			p := parsePath(raw)
			if p.Title == "" {
				continue
			}
			out.Paths = append(out.Paths, p)
		}
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string) (map[string]any, error) {
	return c.doJSON(ctx, http.MethodGet, path, nil, "")
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, token string) (map[string]any, error) {
	var rdr io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		rdr = strings.NewReader(string(raw))
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("LMS %s HTTP %d", path, res.StatusCode)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func asList(obj map[string]any, key string) []map[string]any {
	raw, _ := obj[key].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, _ := item.(map[string]any)
		if m != nil {
			out = append(out, m)
		}
	}
	return out
}

func parseCourse(m map[string]any) Course {
	return Course{
		ID:            str(m["id"]),
		Title:         str(m["title"]),
		Format:        str(m["format"]),
		Topic:         str(m["topic"]),
		Price:         str(m["price"]),
		Duration:      str(m["duration"]),
		Difficulty:    str(m["difficulty"]),
		Description:   clip(str(m["description"]), 400),
		Objectives:    clip(str(m["objectives"]), 400),
		Prerequisites: clip(str(m["prerequisites"]), 240),
		Syllabus:      clip(syllabusText(m["syllabus"]), 400),
		Published:     boolish(m["isPublished"]),
	}
}

func parsePath(m map[string]any) Path {
	p := Path{
		Title:       str(m["title"]),
		Slug:        str(m["slug"]),
		Description: clip(str(m["description"]), 240),
	}
	for _, raw := range asList(m, "microCourses") {
		t := str(raw["title"])
		if t != "" {
			p.Micros = append(p.Micros, t)
		}
	}
	return p
}

func syllabusText(v any) string {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
			var parts []string
			if json.Unmarshal([]byte(s), &parts) == nil {
				return strings.Join(parts, "; ")
			}
			var obj []any
			if json.Unmarshal([]byte(s), &obj) == nil {
				var out []string
				for _, x := range obj {
					if s := str(x); s != "" {
						out = append(out, s)
					}
				}
				return strings.Join(out, "; ")
			}
		}
		return s
	case []any:
		var out []string
		for _, x := range t {
			if s := str(x); s != "" {
				out = append(out, s)
			}
		}
		return strings.Join(out, "; ")
	default:
		return str(v)
	}
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int(t)) {
			return fmt.Sprintf("%.2f", t)
		}
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	case json.Number:
		return t.String()
	default:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func boolish(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if n > 0 && len(s) > n {
		return strings.TrimSpace(s[:n]) + "…"
	}
	return s
}

// Match returns catalog courses whose title is mentioned in the student text.
func Match(cat Catalog, text string) []Course {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return nil
	}
	var out []Course
	seen := map[string]bool{}
	add := func(c Course) {
		if c.ID == "" || seen[c.ID] {
			return
		}
		seen[c.ID] = true
		out = append(out, c)
	}
	for _, c := range cat.Courses {
		if titleHits(text, c.Title) {
			add(c)
		}
	}
	for _, p := range cat.Paths {
		if !titleHits(text, p.Title) {
			continue
		}
		for _, micro := range p.Micros {
			for _, c := range cat.Courses {
				if strings.EqualFold(c.Title, micro) {
					add(c)
				}
			}
		}
	}
	if len(out) > 4 {
		out = out[:4]
	}
	return out
}

func titleHits(text, title string) bool {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return false
	}
	if len(title) >= 6 && strings.Contains(text, title) {
		return true
	}
	hits := 0
	for _, w := range strings.Fields(title) {
		w = strings.Trim(w, ".,:;!?")
		if len(w) < 5 {
			continue
		}
		if strings.Contains(text, w) {
			hits++
		}
	}
	return hits >= 2
}

// PromptBlock is compact catalog text for the community worker. Never invent outside it.
func PromptBlock(cat Catalog, matched []Course) string {
	var b strings.Builder
	b.WriteString("Live LMS catalog (public API, fetched " + cat.Fetched.UTC().Format(time.RFC3339) + "). You may only name these titles. Prices are as listed on the LMS — do not convert currency, do not invent a discount.\n")
	if len(matched) > 0 {
		b.WriteString("\nCourses the student appears to be asking about:\n")
		for _, c := range matched {
			writeCourse(&b, c)
		}
	}
	b.WriteString("\nMini paths:\n")
	for _, p := range cat.Paths {
		b.WriteString("- " + p.Title)
		if len(p.Micros) > 0 {
			b.WriteString(" (" + strings.Join(p.Micros, "; ") + ")")
		}
		b.WriteString("\n")
	}
	b.WriteString("\nPublished courses:\n")
	for _, c := range cat.Courses {
		if !c.Published {
			continue
		}
		line := "- " + c.Title + " [" + c.Format + "]"
		if c.Price != "" {
			line += " list_price=" + c.Price
		}
		if c.Duration != "" {
			line += " " + c.Duration
		}
		b.WriteString(line + "\n")
	}
	s := b.String()
	if len(s) > 9000 {
		return s[:8990] + "\n…"
	}
	return s
}

func writeCourse(b *strings.Builder, c Course) {
	b.WriteString("- " + c.Title + " [" + c.Format + "]")
	if c.Price != "" {
		b.WriteString(" list_price=" + c.Price)
	}
	b.WriteString("\n")
	if c.Objectives != "" {
		b.WriteString("  objectives: " + c.Objectives + "\n")
	}
	if c.Prerequisites != "" {
		b.WriteString("  prerequisites: " + c.Prerequisites + "\n")
	}
	if c.Syllabus != "" {
		b.WriteString("  syllabus: " + c.Syllabus + "\n")
	}
}

func Cards(matched []Course) []map[string]any {
	out := make([]map[string]any, 0, len(matched))
	for _, c := range matched {
		out = append(out, map[string]any{
			"id":            c.ID,
			"title":         c.Title,
			"format":        c.Format,
			"price":         c.Price,
			"duration":      c.Duration,
			"objectives":    c.Objectives,
			"prerequisites": c.Prerequisites,
			"syllabus":      c.Syllabus,
			"published":     c.Published,
		})
	}
	return out
}
