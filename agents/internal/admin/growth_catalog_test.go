package admin

import "testing"

func TestPublicGrowthOffersSkipsParkedAndKilled(t *testing.T) {
	got := publicGrowthOffers([]Project{
		{Name: "HomeGauge", Slug: "homegauge", Stage: "selling", OneLiner: "Salaried Nigerian homebuyers get mortgage help.", URL: "https://homegauge.themobileprof.com"},
		{Name: "Dead", Slug: "dead", Stage: "killed", OneLiner: "Gone."},
		{Name: "Park", Slug: "park", Stage: "parked", OneLiner: "Later."},
		{Name: "Academy", Slug: "academy", Stage: "testing", OneLiner: "Live courses.", URL: "https://lms.themobileprof.com"},
	})
	if len(got) != 2 {
		t.Fatalf("want 2 public products, got %+v", got)
	}
	if got[0]["name"] != "HomeGauge" || got[1]["name"] != "Academy" {
		t.Fatalf("%+v", got)
	}
	if _, ok := got[0]["burn_ngn"]; ok {
		t.Fatal("customer catalog must not include burn")
	}
}

func TestAttachGrowthCatalogOnWhatsAppSales(t *testing.T) {
	data := map[string]any{"channel": "whatsapp"}
	projects := []Project{
		{ID: 7, Name: "HomeGauge", Slug: "homegauge", Stage: "selling", OneLiner: "Salaried Nigerian homebuyers get mortgage help.", URL: "https://homegauge.themobileprof.com"},
		{ID: 9, Name: "Mechazone", Slug: "mechazone", Stage: "testing", OneLiner: "A bookkeeping assistant for mechanic workshops.", URL: "https://mechazone.themobileprof.com"},
	}
	attachGrowthCatalog(data, projects)
	p, ok := matchProject(projects, data, "How much is HomeGauge?")
	if !ok || p.Slug != "homegauge" {
		t.Fatalf("match %+v %v", p, ok)
	}
	attachCustomerOffer(data, p)
	products, _ := data["products"].([]map[string]any)
	if len(products) != 2 {
		t.Fatalf("products %+v", data["products"])
	}
	proj, _ := data["project"].(map[string]any)
	if proj["name"] != "HomeGauge" || proj["url"] != "https://homegauge.themobileprof.com" {
		t.Fatalf("project %+v", proj)
	}
	if _, ok := proj["burn_ngn"]; ok {
		t.Fatal("customer overlay must not leak burn")
	}
	if data["company"] != "TheMobileProf Technologies" {
		t.Fatalf("company %v", data["company"])
	}
}

func TestPickWhatsAppOfferStickyUntilTheyNameAnother(t *testing.T) {
	projects := []Project{
		{ID: 7, Name: "HomeGauge", Slug: "homegauge", Stage: "selling"},
		{ID: 9, Name: "Mechazone", Slug: "mechazone", Stage: "testing"},
		{ID: 3, Name: "Dead", Slug: "dead", Stage: "killed"},
	}
	home := projects[0]
	p, kind := pickWhatsAppOffer(projects, "Tell me about HomeGauge", Project{})
	if kind != offerNamed || p.ID != 7 {
		t.Fatalf("first turn %+v %s", p, kind)
	}
	p, kind = pickWhatsAppOffer(projects, "How much is it?", home)
	if kind != offerSticky || p.ID != 7 {
		t.Fatalf("follow-up should keep HomeGauge %+v %s", p, kind)
	}
	p, kind = pickWhatsAppOffer(projects, "What about Mechazone instead?", home)
	if kind != offerNamed || p.ID != 9 {
		t.Fatalf("named switch %+v %s", p, kind)
	}
	p, kind = pickWhatsAppOffer(projects, "Compare HomeGauge and Mechazone", home)
	if kind != offerSeveral || p.ID != 0 {
		t.Fatalf("two names must not pick one %+v %s", p, kind)
	}
	p, kind = pickWhatsAppOffer(projects, "How much is Dead?", home)
	if kind != offerSticky || p.ID != 7 {
		t.Fatalf("killed name is not a switch %+v %s", p, kind)
	}
}

func TestAcademyAskDoesNotFireOnMechanicWorkshop(t *testing.T) {
	if academyAsk("How much is the bookkeeping assistant for a mechanic workshop in Ikeja?", Project{Name: "Mechazone", Slug: "mechazone"}) {
		t.Fatal("workshop in a product question is not Academy")
	}
	if !academyAsk("Do you have a Python course?", Project{}) {
		t.Fatal("course")
	}
	if !academyAsk("What does Academy cost?", Project{Name: "Academy", Slug: "academy"}) {
		t.Fatal("named Academy")
	}
}
