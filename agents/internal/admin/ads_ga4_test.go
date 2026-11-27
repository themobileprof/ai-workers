package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/samuel/ai-workers/agents/internal/departments/growth"
)

func TestParseAdsGA4Snapshot(t *testing.T) {
	if _, err := parseAdsGA4Snapshot(nil, nil); err == nil {
		t.Fatal("empty")
	}
	snap, err := parseAdsGA4Snapshot(map[string]any{
		"source":      "ga4_data_api",
		"property_id": "properties/99",
		"paid": map[string]any{
			"dimensionHeaders": []any{
				map[string]any{"name": "sessionSourceMedium"},
				map[string]any{"name": "sessionCampaignName"},
				map[string]any{"name": "sessionManualAdContent"},
			},
			"metricHeaders": []any{
				map[string]any{"name": "sessions"},
				map[string]any{"name": "engagedSessions"},
				map[string]any{"name": "bounceRate"},
				map[string]any{"name": "keyEvents"},
			},
			"rows": []any{
				map[string]any{
					"dimensionValues": []any{
						map[string]any{"value": "google / cpc"},
						map[string]any{"value": "homegauge"},
						map[string]any{"value": "lagos-mortgage"},
					},
					"metricValues": []any{
						map[string]any{"value": "12"},
						map[string]any{"value": "8"},
						map[string]any{"value": "0.25"},
						map[string]any{"value": "2"},
					},
				},
			},
		},
		"funnel": map[string]any{
			"funnelTable": map[string]any{
				"dimensionHeaders": []any{
					map[string]any{"name": "funnelStepName"},
					map[string]any{"name": "sessionCampaignName"},
				},
				"metricHeaders": []any{map[string]any{"name": "activeUsers"}},
				"rows": []any{
					map[string]any{
						"dimensionValues": []any{map[string]any{"value": "Session"}, map[string]any{"value": "homegauge"}},
						"metricValues":    []any{map[string]any{"value": "10"}},
					},
					map[string]any{
						"dimensionValues": []any{map[string]any{"value": "Landing"}, map[string]any{"value": "homegauge"}},
						"metricValues":    []any{map[string]any{"value": "6"}},
					},
				},
			},
		},
	}, growth.DefaultAdsFunnelSteps())
	if err != nil {
		t.Fatal(err)
	}
	if snap.PropertyID != "99" || len(snap.Paid) != 1 || snap.Paid[0].Campaign != "homegauge" || snap.Paid[0].KeyEvents != 2 {
		t.Fatalf("%+v", snap.Paid)
	}
	if len(snap.Funnel) != 2 || snap.Funnel[1].Users != 6 || snap.Funnel[1].Keep != 0.6 {
		t.Fatalf("%+v", snap.Funnel)
	}
	filtered := snap.ForSlug("homegauge")
	if len(filtered.Paid) != 1 {
		t.Fatalf("%+v", filtered)
	}
	empty := snap.ForSlug("academy")
	if len(empty.Paid) != 0 || !strings.Contains(empty.Note, "bet slug") {
		t.Fatalf("%+v", empty)
	}
}

func TestAdsGlanceHasPaidAndDay(t *testing.T) {
	if adsGlanceHasPaid(AdsGA4View{}) || adsGlanceHasPaid(AdsGA4View{Paid: []AdsGA4PaidRow{{Sessions: 0}}}) {
		t.Fatal("empty paid")
	}
	if !adsGlanceHasPaid(AdsGA4View{Paid: []AdsGA4PaidRow{{Sessions: 2}}}) {
		t.Fatal("paid")
	}
	id := glanceDayID(time.Date(2026, 11, 21, 23, 0, 0, 0, time.UTC))
	if id < 20261121 || id > 20261122 {
		t.Fatalf("day %d", id)
	}
	body := growth.FormatAdsGlance(compactAdsGA4(AdsGA4View{
		Found:  true,
		Paid:   []AdsGA4PaidRow{{Campaign: "homegauge", Content: "lagos", Sessions: 4, Engaged: 2}},
		Funnel: []AdsGA4FunnelRow{{Name: "Session", Users: 4}, {Name: "Landing", Users: 2}},
	}))
	if !strings.Contains(body, "4 sessions") || !strings.Contains(body, "cannot spend") {
		t.Fatal(body)
	}
}

func TestWinningPaidCellsAndSenseCopy(t *testing.T) {
	snap := AdsGA4View{
		Found: true,
		Paid: []AdsGA4PaidRow{
			{Campaign: "homegauge-discovery", Content: "lagos-mortgage", Sessions: 12, Engaged: 8},
			{Campaign: "homegauge-discovery", Content: "abuja-rent", Sessions: 3, Engaged: 1},
			{Campaign: "homegauge-discovery", Content: "quiet", Sessions: 0, Engaged: 0},
		},
	}
	winners := winningPaidCells(snap.Paid, 3)
	if len(winners) != 2 || winners[0].Content != "lagos-mortgage" {
		t.Fatalf("%+v", winners)
	}
	ev, segs, body := formatPaidSense(Project{Name: "HomeGauge", Slug: "homegauge"}, snap)
	if !strings.Contains(ev, "lagos-mortgage") || !strings.Contains(segs, "lagos-mortgage") {
		t.Fatalf("evidence %q segs %q", ev, segs)
	}
	if !strings.Contains(body, "homegauge-discovery") || !strings.Contains(strings.ToLower(body), "cannot mark the card mature") {
		t.Fatal(body)
	}
	if strings.Contains(strings.ToLower(body), "kill the bet from chat") {
		t.Fatal(body)
	}
}

func TestForSlugMatchesDiscoveryCampaign(t *testing.T) {
	snap := AdsGA4View{Found: true, Paid: []AdsGA4PaidRow{
		{Campaign: "homegauge-discovery", Content: "lagos-mortgage", Sessions: 4},
	}}
	got := snap.ForSlug("homegauge")
	if len(got.Paid) != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestInternalAdsGA4Unauthorized(t *testing.T) {
	s, err := New(nil, "secret")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/v1/ads/ga4", strings.NewReader(`{"source":"ga4_data_api"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}
