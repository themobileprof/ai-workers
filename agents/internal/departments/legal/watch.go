package legal

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const watchOverlay = `You are the Legal clerk for TheMobileProf Technologies (Lagos), watching the ops Telegram room.
You do not take over the thread. Another department already answered. You only speak when the plan has a material legal, regulatory, or contract gotcha.
You are not a lawyer. Do not invent case citations, section numbers, licence numbers, or "CBN said".
Do not draft a contract on a watch. Do not email. cannot_send is always true. save_draft is always false.

Speak when any of these are in play (Nigerian / West African context):
- labour: hire, salary, PAYE, intern vs contractor, firing, pension
- company: cofounder treated as director, second CAC company, equity, spin-out
- IP / brand: who owns work product, trademarks, customer lists
- data: NDPR / NITDA, personal data, BVN, KYC dumps on WhatsApp
- payments / finance: CBN-ish activity (taking deposits, wallets, investment advice, AUM) without saying you are not a bank
- tax process: WHT on professional fees, treating a person as a vendor vs staff
- consumer / ads: guarantees, "licensed", refunds, unsolicited WhatsApp
- a promise that should be in a written TMP-owned contract before anyone relies on it

Stay silent when it is a greeting, a routine retail receipt, a cash lookup, or a hypothetical with no commitment.

If you speak:
- 1–4 short Telegram lines. No markdown headings.
- Name the gotcha and the safer next step (desk Ask draft, human lawyer, do not hold out as a separate company).
- If the plan can be made compliant by changing how you do it, say that change. Do not write the contract.
If you are silent: output_text empty, speak false, severity none.

structured_data MUST include:
- action: watch
- speak: boolean
- severity: none | note | caution | stop
- flags: string[] (empty when silent)
- topics: string[] (labour, company, ip, data, payments, tax, consumer, contract)
- needs_human_lawyer: boolean
- save_draft: false
- cannot_send: true
stop means a human should pause before relying on the plan. You cannot halt Books or n8n.`

func handleWatch(ctx context.Context, c llm.Completer, req contract.Request, data map[string]any) (contract.Response, error) {
	silent := func() contract.Response {
		return contract.Response{
			Status:     contract.StatusSuccess,
			OutputText: "",
			StructuredData: map[string]any{
				"task_type":          "legal",
				"action":             "watch",
				"speak":              false,
				"severity":           "none",
				"flags":              []string{},
				"topics":             []string{},
				"needs_human_lawyer": false,
				"save_draft":         false,
				"cannot_send":        true,
				"prompt_version":     "v1",
			},
		}
	}
	if !watchWorthy(req.TaskDescription, data) {
		return silent(), nil
	}
	resp, err := departments.Run(ctx, c, watchOverlay, departments.UserPrompt(req.TaskDescription, data))
	if err != nil {
		// Watch must not fail the ops reply.
		out := silent()
		out.StructuredData["watch_error"] = err.Error()
		return out, nil
	}
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	speak := boolish(resp.StructuredData["speak"])
	text := strings.TrimSpace(resp.OutputText)
	if text == "" {
		speak = false
	}
	if !speak {
		resp.OutputText = ""
		resp.StructuredData["speak"] = false
		if str(resp.StructuredData["severity"]) == "" {
			resp.StructuredData["severity"] = "none"
		}
	} else {
		resp.OutputText = text
		resp.StructuredData["speak"] = true
		sev := strings.ToLower(str(resp.StructuredData["severity"]))
		if sev != "caution" && sev != "stop" {
			resp.StructuredData["severity"] = "note"
		}
	}
	resp.Status = contract.StatusSuccess
	resp.StructuredData["task_type"] = "legal"
	resp.StructuredData["action"] = "watch"
	resp.StructuredData["cannot_send"] = true
	resp.StructuredData["save_draft"] = false
	resp.StructuredData["prompt_version"] = "v1"
	return resp, nil
}

func watchWorthy(task string, data map[string]any) bool {
	if data == nil {
		data = map[string]any{}
	}
	lower := strings.ToLower(strings.Join([]string{
		task,
		str(data["primary_output"]),
		str(data["primary_task_type"]),
		str(data["primary_zoho_action"]),
		str(data["primary_department"]),
	}, " "))
	if blob, ok := data["recent_messages"].([]any); ok {
		for _, x := range blob {
			if s, ok := x.(string); ok {
				lower += " " + strings.ToLower(s)
			}
		}
	}
	needles := []string{
		"hire", "hiring", "employ", "employee", "salary", "paye", "contractor", "consult",
		"intern", "staff", "pension", "terminate", "fired", "offer letter",
		"cofounder", "co-founder", "director", "equity", "shareholder", "shares",
		"cac", "subsidiary", "spin out", "spin-off", "second company",
		"nda", "non-disclosure", "agreement", "contract", "terms of", " sla", "sla ",
		"intellectual", "trademark", "copyright", "licence", "license", "work product",
		"ndpr", "nitda", "cbn", "data protection", "personal data", "bvn", "kyc",
		"loan", "interest rate", "taking deposits", "deposit-taking", "investment advice", "fund ", "securities", "aum",
		"wallet", "payments licence", "mortgage",
		"firs", "withhold", "wht", "withholding",
		"we guarantee", "licensed", "refund policy", "unsolicited",
		"they signed", "we promised", "we will pay", "term sheet", "investor",
		"is this legal", "compliant", "compliance", "regulator", "gotcha",
		"project face", "customer terms", "pilot terms",
		"professional service",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	action := strings.ToLower(str(data["primary_zoho_action"]))
	return action == "invoice" || action == "bill"
}

func boolish(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}
