package departments

import "testing"

func TestDecodeResponseJSONObject(t *testing.T) {
	raw := `{
	  "status": "success",
	  "output_text": "VAT is 7.5%.",
	  "structured_data": {"vat_rate": 0.075},
	  "task_type": "accounts"
	}`
	got, err := decodeResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "success" || got.OutputText != "VAT is 7.5%." {
		t.Fatalf("unexpected contract: %+v", got)
	}
	if got.StructuredData["task_type"] != "accounts" {
		t.Fatalf("task_type not merged: %+v", got.StructuredData)
	}
}

func TestDecodeResponseFenced(t *testing.T) {
	raw := "```json\n{\"status\":\"failed\",\"output_text\":\"no\",\"structured_data\":{}}\n```"
	got, err := decodeResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" {
		t.Fatalf("status=%s", got.Status)
	}
}
