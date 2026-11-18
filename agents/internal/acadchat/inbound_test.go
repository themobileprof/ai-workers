package acadchat

import (
	"strings"
	"testing"
)

func TestParseJoin(t *testing.T) {
	raw := []byte(`{"update_id":1,"message":{"message_id":2,"date":1,"from":{"id":9,"is_bot":false,"first_name":"Ada"},"chat":{"id":-100,"type":"supergroup"},"new_chat_members":[{"id":11,"is_bot":false,"first_name":"Bola"}]}}`)
	in, ok := ParseUpdate(raw)
	if !ok || !in.Join || in.ChatID != "-100" || len(in.JoinNames) != 1 {
		t.Fatalf("%+v ok=%v", in, ok)
	}
}

func TestDecideLinkEmailInDM(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "2", Text: "my lms is bola@themobileprof.com", Group: false})
	if d.Kind != KindLink || d.Email != "bola@themobileprof.com" {
		t.Fatalf("%+v", d)
	}
	d = Decide(Inbound{ChatID: "-1", FromID: "2", Text: "my lms is bola@themobileprof.com", Group: true})
	if d.Kind != KindHelp || !strings.Contains(d.Reply, "Do not put your LMS email") {
		t.Fatalf("group must not link email %+v", d)
	}
}

func TestDecideJoinOnboard(t *testing.T) {
	d := Decide(Inbound{ChatID: "-1", FromID: "2", Join: true, JoinNames: []string{"Ada"}, Group: true})
	if d.Kind != KindOnboard {
		t.Fatalf("%+v", d)
	}
}

func TestDecideStartAndInvite(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "2", Text: "/start r_abc123", Group: false})
	if d.Kind != KindStart || d.RefCode != "abc123" {
		t.Fatalf("%+v", d)
	}
	d = Decide(Inbound{ChatID: "1", FromID: "2", Text: "/invite", Group: false})
	if d.Kind != KindInvite {
		t.Fatalf("%+v", d)
	}
}

func TestDecideRefuseOps(t *testing.T) {
	d := Decide(Inbound{ChatID: "1", FromID: "2", Text: "/accounts paid 12k"})
	if d.Kind != KindRefuse {
		t.Fatalf("%+v", d)
	}
	if strings.Contains(strings.ToLower(d.Reply), "ops") || strings.Contains(strings.ToLower(d.Reply), "worker") {
		t.Fatalf("students must not hear office routing: %s", d.Reply)
	}
}

func TestHelpTextIsStudentFacing(t *testing.T) {
	h := strings.ToLower(HelpText())
	for _, bad := range []string{"ops", "worker", "desk", "books", "legal"} {
		if strings.Contains(h, bad) {
			t.Fatalf("help mentions %q: %s", bad, HelpText())
		}
	}
	if strings.Contains(HelpText(), "course, lesson, and step") {
		t.Fatal("help must not demand every fact in one breath")
	}
	if !strings.Contains(strings.ToLower(HelpText()), "one thing at a time") {
		t.Fatal(HelpText())
	}
}

func TestParseMigrateToSupergroup(t *testing.T) {
	raw := []byte(`{"update_id":1,"message":{"message_id":2,"date":1,"from":{"id":9,"is_bot":true,"first_name":"Bot"},"chat":{"id":-123456789,"type":"group"},"migrate_to_chat_id":-1003968683203}}`)
	in, ok := ParseUpdate(raw)
	if !ok || in.ChatID != "-123456789" || in.MigrateTo != "-1003968683203" {
		t.Fatalf("%+v ok=%v", in, ok)
	}
}

func TestParsePhotoWithoutCaption(t *testing.T) {
	raw := []byte(`{"update_id":1,"message":{"message_id":2,"date":1,"from":{"id":9,"is_bot":false,"first_name":"Ada"},"chat":{"id":1,"type":"private"},"photo":[{"file_id":"x"}]}}`)
	in, ok := ParseUpdate(raw)
	if !ok || !in.HasPhoto || in.Group {
		t.Fatalf("%+v ok=%v", in, ok)
	}
	d := Decide(in)
	if d.Kind != KindWork || !strings.Contains(d.Task, "photo") {
		t.Fatalf("%+v", d)
	}
}
