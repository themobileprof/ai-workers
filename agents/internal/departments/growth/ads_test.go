package growth

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func TestClipAdsLimits(t *testing.T) {
	p := ClipAds(AdsPack{
		Headlines:    []AdsLine{{Text: strings.Repeat("A", 40)}},
		Descriptions: []AdsLine{{Text: strings.Repeat("B", 100)}},
	})
	if len([]rune(p.Headlines[0].Text)) != 30 {
		t.Fatal(p.Headlines[0].Text)
	}
	if len([]rune(p.Descriptions[0].Text)) != 90 {
		t.Fatal(len([]rune(p.Descriptions[0].Text)))
	}
}

func TestLooksLikeAds(t *testing.T) {
	if LooksLikeAds(map[string]any{"channel": "whatsapp"}, "google ads for HomeGauge") {
		t.Fatal("customer WhatsApp stays sales")
	}
	if !LooksLikeAds(map[string]any{"channel": "desk", "action": "ads"}, "continue") {
		t.Fatal("desk ads")
	}
	if !LooksLikeAds(map[string]any{"channel": "telegram"}, "Search ads for HomeGauge. Pexels queries.") {
		t.Fatal("telegram ads")
	}
	if LooksLike(map[string]any{"channel": "desk", "action": "ads"}, "google ads") && !LooksLikeAds(map[string]any{"channel": "desk", "action": "ads"}, "google ads") {
		t.Fatal("ads must win")
	}
}

func TestAdsMarkdownNoSpend(t *testing.T) {
	md := AdsMarkdown(SeedAds("https://homegauge.themobileprof.com"), "HomeGauge", "homegauge")
	if !strings.Contains(md, "Place") || !strings.Contains(md, "Pexels") || !strings.Contains(md, "homegauge") || !strings.Contains(md, "experiment") {
		t.Fatal(md)
	}
	if !strings.Contains(md, "Book it yourself") || !strings.Contains(md, "ads.google.com") {
		t.Fatal(md)
	}
	if strings.Contains(strings.ToLower(md), "waitlist") || strings.Contains(md, "MAU") {
		t.Fatal(md)
	}
}

func TestHandleAdsSetsCannotSpend(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"URL from the Card. Next?","structured_data":{"task_type":"ads","ads":{"final_url":"https://homegauge.themobileprof.com","current":1}}}`}, contract.Request{
		TaskDescription: "Search ads for HomeGauge",
		ContextData:     []byte(`{"channel":"desk","action":"ads","project":{"slug":"homegauge","url":"https://homegauge.themobileprof.com","one_liner":"Salaried Nigerian homebuyers get mortgage help."}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "ads" || resp.StructuredData["cannot_spend"] != true || resp.StructuredData["cannot_send"] != true {
		t.Fatalf("%+v", resp.StructuredData)
	}
}

func TestPexelsSearchURL(t *testing.T) {
	u := PexelsQuery{Query: "house keys close-up"}.SearchURL()
	if !strings.Contains(u, "pexels.com/search/") || !strings.Contains(u, "keys") {
		t.Fatal(u)
	}
}

func TestAdsWalkMarksFilledSteps(t *testing.T) {
	p := ClipAds(AdsPack{
		Current:   3,
		FinalURL:  "https://homegauge.themobileprof.com",
		Cells:     []AdsCell{{Geo: "Lagos", Theme: "mortgage help"}},
		Keywords:  []string{"mortgage help Lagos"},
		Headlines: []AdsLine{{Text: strings.Repeat("H", 40)}},
	})
	walk := p.Walk()
	if len(walk) != 6 {
		t.Fatalf("steps %d", len(walk))
	}
	if walk[0].Status != "agreed" || walk[1].Status != "agreed" || walk[2].Status != "draft" {
		t.Fatalf("%+v", walk)
	}
	if len([]rune(p.Headlines[0].Text)) != 30 {
		t.Fatal(p.Headlines[0].Text)
	}
	if walk[3].Status != "agreed" || walk[4].Status != "draft" || walk[5].Status != "draft" {
		t.Fatalf("filled-not-current agreed, empty draft: %+v", walk)
	}
}

func TestAdsWasteReason(t *testing.T) {
	if AdsWasteReason(0, 0, 0) != "" {
		t.Fatal("no spend is still learning")
	}
	if AdsWasteReason(1, 0, 0) != "spent with 0 clicks after a week" {
		t.Fatal(AdsWasteReason(1, 0, 0))
	}
	if AdsWasteReason(10, 4, 0) != "" {
		t.Fatal("four clicks is not enough to cut")
	}
	if AdsWasteReason(10, 5, 0) != "clicks with 0 conversions after two weeks" {
		t.Fatal(AdsWasteReason(10, 5, 0))
	}
	if AdsWasteReason(10, 5, 1) != "" {
		t.Fatal("a conversion stays")
	}
	if AdsCellWasteReason(3, 5, 0, 0, true) != "" {
		t.Fatal("three days is too short to cut a cell")
	}
	if AdsCellWasteReason(7, 1, 0, 0, true) != "spent with 0 clicks after a week" {
		t.Fatal("a week of spend with no clicks is dead")
	}
	if AdsCellWasteReason(7, 10, 5, 0, true) != "" {
		t.Fatal("conversions wait two weeks")
	}
	if AdsCellWasteReason(14, 10, 5, 0, false) != "" {
		t.Fatal("missing conversion tracking must not pause Search")
	}
}

func hourPtr(h int) *int { return &h }

func TestAdsDaypartID(t *testing.T) {
	if AdsDaypartID("MONDAY", 8, true) != "weekday_morning" {
		t.Fatal(AdsDaypartID("MONDAY", 8, true))
	}
	if AdsDaypartID("FRIDAY", 18, true) != "weekday_evening" {
		t.Fatal(AdsDaypartID("FRIDAY", 18, true))
	}
	if AdsDaypartID("SATURDAY", 12, true) != "weekend" {
		t.Fatal(AdsDaypartID("SATURDAY", 12, true))
	}
	if AdsDaypartID("SUNDAY", 3, true) != "night" {
		t.Fatal(AdsDaypartID("SUNDAY", 3, true))
	}
	if AdsDaypartID("", 8, true) != "hour_morning" {
		t.Fatal(AdsDaypartID("", 8, true))
	}
	if AdsDaypartID("SATURDAY", 0, false) != "days_weekend" {
		t.Fatal(AdsDaypartID("SATURDAY", 0, false))
	}
}

func TestAdsDecideWaitsAWeek(t *testing.T) {
	out := AdsDecide(AdsDecideIn{
		Channel: "google", ID: "customers/1/campaigns/2", AgeDays: 2,
		Rows: []AdsMetricRow{{Key: "ag1", Name: "Lagos", Cost: 4, Hour: hourPtr(8), DayOfWeek: "MONDAY"}},
	})
	if !out.Skip {
		t.Fatalf("%+v", out)
	}
}

func TestAdsDecideCutsDeadCellAfterWeek(t *testing.T) {
	out := AdsDecide(AdsDecideIn{
		Channel: "google", ID: "c", CustomerID: "1", CampaignID: "2", AgeDays: 8,
		Rows: []AdsMetricRow{{Key: "customers/1/adGroups/9", Name: "Ikeja · mortgage", Cost: 3, Clicks: 0, Hour: hourPtr(9), DayOfWeek: "MONDAY"}},
	})
	if out.Skip || !strings.Contains(out.Note, "spent with 0 clicks after a week") {
		t.Fatalf("%+v", out)
	}
	ops, _ := out.GoogleBody["mutateOperations"].([]map[string]any)
	if len(ops) != 1 {
		t.Fatalf("ops %+v", out.GoogleBody)
	}
}

func TestAdsDecideCutsDeadDaypartKeepsCell(t *testing.T) {
	out := AdsDecide(AdsDecideIn{
		Channel:          "google",
		ID:               "c",
		CustomerID:       "1",
		CampaignID:       "2",
		CampaignResource: "customers/1/campaigns/2",
		AgeDays:          8,
		Rows: []AdsMetricRow{
			{Key: "ag1", Name: "Lagos · mortgage", Cost: 2, Clicks: 0, Hour: hourPtr(3), DayOfWeek: "TUESDAY"},
			{Key: "ag1", Name: "Lagos · mortgage", Cost: 2, Clicks: 4, Hour: hourPtr(18), DayOfWeek: "TUESDAY"},
		},
	})
	if out.Skip {
		t.Fatal("should cut night")
	}
	if strings.Contains(out.Note, "ad group") || strings.Contains(out.Note, "spent with 0 clicks after a week") {
		t.Fatalf("must not pause the cell: %s", out.Note)
	}
	if !strings.Contains(out.Note, "night") {
		t.Fatal(out.Note)
	}
	found := false
	for _, d := range out.Dropped {
		if d == "night" {
			found = true
		}
	}
	if !found {
		t.Fatalf("dropped %+v", out.Dropped)
	}
	ops, _ := out.GoogleBody["mutateOperations"].([]map[string]any)
	if len(ops) == 0 {
		t.Fatal("need schedule mutate")
	}
	for _, op := range ops {
		if _, ok := op["adGroupOperation"]; ok {
			t.Fatalf("paused the cell %+v", op)
		}
	}
}

func TestAdsDecideDoesNotReopenDroppedNight(t *testing.T) {
	out := AdsDecide(AdsDecideIn{
		Channel: "google", ID: "c", CustomerID: "1", CampaignID: "2", AgeDays: 10,
		Dropped: []string{"night"},
		Rows: []AdsMetricRow{
			{Key: "ag1", Name: "Lagos", Cost: 2, Clicks: 3, Hour: hourPtr(10), DayOfWeek: "WEDNESDAY"},
		},
	})
	if !out.Skip {
		t.Fatalf("no new waste: %+v", out)
	}
	if len(out.Dropped) != 1 || out.Dropped[0] != "night" {
		t.Fatalf("%+v", out.Dropped)
	}
}

func TestAdsDecideMetaCutsWeekendPerCell(t *testing.T) {
	out := AdsDecide(AdsDecideIn{
		Channel: "meta", ID: "120", AgeDays: 8,
		Rows: []AdsMetricRow{
			{Key: "111", Name: "Abuja · training", Cost: 3, Clicks: 0, Date: "2026-11-21"}, // Saturday
			{Key: "111", Name: "Abuja · training", Cost: 3, Clicks: 5, Date: "2026-11-16"}, // Monday
			{Key: "111", Name: "Abuja · training", Cost: 1, Clicks: 2, Hour: hourPtr(12)},
		},
	})
	if out.Skip {
		t.Fatalf("%+v", out)
	}
	if len(out.Batch) != 1 {
		t.Fatalf("batch %+v", out.Batch)
	}
	body, _ := out.Batch[0]["body"].(string)
	if !strings.Contains(body, "adset_schedule=") {
		t.Fatal(body)
	}
}

func TestAdsCampaignIDs(t *testing.T) {
	cust, camp := AdsCampaignIDs("google", "customers/123/campaigns/456")
	if cust != "123" || camp != "456" {
		t.Fatalf("%s %s", cust, camp)
	}
	_, camp = AdsCampaignIDs("meta", "120398")
	if camp != "120398" {
		t.Fatal(camp)
	}
}

func TestAdsPlaceReady(t *testing.T) {
	if SeedAds("https://x.test").PlaceReady() == "" {
		t.Fatal("empty pack must not place")
	}
	p := ClipAds(AdsPack{
		FinalURL:     "https://homegauge.themobileprof.com",
		Keywords:     []string{"mortgage help Lagos"},
		Headlines:    []AdsLine{{Text: "Mortgage help"}, {Text: "Lagos buyers"}, {Text: "TheMobileProf"}},
		Descriptions: []AdsLine{{Text: "Salaried Nigerian homebuyers get mortgage help."}, {Text: "Named desks. Named officers."}},
	})
	if p.PlaceReady() != "" {
		t.Fatal(p.PlaceReady())
	}
	p = p.WithPlacement(AdsPlacement{Channel: "google", ID: "customers/1/campaigns/2", Mode: "place", At: "2026-11-21"})
	if !p.PlacedOn("google") || p.PlacedOn("meta") {
		t.Fatalf("%+v", p.Placements)
	}
	if p.BookedHow("google") != "place" {
		t.Fatalf("place mode %+v", p.Placements)
	}
}

func TestAdsBookConflictExclusive(t *testing.T) {
	manual := SeedAds("https://homegauge.themobileprof.com").WithPlacement(AdsPlacement{Channel: "google", ID: "manual", Mode: "manual"})
	if AdsBookConflict(manual, "google", "place") == "" {
		t.Fatal("manual must block Place")
	}
	if AdsBookConflict(manual, "google", "manual") != "" {
		t.Fatal("updating a manual campaign id is allowed")
	}
	placed := SeedAds("https://homegauge.themobileprof.com").WithPlacement(AdsPlacement{Channel: "google", ID: "customers/1/campaigns/2", Mode: "place"})
	if AdsBookConflict(placed, "google", "manual") == "" {
		t.Fatal("Place must block booking the same cells in the UI")
	}
	if AdsBookConflict(placed, "meta", "manual") != "" {
		t.Fatal("the other channel stays free")
	}
}

func TestAdsManualGoogleListsExactAndUTM(t *testing.T) {
	p := ClipAds(AdsPack{
		FinalURL:     "https://homegauge.themobileprof.com",
		Cells:        []AdsCell{{Geo: "Lagos", Theme: "mortgage help"}},
		Keywords:     []string{"mortgage help Lagos"},
		Headlines:    []AdsLine{{Text: "Mortgage help in Lagos"}},
		Descriptions: []AdsLine{{Text: "Salaried buyers get a desk that quotes today."}, {Text: "Named places on the Card only."}},
	})
	md := AdsManualMarkdown(p, "HomeGauge", "homegauge")
	for _, want := range []string{
		"Book it yourself",
		"Place automatically",
		"ads.google.com",
		"[mortgage help Lagos]",
		"homegauge",
		"utm_campaign=homegauge",
		"Paused",
		"pay twice",
		"adsmanager.facebook.com",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in %s", want, md)
		}
	}
	steps := AdsManualGoogle(p, "HomeGauge", "homegauge")
	if len(steps) < 10 {
		t.Fatalf("google steps %d", len(steps))
	}
}

func TestLooksLikeDoesNotClaimAds(t *testing.T) {
	data := map[string]any{"channel": "desk", "action": "ads"}
	if LooksLike(data, "google ads") {
		t.Fatal("ads must not look like a pitch deck")
	}
}
