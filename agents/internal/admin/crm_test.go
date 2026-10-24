package admin

import (
	"strings"
	"testing"
	"time"
)

func TestCRMStage(t *testing.T) {
	if crmStage("PROPOSAL") != "proposal" || crmStage("") != "new" {
		t.Fatal(crmStage("PROPOSAL"), crmStage(""))
	}
	c := CRMContact{Stage: "new"}
	if !c.Open() {
		t.Fatal("new")
	}
	c.Stage = "won"
	if c.Open() {
		t.Fatal("won")
	}
}

func TestFilterCRMContacts(t *testing.T) {
	list := []CRMContact{
		{ID: 1, Name: "Ada", ProjectID: 9},
		{ID: 2, Name: "Company-wide", ProjectID: 0},
		{ID: 3, Name: "Other bet", ProjectID: 8},
	}
	projects := []Project{{ID: 9, Members: []ProjectMember{{UserID: 3, Seat: "cofounder"}}}}
	office := filterCRMContacts(&User{Role: "owner"}, list, projects)
	if len(office) != 3 {
		t.Fatalf("office %d", len(office))
	}
	face := filterCRMContacts(&User{ID: 3, Role: "cofounder"}, list, projects)
	if len(face) != 1 || face[0].Name != "Ada" {
		t.Fatalf("face %+v", face)
	}
	handler := filterCRMContacts(&User{Role: "handler", Desks: []string{"crm"}}, list, projects)
	if len(handler) != 3 {
		t.Fatalf("crm handler %d", len(handler))
	}
	if canSeeCRMContact(&User{ID: 3, Role: "cofounder"}, list[1], projects) {
		t.Fatal("cofounder must not see company-wide")
	}
}

func TestCRMChannel(t *testing.T) {
	if crmChannel("WhatsApp") != "whatsapp" || crmChannel("mail") != "email" || crmChannel("voice") != "call" {
		t.Fatal(crmChannel("WhatsApp"), crmChannel("mail"), crmChannel("voice"))
	}
	if crmChannel("notes") != "call" || crmChannel("fathom") != "call" {
		t.Fatal(crmChannel("notes"), crmChannel("fathom"))
	}
	if crmDirection("inbound") != "in" || crmDirection("outbound") != "out" {
		t.Fatal(crmDirection("inbound"))
	}
}

func TestCRMBooksEligible(t *testing.T) {
	if !crmBooksEligible("ada@apex.ng", "new") || !crmBooksEligible("", "qualified") || !crmBooksEligible("  ", "proposal") {
		t.Fatal("email or qualified+")
	}
	if crmBooksEligible("2348091112222", "new") || crmBooksEligible("", "nurture") || crmBooksEligible("not-an-email", "lost") {
		t.Fatal("phone-only new stays desk CRM")
	}
}

func TestCRMWhatsAppFollowup(t *testing.T) {
	now := time.Date(2026, 10, 23, 16, 0, 0, 0, lagosLocation()) // Friday
	action, by := crmWhatsAppFollowup("whatsapp", "new", "", "", 0.8, now)
	if action != "Reply on WhatsApp" || by != "2026-10-26" {
		t.Fatalf("high intent weekend skip %q %q", action, by)
	}
	keepA, keepB := crmWhatsAppFollowup("whatsapp", "new", "Call Ikeja", "2026-10-28", 0.9, now)
	if keepA != "Call Ikeja" || keepB != "2026-10-28" {
		t.Fatalf("keep model date %q %q", keepA, keepB)
	}
	lowA, lowB := crmWhatsAppFollowup("whatsapp", "new", "", "", 0.2, now)
	if lowA != "" || lowB != "" {
		t.Fatalf("low intent stays empty %q %q", lowA, lowB)
	}
	tgA, tgB := crmWhatsAppFollowup("telegram", "qualified", "", "", 1, now)
	if tgA != "" || tgB != "" {
		t.Fatalf("telegram is not auto-dated %q %q", tgA, tgB)
	}
	qualA, qualB := crmWhatsAppFollowup("whatsapp", "qualified", "", "", 0.1, now)
	if qualA != "Reply on WhatsApp" || qualB != "2026-10-26" {
		t.Fatalf("qualified still dated %q %q", qualA, qualB)
	}
}

func TestCRMLookupQuery(t *testing.T) {
	if crmLookupQuery("/crm follow up Apex") != "Apex" {
		t.Fatal(crmLookupQuery("/crm follow up Apex"))
	}
	if crmLookupQuery("Follow-up Apex Motors") != "Apex Motors" {
		t.Fatal(crmLookupQuery("Follow-up Apex Motors"))
	}
	if crmLookupQuery("Chinedu") != "Chinedu" {
		t.Fatal(crmLookupQuery("Chinedu"))
	}
}

func TestCRMDueDigest(t *testing.T) {
	today := time.Date(2026, 10, 24, 10, 0, 0, 0, lagosLocation())
	if crmDueText(nil, today) != "" {
		t.Fatal("empty")
	}
	empty := crmDueDigest(nil, []User{{Role: "owner", Active: true, TelegramID: "111"}}, today)
	if empty["count"] != 0 {
		t.Fatalf("%+v", empty)
	}
	people, _ := empty["people"].([]map[string]any)
	if len(people) != 0 {
		t.Fatal("no ping when nothing is due")
	}
	list := []CRMContact{
		{ID: 12, Name: "Chinedu", Company: "Apex Motors", NextAction: "send the 30-day pilot note", NextActionBy: "2026-10-22"},
		{ID: 3, Name: "Ada", NextAction: "call Ikeja workshop", NextActionBy: "2026-10-24"},
	}
	text := crmDueText(list, today)
	if !strings.Contains(text, "overdue 22 Oct") || !strings.Contains(text, "/admin/crm/12") {
		t.Fatal(text)
	}
	if !strings.Contains(text, "(today)") || !strings.Contains(text, "does not message the customer") {
		t.Fatal(text)
	}
	if strings.Contains(strings.ToLower(text), "whatsapp") && strings.Contains(text, "sent") {
		t.Fatal("must not claim a customer send")
	}
	users := []User{
		{ID: 1, Name: "Sam", Role: "owner", Active: true, TelegramID: "111"},
		{ID: 2, Name: "Biz", Role: "bdm", Active: true, TelegramID: "222"},
		{ID: 3, Name: "Ada", Role: "cofounder", Active: true, TelegramID: "333"},
	}
	got := crmDueDigest(list, users, today)
	pings, _ := got["people"].([]map[string]any)
	if len(pings) != 2 {
		t.Fatalf("owner and bdm only %+v", pings)
	}
	if pings[0]["speak"] != true || pings[0]["telegram_id"] != "111" {
		t.Fatalf("%+v", pings[0])
	}
}

func TestCRMEventWhenLabel(t *testing.T) {
	e := CRMEvent{Channel: "call", Direction: "in", HappenedAt: time.Date(2026, 10, 22, 14, 5, 0, 0, time.UTC)}
	if e.KindLabel() != "call · in" {
		t.Fatal(e.KindLabel())
	}
	if e.WhenLabel() == "" {
		t.Fatal("when")
	}
}
