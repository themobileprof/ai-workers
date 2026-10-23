package admin

import (
	"strings"
	"testing"
)

func TestKeywordInventoryBodyOutranksTitle(t *testing.T) {
	pages := []fetchedPage{
		{
			URL:   "https://homegauge.themobileprof.com/",
			Title: "Welcome solutions platform",
			Meta:  "Our solutions for everyone",
			H1:    "Solutions",
			Text:  "Ikeja mortgage paperwork for salaried staff. Ikeja mortgage paperwork is filed at the desk. Salaried Nigerians in Ikeja get eligibility checks.",
		},
	}
	got := keywordInventory(pages)
	if len(got) == 0 {
		t.Fatal("empty")
	}
	var bodyPhrase, brochure *keywordPhrase
	for i := range got {
		k := &got[i]
		if k.Phrase == "ikeja mortgage paperwork" {
			bodyPhrase = k
		}
		if k.Phrase == "welcome solutions platform" || k.Phrase == "solutions platform" {
			brochure = k
		}
	}
	if bodyPhrase == nil {
		t.Fatalf("missing body phrase: %+v", got)
	}
	if bodyPhrase.Role != "gap" || !bodyPhrase.InBody || bodyPhrase.InTitle {
		t.Fatalf("body phrase %+v", bodyPhrase)
	}
	if brochure != nil && brochure.Role != "brochure" {
		t.Fatalf("title-only should be brochure: %+v", brochure)
	}
	if brochure != nil && got[0].Phrase == brochure.Phrase {
		t.Fatal("brochure title ranked above body")
	}
	if got[0].Role == "brochure" {
		t.Fatalf("top phrase should be from the body, got %+v", got[0])
	}
}

func TestKeywordInventoryFromPages(t *testing.T) {
	pages := []fetchedPage{
		{
			URL:   "https://homegauge.themobileprof.com/",
			Title: "HomeGauge mortgage help Lagos",
			H1:    "Mortgage help for salaried Nigerians",
			Meta:  "Ikeja mortgage rates and paperwork",
			Text:  "HomeGauge mortgage help Lagos. Ikeja mortgage rates for salaried Nigerians. Click here to read more.",
		},
		{
			URL:   "https://homegauge.themobileprof.com/rates",
			Title: "Ikeja mortgage rates",
			H1:    "Ikeja mortgage rates",
			Text:  "Today’s Ikeja mortgage rates from named desks. Ikeja mortgage rates update each week.",
		},
	}
	got := keywordInventory(pages)
	if len(got) == 0 {
		t.Fatal("empty")
	}
	var sawRates, sawClick bool
	for _, k := range got {
		if k.Phrase == "click here" || k.Phrase == "read more" {
			sawClick = true
		}
		if k.Phrase == "ikeja mortgage rates" {
			sawRates = true
			if k.Role != "aligned" && k.Role != "cannibal" && k.Role != "gap" {
				t.Fatalf("rates role %s", k.Role)
			}
			if !k.InBody {
				t.Fatal("rates should count in body")
			}
		}
	}
	if !sawRates {
		t.Fatalf("missing ikeja mortgage rates: %+v", got)
	}
	if sawClick {
		t.Fatal("nav junk")
	}
	m := keywordMap(pages, got)
	if len(m) != 2 {
		t.Fatalf("map %d", len(m))
	}
	if m[1].BodyPrimary == "" {
		t.Fatalf("rates page missing body_primary %+v", m[1])
	}
}

func TestSeoPackMarkdownForCursor(t *testing.T) {
	body := seoPackMarkdown(SeoBrief{
		ID: 3, Kind: "onsite", Title: "Ikeja rates", URL: "https://homegauge.themobileprof.com/rates",
		Status: "approved", PackToken: "abc",
		Inventory: []keywordPhrase{{Phrase: "ikeja mortgage rates", Role: "gap", BodyCount: 4, InBody: true}},
	}, Project{Name: "HomeGauge", Slug: "homegauge", GitRepo: "themobileprof/homegauge"}, nil)
	for _, want := range []string{"does not publish", "Ikeja rates", "ikeja mortgage rates", "Cursor", "does not open a GitHub PR", "visible page", "Visible first"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	d := SeoBrief{ID: 3, Status: "proposed", PackToken: "abc", ProjectName: "HomeGauge"}
	if d.PackPath() != "/site/seo/abc" || d.PackDownloadPath() != "/admin/seo/3/pack.md" {
		t.Fatalf("%s %s", d.PackPath(), d.PackDownloadPath())
	}
	d.Status = "rejected"
	if d.PackPath() != "" {
		t.Fatalf("rejected pack still public %s", d.PackPath())
	}
	if d.PackFilename() != "homegauge-seo-3.md" {
		t.Fatal(d.PackFilename())
	}
}
