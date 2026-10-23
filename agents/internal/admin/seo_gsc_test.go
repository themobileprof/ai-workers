package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGscSiteURL(t *testing.T) {
	if got := gscSiteURL("sc-domain:TheMobileProf.com"); got != "sc-domain:themobileprof.com" {
		t.Fatalf("got %q", got)
	}
	if got := gscSiteURL("https://lms.themobileprof.com"); got != "https://lms.themobileprof.com/" {
		t.Fatalf("got %q", got)
	}
}

func TestParseGscSnapshot(t *testing.T) {
	if _, err := parseGscSnapshot(nil); err == nil {
		t.Fatal("empty")
	}
	if _, err := parseGscSnapshot(map[string]any{"error": map[string]any{"code": 403}}); err == nil {
		t.Fatal("google error")
	}
	if _, err := parseGscSnapshot(map[string]any{"source": "gsc_searchanalytics"}); err == nil {
		t.Fatal("no site")
	}
	queries := make([]any, 0, 20)
	for i := 0; i < 20; i++ {
		queries = append(queries, map[string]any{"query": "q", "clicks": i, "impressions": 10, "position": "8.1"})
	}
	snap, err := parseGscSnapshot(map[string]any{
		"source":      "gsc_searchanalytics",
		"site_url":    "sc-domain:themobileprof.com",
		"clicks":      "40",
		"impressions": 900,
		"queries":     queries,
		"pages": []any{
			map[string]any{"url": "https://lms.themobileprof.com/", "clicks": 3},
			map[string]any{"url": ""},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strVal(snap["site_url"]) != "sc-domain:themobileprof.com" || jsonFloat(snap["clicks"]) != 40 || jsonFloat(snap["impressions"]) != 900 {
		t.Fatalf("%+v", snap)
	}
	qs, _ := snap["queries"].([]map[string]any)
	if len(qs) != gscSnapshotCap {
		t.Fatalf("cap %d", len(qs))
	}
	pages, _ := snap["pages"].([]map[string]any)
	if len(pages) != 1 || strVal(pages[0]["url"]) == "" {
		t.Fatalf("%+v", pages)
	}
}

func TestInternalSeoGSCUnauthorized(t *testing.T) {
	s, err := New(nil, "secret")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/v1/seo/gsc", strings.NewReader(`{"source":"gsc_searchanalytics"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
	ok := httptest.NewRequest(http.MethodPost, "/internal/v1/seo/gsc", strings.NewReader(`{"source":"gsc_searchanalytics","site_url":"sc-domain:x.com"}`))
	ok.Header.Set("X-Internal-Token", "secret")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, ok)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("nil store %d", rec.Code)
	}
}
