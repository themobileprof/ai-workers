package admin

import (
	"strings"
	"testing"
)

func TestNormalizeTelegramID(t *testing.T) {
	if got := normalizeTelegramID("  @123 45 "); got != "12345" {
		t.Fatalf("%q", got)
	}
}

func TestStaffMay(t *testing.T) {
	owner := User{Role: "owner", Active: true}
	if !staffMay(owner, "accounts") || !staffCanApprove(owner) {
		t.Fatal("owner")
	}
	bdm := User{Role: "bdm", Active: true}
	if !staffMay(bdm, "legal") || !staffCanApprove(bdm) {
		t.Fatal("bdm")
	}
	face := User{Role: "cofounder", Active: true}
	if !staffMay(face, "delivery") || !staffMay(face, "files") || staffMay(face, "accounts") || staffCanApprove(face) {
		t.Fatal("cofounder")
	}
	assist := User{Role: "assistant", Active: true}
	if !staffMay(assist, "delivery") || staffMay(assist, "files") || staffMay(assist, "legal") {
		t.Fatal("assistant")
	}
	handler := User{Role: "assistant", Active: true, Desks: []string{"legal"}}
	if !staffMay(handler, "legal") || staffMay(handler, "accounts") {
		t.Fatal("ticked desk")
	}
}

func TestStaffDefaultDept(t *testing.T) {
	if staffDefaultDept(User{Role: "owner", Active: true}) != "internal-ops" {
		t.Fatal("office")
	}
	if staffDefaultDept(User{Role: "cofounder", Active: true}) != "delivery" {
		t.Fatal("face")
	}
}

func TestStaffPrefixes(t *testing.T) {
	owner := staffPrefixes(User{Role: "owner", Active: true})
	need := []string{"/accounts", "/ops", "/file", "/approve", "/stop"}
	for _, p := range need {
		ok := false
		for _, got := range owner {
			if got == p {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("owner missing %s: %v", p, owner)
		}
	}
	face := staffPrefixes(User{Role: "cofounder", Active: true})
	if len(face) != 2 || face[0] != "/today" || face[1] != "/file" {
		t.Fatalf("cofounder %v", face)
	}
	if staffCanApprove(User{Role: "cofounder", Active: true}) {
		t.Fatal("face must not approve")
	}
}

func TestMailWatchers(t *testing.T) {
	users := []User{
		{ID: 1, Name: "Sam", Role: "owner", Active: true, TelegramID: "111"},
		{ID: 2, Name: "Biz", Role: "bdm", Active: true, TelegramID: "222"},
		{ID: 3, Name: "Ada", Role: "cofounder", Active: true, TelegramID: "333"},
		{ID: 4, Name: "Quiet", Role: "cofounder", Active: true},
		{ID: 5, Name: "Field", Role: "assistant", Active: true, TelegramID: "555"},
		{ID: 6, Name: "Old", Role: "owner", Active: false, TelegramID: "666"},
	}
	projects := []Project{{
		ID: 9, Slug: "academy", Name: "Academy",
		Members: []ProjectMember{
			{UserID: 3, Seat: "cofounder"},
			{UserID: 5, Seat: "assistant"},
			{UserID: 4, Seat: "cofounder"},
		},
	}}
	none := mailWatchers(users, projects, "")
	if len(none) != 2 {
		t.Fatalf("office only %+v", none)
	}
	got := mailWatchers(users, projects, "Academy")
	if len(got) != 3 {
		t.Fatalf("plus face %+v", got)
	}
	var roles []string
	for _, row := range got {
		roles = append(roles, row["role"].(string))
	}
	if strings.Join(roles, ",") != "owner,bdm,cofounder" {
		t.Fatalf("roles %v", roles)
	}
	if got[2]["project"] != "academy" || got[2]["telegram_id"] != "333" {
		t.Fatalf("face %+v", got[2])
	}
}
