package lms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Lesson is one LMS lesson body. Admin login sees paid courses; do not treat this as a student's receipt.
type Lesson struct {
	ID          string
	CourseID    string
	Title       string
	Description string
	Content     string
	VideoURL    string
	Order       int
	DurationMin int
	Published   bool
	Workshop    Workshop
}

func (c *Client) login(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("LMS client is nil")
	}
	if c.token != "" && time.Since(c.tokenAt) < 6*time.Hour {
		return nil
	}
	if c.email == "" || c.pass == "" {
		return fmt.Errorf("LMS_EMAIL and LMS_PASSWORD are not set")
	}
	obj, err := c.doJSON(ctx, http.MethodPost, "/api/auth/admin/login", map[string]string{
		"email":    c.email,
		"password": c.pass,
	}, "")
	if err != nil {
		return err
	}
	tok := str(obj["token"])
	if tok == "" {
		if u, _ := obj["user"].(map[string]any); u != nil {
			tok = str(u["token"])
		}
	}
	if tok == "" {
		return fmt.Errorf("LMS admin login returned no token")
	}
	c.token = tok
	c.tokenAt = time.Now()
	return nil
}

func (c *Client) authJSON(ctx context.Context, path string) (map[string]any, error) {
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	obj, err := c.doJSON(ctx, http.MethodGet, path, nil, c.token)
	if err != nil && strings.Contains(err.Error(), "HTTP 401") {
		c.token = ""
		if err := c.login(ctx); err != nil {
			return nil, err
		}
		obj, err = c.doJSON(ctx, http.MethodGet, path, nil, c.token)
	}
	return obj, err
}

// Lessons returns paid and free lesson bodies for one course. Empty if LMS_EMAIL is unset.
func (c *Client) Lessons(ctx context.Context, courseID string) ([]Lesson, error) {
	if c == nil {
		c = NewClient("")
	}
	courseID = strings.TrimSpace(courseID)
	if courseID == "" {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.lessons[courseID]; ok && time.Since(cached.at) < c.ttl && len(cached.items) > 0 {
		return cached.items, nil
	}
	if !c.HasAuth() {
		return nil, nil
	}
	obj, err := c.authJSON(ctx, "/api/admin/courses/"+courseID+"/lessons?limit=50")
	if err != nil {
		return nil, err
	}
	var out []Lesson
	for _, raw := range asList(obj, "lessons") {
		le := parseLesson(raw)
		if le.ID == "" && le.Title == "" {
			continue
		}
		if le.CourseID == "" {
			le.CourseID = courseID
		}
		out = append(out, le)
	}
	c.lessons[courseID] = cachedLessons{at: time.Now(), items: out}
	return out, nil
}

func parseLesson(m map[string]any) Lesson {
	return Lesson{
		ID:          firstStr(m, "id"),
		CourseID:    firstStr(m, "courseId", "course_id"),
		Title:       firstStr(m, "title"),
		Description: clip(plain(firstStr(m, "description")), 400),
		Content:     clip(plain(firstStr(m, "content")), 4000),
		VideoURL:    firstStr(m, "videoUrl", "video_url"),
		Order:       intish(m["orderIndex"], m["order_index"]),
		DurationMin: intish(m["durationMinutes"], m["duration_minutes"]),
		Published:   boolish(firstAny(m, "isPublished", "is_published")),
	}
}

func firstStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := str(m[k]); s != "" {
			return s
		}
	}
	return ""
}

func firstAny(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func intish(vs ...any) int {
	for _, v := range vs {
		switch t := v.(type) {
		case int:
			return t
		case float64:
			return int(t)
		case json.Number:
			n, _ := t.Int64()
			return int(n)
		case string:
			n, _ := strconv.Atoi(strings.TrimSpace(t))
			if n != 0 || strings.TrimSpace(t) == "0" {
				return n
			}
		}
	}
	return 0
}

var tagRe = regexp.MustCompile(`(?s)<[^>]*>`)

func plain(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "</li>", "\n")
	s = tagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return strings.TrimSpace(s)
}

var lessonNumRe = regexp.MustCompile(`(?i)\blesson\s*#?\s*(\d+)\b`)

// MatchLessons picks lessons the student named (number or title). Empty means outline-only.
func MatchLessons(lessons []Lesson, text string) []Lesson {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" || len(lessons) == 0 {
		return nil
	}
	var out []Lesson
	seen := map[string]bool{}
	add := func(l Lesson) {
		key := l.ID
		if key == "" {
			key = l.Title
		}
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, l)
	}
	if m := lessonNumRe.FindStringSubmatch(text); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		for _, l := range lessons {
			if l.Order == n {
				add(l)
			}
		}
	}
	for _, l := range lessons {
		if titleHits(text, l.Title) {
			add(l)
		}
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

// LessonBlock is compact lesson text for the community worker.
func LessonBlock(courseTitle string, lessons, focus []Lesson) string {
	if len(lessons) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("LMS lesson bodies for " + courseTitle + " (admin catalog, includes paid lessons). Answer study questions from this. Do not paste the whole lesson. Do not invent quiz answers. Do not paste a video URL that is not written here.\n")
	b.WriteString("Outline:\n")
	for _, l := range lessons {
		line := fmt.Sprintf("- lesson %d: %s", l.Order, l.Title)
		if !l.Published {
			line += " (unpublished)"
		}
		b.WriteString(line + "\n")
	}
	use := focus
	if len(use) == 0 && len(lessons) <= 2 {
		use = lessons
	}
	if len(use) > 0 {
		b.WriteString("\nLesson text:\n")
		for _, l := range use {
			b.WriteString(fmt.Sprintf("### lesson %d: %s\n", l.Order, l.Title))
			if l.Description != "" {
				b.WriteString(l.Description + "\n")
			}
			if l.Content != "" {
				b.WriteString(clip(l.Content, 1800) + "\n")
			}
			if v := publicVideo(l.VideoURL); v != "" {
				b.WriteString("video: " + v + "\n")
			} else if l.VideoURL != "" {
				b.WriteString("video: on the LMS lesson page (do not invent a public link)\n")
			}
			if l.Workshop.HasSteps() {
				b.WriteString(fmt.Sprintf("workshop: %d steps (use workshop_live; do not lecture the concept)\n", len(l.Workshop.Steps)))
			}
			b.WriteString("\n")
		}
	}
	s := b.String()
	if len(s) > 6000 {
		return s[:5990] + "\n…"
	}
	return s
}

func publicVideo(u string) string {
	u = strings.TrimSpace(u)
	low := strings.ToLower(u)
	if strings.Contains(low, "youtube.com") || strings.Contains(low, "youtu.be") || strings.Contains(low, "vimeo.com") {
		return u
	}
	return ""
}

func LessonCards(lessons, focus []Lesson) []map[string]any {
	want := map[string]bool{}
	for _, l := range focus {
		want[l.ID] = true
	}
	out := make([]map[string]any, 0, len(lessons))
	for _, l := range lessons {
		row := map[string]any{
			"id":       l.ID,
			"title":    l.Title,
			"order":    l.Order,
			"duration": l.DurationMin,
		}
		if l.Workshop.HasSteps() {
			row["workshop"] = true
			row["workshop_steps"] = len(l.Workshop.Steps)
		}
		if want[l.ID] || len(focus) == 0 && len(lessons) <= 2 {
			row["description"] = l.Description
			row["body"] = clip(l.Content, 1200)
		}
		out = append(out, row)
	}
	return out
}
