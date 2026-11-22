package growth

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func TestLooksLikeGlance(t *testing.T) {
	if LooksLikeGlance(map[string]any{"channel": "whatsapp"}, "how are the ads doing in GA4") {
		t.Fatal("customer WhatsApp stays sales")
	}
	if !LooksLikeGlance(map[string]any{"channel": "telegram"}, "How are the HomeGauge ads doing in GA4?") {
		t.Fatal("ops glance")
	}
	if LooksLikeGlance(map[string]any{"channel": "desk"}, "Search ads for HomeGauge") {
		t.Fatal("ads pack is not a glance")
	}
}

func TestHandleGlanceDoesNotCallSpend(t *testing.T) {
	resp, err := Handle(context.Background(), stubLLM{text: `{"status":"success","output_text":"PLACE NOW","structured_data":{"task_type":"ads","cannot_spend":false}}`}, contract.Request{
		TaskDescription: "How are the HomeGauge ads doing in GA4?",
		ContextData:     []byte(`{"channel":"desk","project":{"slug":"homegauge"},"ads_ga4":{"found":true,"paid":[{"campaign":"homegauge","content":"lagos-mortgage","sessions":12,"engaged":8}],"funnel":[{"name":"Session","users":10},{"name":"Landing","users":6}]}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["task_type"] != "ads_glance" || resp.StructuredData["cannot_spend"] != true || resp.StructuredData["save_deck"] != false {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if strings.Contains(resp.OutputText, "PLACE NOW") || !strings.Contains(resp.OutputText, "12 sessions") {
		t.Fatal(resp.OutputText)
	}
}

func TestFormatAdsGlanceEmpty(t *testing.T) {
	if FormatAdsGlance(nil) != "" || FormatAdsGlance(map[string]any{"found": false}) != "" {
		t.Fatal("empty")
	}
}
