package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const gscSnapshotCap = 15

func gscSiteURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "sc-domain:") {
		host := strings.TrimSpace(s[len("sc-domain:"):])
		return "sc-domain:" + strings.ToLower(host)
	}
	if !strings.HasSuffix(s, "/") {
		s += "/"
	}
	return s
}

func parseGscSnapshot(body map[string]any) (map[string]any, error) {
	if body == nil {
		return nil, errors.New("empty GSC body")
	}
	if _, bad := body["error"]; bad {
		return nil, errors.New("GSC error payload")
	}
	src := strings.TrimSpace(strVal(body["source"]))
	if src != "gsc_searchanalytics" {
		return nil, errors.New("source must be gsc_searchanalytics")
	}
	site := gscSiteURL(strVal(body["site_url"]))
	if site == "" {
		return nil, errors.New("site_url required")
	}
	out := map[string]any{
		"source":      src,
		"site_url":    site,
		"range":       firstNonEmpty(strVal(body["range"]), "28d"),
		"clicks":      jsonFloat(body["clicks"]),
		"impressions": jsonFloat(body["impressions"]),
		"ctr":         jsonFloat(body["ctr"]),
		"position":    jsonFloat(body["position"]),
		"queries":     parseGscRows(body["queries"], "query"),
		"pages":       parseGscRows(body["pages"], "url"),
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

func parseGscRows(v any, key string) []map[string]any {
	list, _ := v.([]any)
	out := make([]map[string]any, 0, len(list))
	for _, row := range list {
		m, _ := row.(map[string]any)
		if m == nil {
			continue
		}
		name := strings.TrimSpace(strVal(m[key]))
		if name == "" {
			name = strings.TrimSpace(firstNonEmpty(strVal(m["query"]), strVal(m["url"])))
		}
		if name == "" {
			name = strings.TrimSpace(strVal(m["page"]))
		}
		if name == "" {
			continue
		}
		if len(name) > 500 {
			name = name[:500]
		}
		out = append(out, map[string]any{
			key:          name,
			"clicks":      jsonFloat(m["clicks"]),
			"impressions": jsonFloat(m["impressions"]),
			"ctr":         jsonFloat(m["ctr"]),
			"position":    jsonFloat(m["position"]),
		})
		if len(out) >= gscSnapshotCap {
			break
		}
	}
	return out
}

func (s *Store) UpsertSeoGSC(ctx context.Context, snap map[string]any) error {
	if s == nil || s.pool == nil {
		return errors.New("store")
	}
	site := gscSiteURL(strVal(snap["site_url"]))
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
INSERT INTO seo_gsc (id, site_url, snapshot, fetched_at)
VALUES (1, $1, $2::jsonb, $3)
ON CONFLICT (id) DO UPDATE SET site_url = EXCLUDED.site_url, snapshot = EXCLUDED.snapshot, fetched_at = EXCLUDED.fetched_at
`, site, raw, fetched)
	return err
}

func (s *Store) GetSeoGSC(ctx context.Context) (map[string]any, error) {
	if s == nil || s.pool == nil {
		return nil, nil
	}
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT snapshot FROM seo_gsc WHERE id = 1`).Scan(&raw)
	if err != nil {
		return nil, nil
	}
	var snap map[string]any
	if err := json.Unmarshal(raw, &snap); err != nil || len(snap) == 0 {
		return nil, nil
	}
	return snap, nil
}

func (s *Server) internalSeoGSCGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.store == nil {
		http.Error(w, "store", http.StatusBadGateway)
		return
	}
	snap, err := s.store.GetSeoGSC(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if snap == nil {
		writeJSON(w, map[string]any{"found": false})
		return
	}
	writeJSON(w, map[string]any{"found": true, "gsc": snap})
}

func (s *Server) internalSeoGSCPOST(w http.ResponseWriter, r *http.Request) {
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
	want := gscSiteURL(st["seo.gsc_site_url"])
	if want == "" {
		http.Error(w, "seo.gsc_site_url is empty on Defaults", http.StatusBadRequest)
		return
	}
	snap, err := parseGscSnapshot(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strVal(snap["site_url"]) != want {
		http.Error(w, "site_url does not match Defaults", http.StatusBadRequest)
		return
	}
	if err := s.store.UpsertSeoGSC(r.Context(), snap); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gsc": snap})
}
