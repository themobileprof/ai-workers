package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const ga4SnapshotCap = 15

func ga4PropertyID(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "properties/")
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func jsonFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

func parseGa4Snapshot(body map[string]any) (map[string]any, error) {
	if body == nil {
		return nil, errors.New("empty GA4 body")
	}
	if _, bad := body["error"]; bad {
		return nil, errors.New("GA4 error payload")
	}
	src := strings.TrimSpace(strVal(body["source"]))
	if src != "ga4_data_api" {
		return nil, errors.New("source must be ga4_data_api")
	}
	pid := ga4PropertyID(strVal(body["property_id"]))
	if pid == "" {
		return nil, errors.New("property_id required")
	}
	out := map[string]any{
		"source":            src,
		"property_id":       pid,
		"range":             firstNonEmpty(strVal(body["range"]), "28daysAgo:yesterday"),
		"sessions":          jsonFloat(body["sessions"]),
		"organic_sessions":  jsonFloat(body["organic_sessions"]),
		"channels":          parseGa4Channels(body["channels"]),
		"landing_pages":     parseGa4Landings(body["landing_pages"]),
	}
	if ts := strings.TrimSpace(strVal(body["fetched_at"])); ts != "" {
		if _, err := time.Parse(time.RFC3339, ts); err == nil {
			out["fetched_at"] = ts
		}
	}
	if _, ok := out["fetched_at"]; !ok {
		out["fetched_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	return out, nil
}

func parseGa4Channels(v any) []map[string]any {
	list, _ := v.([]any)
	out := make([]map[string]any, 0, len(list))
	for _, row := range list {
		m, _ := row.(map[string]any)
		if m == nil {
			continue
		}
		name := strings.TrimSpace(strVal(m["name"]))
		if name == "" {
			continue
		}
		out = append(out, map[string]any{
			"name":              name,
			"sessions":          jsonFloat(m["sessions"]),
			"engaged_sessions":  jsonFloat(m["engaged_sessions"]),
			"bounce_rate":       jsonFloat(m["bounce_rate"]),
		})
		if len(out) >= ga4SnapshotCap {
			break
		}
	}
	return out
}

func parseGa4Landings(v any) []map[string]any {
	list, _ := v.([]any)
	out := make([]map[string]any, 0, len(list))
	for _, row := range list {
		m, _ := row.(map[string]any)
		if m == nil {
			continue
		}
		path := strings.TrimSpace(strVal(m["path"]))
		if path == "" {
			continue
		}
		if len(path) > 500 {
			path = path[:500]
		}
		out = append(out, map[string]any{
			"path":             path,
			"sessions":         jsonFloat(m["sessions"]),
			"engagement_rate":  jsonFloat(m["engagement_rate"]),
		})
		if len(out) >= ga4SnapshotCap {
			break
		}
	}
	return out
}

func (s *Store) UpsertSeoGA4(ctx context.Context, snap map[string]any) error {
	if s == nil || s.pool == nil {
		return errors.New("store")
	}
	pid := ga4PropertyID(strVal(snap["property_id"]))
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	fetched := time.Now().UTC()
	if ts := strVal(snap["fetched_at"]); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			fetched = t
		}
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO seo_ga4 (id, property_id, snapshot, fetched_at)
VALUES (1, $1, $2::jsonb, $3)
ON CONFLICT (id) DO UPDATE SET property_id = EXCLUDED.property_id, snapshot = EXCLUDED.snapshot, fetched_at = EXCLUDED.fetched_at
`, pid, raw, fetched)
	return err
}

func (s *Store) GetSeoGA4(ctx context.Context) (map[string]any, error) {
	if s == nil || s.pool == nil {
		return nil, nil
	}
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT snapshot FROM seo_ga4 WHERE id = 1`).Scan(&raw)
	if err != nil {
		return nil, nil
	}
	var snap map[string]any
	if err := json.Unmarshal(raw, &snap); err != nil || len(snap) == 0 {
		return nil, nil
	}
	return snap, nil
}

func (s *Server) internalSeoGA4GET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.store == nil {
		http.Error(w, "store", http.StatusBadGateway)
		return
	}
	snap, err := s.store.GetSeoGA4(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if snap == nil {
		writeJSON(w, map[string]any{"found": false})
		return
	}
	writeJSON(w, map[string]any{"found": true, "ga4": snap})
}

func (s *Server) internalSeoGA4POST(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.store == nil {
		http.Error(w, "store", http.StatusBadGateway)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	st, err := s.store.Settings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	want := ga4PropertyID(st["seo.ga4_property_id"])
	if want == "" {
		http.Error(w, "seo.ga4_property_id is empty on Defaults", http.StatusBadRequest)
		return
	}
	snap, err := parseGa4Snapshot(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strVal(snap["property_id"]) != want {
		http.Error(w, "property_id does not match Defaults", http.StatusBadRequest)
		return
	}
	if err := s.store.UpsertSeoGA4(r.Context(), snap); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "ga4": snap})
}
