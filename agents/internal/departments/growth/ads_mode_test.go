package growth

import "testing"

func TestAdsCampaignNameSplitsDiscoveryFromMature(t *testing.T) {
	if got := AdsCampaignName("homegauge", "HomeGauge", AdsModeDiscovery); got != "homegauge-discovery" {
		t.Fatalf("discovery %q", got)
	}
	if got := AdsCampaignName("homegauge", "HomeGauge", AdsModeMature); got != "homegauge" {
		t.Fatalf("mature %q", got)
	}
	if got := AdsCampaignName("homegauge-discovery", "", AdsModeDiscovery); got != "homegauge-discovery" {
		t.Fatalf("already suffixed %q", got)
	}
	if AdsReserveDivisor(AdsModeDiscovery) != 90 || AdsReserveDivisor(AdsModeMature) != 30 {
		t.Fatal("reserve")
	}
}

func TestFilterAdsCellsMatureNeedsSegments(t *testing.T) {
	cells := []AdsCell{{Geo: "Lagos", Theme: "mortgage help"}, {Geo: "Abuja", Theme: "rent"}}
	got, err := FilterAdsCells(cells, AdsModeDiscovery, "")
	if err != nil || len(got) != 2 {
		t.Fatalf("discovery %+v %v", got, err)
	}
	if _, err := FilterAdsCells(cells, AdsModeMature, ""); err == nil {
		t.Fatal("mature without segments must refuse")
	}
	got, err = FilterAdsCells(cells, AdsModeMature, "Lagos · mortgage help")
	if err != nil || len(got) != 1 || got[0].Geo != "Lagos" {
		t.Fatalf("mature filter %+v %v", got, err)
	}
	if _, err := FilterAdsCells(cells, AdsModeMature, "Ikeja age 25-34"); err == nil {
		t.Fatal("unmatched segment must refuse")
	}
}

func TestNormalizeAdsMode(t *testing.T) {
	if NormalizeAdsMode("") != AdsModeDiscovery || NormalizeAdsMode("scale") != AdsModeMature {
		t.Fatal("normalize")
	}
}
