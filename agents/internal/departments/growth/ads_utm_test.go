package growth

import (
	"net/url"
	"strings"
	"testing"
)

func TestAdsTrackURLStampsMissingUTM(t *testing.T) {
	got := AdsTrackURL("https://homegauge.themobileprof.com/buy", "google", "homegauge", "Lagos mortgage")
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("utm_source") != "google" || q.Get("utm_medium") != "cpc" || q.Get("utm_campaign") != "homegauge" || q.Get("utm_content") != "lagos-mortgage" {
		t.Fatal(got)
	}
	if u.Path != "/buy" {
		t.Fatal(u.Path)
	}
}

func TestAdsTrackURLLeavesExisting(t *testing.T) {
	got := AdsTrackURL("https://x.test/?utm_source=newsletter&utm_campaign=old", "meta", "homegauge", "ikeja")
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("utm_source") != "newsletter" || q.Get("utm_campaign") != "old" {
		t.Fatal(got)
	}
	if q.Get("utm_medium") != "cpc" {
		t.Fatal(got)
	}
}

func TestAdsTrackURLMetaSource(t *testing.T) {
	got := AdsTrackURL("https://x.test/", "meta", "academy", "")
	if !strings.Contains(got, "utm_source=facebook") {
		t.Fatal(got)
	}
}

func TestParseAdsFunnelSteps(t *testing.T) {
	d := ParseAdsFunnelSteps("")
	if len(d) != 2 || d[0].Event != "session_start" || d[1].Event != "page_view" {
		t.Fatalf("%+v", d)
	}
	got := ParseAdsFunnelSteps("Session:session_start, Landing:page_view, Lead:generate_lead")
	if len(got) != 3 || got[2].Event != "generate_lead" {
		t.Fatalf("%+v", got)
	}
	path := ParseAdsFunnelSteps("Session:session_start\nWaitlist:/waitlist")
	if len(path) != 2 || path[1].Path != "/waitlist" {
		t.Fatalf("%+v", path)
	}
	if len(ParseAdsFunnelSteps("only-one:page_view")) != 2 {
		t.Fatal("need two steps")
	}
	if AdsFunnelLineOK("") || AdsFunnelLineOK("only-one:page_view") || AdsFunnelLineOK("garbage") {
		t.Fatal("strict line needs two Label:event parts")
	}
	if !AdsFunnelLineOK("Session:session_start, Landing:page_view") {
		t.Fatal("default line should parse")
	}
}

func TestAdsFunnelReportBody(t *testing.T) {
	body := AdsFunnelReportBody(ParseAdsFunnelSteps("Session:session_start, Landing:page_view"))
	funnel, _ := body["funnel"].(map[string]any)
	steps, _ := funnel["steps"].([]map[string]any)
	if len(steps) != 2 {
		t.Fatalf("%+v", body)
	}
	paid := AdsPaidReportBody()
	if paid["limit"] != 30 {
		t.Fatalf("%+v", paid)
	}
}
