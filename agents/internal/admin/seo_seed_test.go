package admin

import (
	"strings"
	"testing"
)

func TestExtractCitesPrefersNamedFacts(t *testing.T) {
	pages := []fetchedPage{{
		URL:   "https://homegauge.themobileprof.com/rates",
		Title: "Rates",
		H1:    "Ikeja mortgage rates",
		Text:  "Click here. HomeGauge lists Ikeja owner-occupier mortgage rates from named desks in Lagos. Privacy policy. Welcome to our site.",
	}}
	got := extractCites(pages, Project{Name: "HomeGauge", Slug: "homegauge"})
	if len(got) == 0 || !strings.Contains(got[0], "Ikeja") {
		t.Fatalf("%q", got)
	}
}

func TestExtractFAQsFromQuestionSentences(t *testing.T) {
	pages := []fetchedPage{{
		H1:   "Academy",
		Text: "What does the Academy teach? Practical tech courses in Lagos. Click next.",
	}}
	got := extractFAQs(pages, []string{"The Academy runs practical tech courses in Lagos."}, Project{Name: "Academy", OneLiner: "Courses for working adults in Lagos."})
	if len(got) == 0 {
		t.Fatal("no faqs")
	}
	found := false
	for _, f := range got {
		if strings.Contains(f.Q, "Academy") && f.A != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%+v", got)
	}
}

func TestProposeSchemaAddsOrganization(t *testing.T) {
	raw := proposeSchemaJSON(nil, Project{URL: "https://academy.themobileprof.com"}, nil)
	if !strings.Contains(raw, "Organization") || !strings.Contains(raw, seoCompanyName) {
		t.Fatal(raw)
	}
}

func TestBuildSeoSeedFillsWalk(t *testing.T) {
	pages := []fetchedPage{{
		URL:    "https://homegauge.themobileprof.com/rates",
		Title:  "Ikeja mortgage rates | HomeGauge",
		H1:     "Ikeja mortgage rates",
		Meta:   "Owner-occupier rates from named Ikeja desks.",
		Text:   "HomeGauge lists Ikeja owner-occupier mortgage rates in Lagos. What is owner-occupier? A person buying the house they will live in.",
		JSONLD: []jsonLDBlock{{Types: []string{"WebPage"}, Raw: `{"@type":"WebPage","name":"Rates"}`}},
	}}
	seed := buildSeoSeed(pages, Project{Name: "HomeGauge", Slug: "homegauge", URL: pages[0].URL, OneLiner: "Ikeja mortgage rates for owner-occupiers."}, nil)
	if seed.Keyword == "" || seed.Title == "" || seed.H1 == "" {
		t.Fatalf("chrome %+v", seed)
	}
	if len(seed.Cites) == 0 || seed.JSONLD == "" || !strings.Contains(seed.JSONLD, "Organization") {
		t.Fatalf("seed %+v json %s", seed.Cites, seed.JSONLD)
	}
}

func TestParseFAQTextAndCiteLines(t *testing.T) {
	faqs := parseFAQText("Q: Where is HomeGauge?\nA: Ikeja, Lagos.\n")
	if len(faqs) != 1 || faqs[0].Q != "Where is HomeGauge?" {
		t.Fatalf("%+v", faqs)
	}
	cites := parseCiteLines("- One fact.\n- Two fact.\n")
	if len(cites) != 2 {
		t.Fatalf("%q", cites)
	}
}

func TestApplyAskFormOverridesSeed(t *testing.T) {
	seed := applyAskForm(seoSeed{Cites: []string{"old"}}, seoAskForm{CitesRaw: "edited fact from the form\n"})
	if len(seed.Cites) != 1 || seed.Cites[0] != "edited fact from the form" {
		t.Fatalf("%q", seed.Cites)
	}
}
