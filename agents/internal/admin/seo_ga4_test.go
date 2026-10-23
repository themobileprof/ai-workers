package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGa4PropertyID(t *testing.T) {
	if got := ga4PropertyID("properties/123456789"); got != "123456789" {
		t.Fatalf("got %q", got)
	}
	if got := ga4PropertyID("  123 456 "); got != "123456" {
		t.Fatalf("got %q", got)
	}
}

func TestParseGa4Snapshot(t *testing.T) {
	if _, err := parseGa4Snapshot(nil); err == nil {
		t.Fatal("empty")
	}
	if _, err := parseGa4Snapshot(map[string]any{"error": map[string]any{"code": 403}}); err == nil {
		t.Fatal("google error")
	}
	if _, err := parseGa4Snapshot(map[string]any{"source": "ga4_data_api"}); err == nil {
		t.Fatal("no property")
	}
	pages := make([]any, 0, 20)
	for i := 0; i < 20; i++ {
		pages = append(pages, map[string]any{"path": "/p", "sessions": i, "engagement_rate": "0.2"})
	}
	snap, err := parseGa4Snapshot(map[string]any{
		"source":           "ga4_data_api",
		"property_id":      "properties/987",
		"sessions":         "40",
		"organic_sessions": 12,
		"channels": []any{
			map[string]any{"name": "Organic Search", "sessions": 12},
			map[string]any{"name": ""},
		},
		"landing_pages": pages,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strVal(snap["property_id"]) != "987" || jsonFloat(snap["sessions"]) != 40 || jsonFloat(snap["organic_sessions"]) != 12 {
		t.Fatalf("%+v", snap)
	}
	land, _ := snap["landing_pages"].([]map[string]any)
	if len(land) != ga4SnapshotCap {
		t.Fatalf("cap %d", len(land))
	}
	ch, _ := snap["channels"].([]map[string]any)
	if len(ch) != 1 || strVal(ch[0]["name"]) != "Organic Search" {
		t.Fatalf("%+v", ch)
	}
}

func TestInternalSeoGA4Unauthorized(t *testing.T) {
	s, err := New(nil, "secret")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/v1/seo/ga4", strings.NewReader(`{"source":"ga4_data_api"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
	ok := httptest.NewRequest(http.MethodPost, "/internal/v1/seo/ga4", strings.NewReader(`{"source":"ga4_data_api","property_id":"1"}`))
	ok.Header.Set("X-Internal-Token", "secret")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, ok)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("nil store %d", rec.Code)
	}
}
