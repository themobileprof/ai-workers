package contract

import (
	"encoding/json"
	"testing"
)

func TestFail(t *testing.T) {
	got := Fail("task_description is required")
	if got.Status != StatusFailed || got.OutputText != "task_description is required" {
		t.Fatalf("%+v", got)
	}
	if got.StructuredData == nil || len(got.StructuredData) != 0 {
		t.Fatalf("structured_data %+v", got.StructuredData)
	}
}

func TestContextObject(t *testing.T) {
	if m := ContextObject(nil); len(m) != 0 {
		t.Fatalf("nil %+v", m)
	}
	if m := ContextObject(json.RawMessage("null")); len(m) != 0 {
		t.Fatalf("null %+v", m)
	}
	obj := ContextObject(json.RawMessage(`{"channel":"whatsapp"}`))
	if obj["channel"] != "whatsapp" {
		t.Fatalf("%+v", obj)
	}
	raw := ContextObject(json.RawMessage(`["not","an","object"]`))
	if _, ok := raw["_raw"]; !ok {
		t.Fatalf("array should keep _raw %+v", raw)
	}
}
