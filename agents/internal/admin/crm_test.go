package admin

import (
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
	if crmDirection("inbound") != "in" || crmDirection("outbound") != "out" {
		t.Fatal(crmDirection("inbound"))
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
