package internalops

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const accountsExtractPrompt = `You are the autonomous Senior Accountant agent for a Nigerian technology startup.
Analyze the provided financial text, transaction log, invoice, Paystack/Flutterwave/Moniepoint notice, receipt, or books question.

You extract and classify. You NEVER compute VAT, WHT, gross, net, or any tax split.
Zoho Books is the ledger. n8n will call the Books API. Tax amounts come from Zoho tax settings (tax_id), not from you.

Return the department JSON envelope. structured_data MUST contain:
- zoho_action: one of
  - expense — already-paid spend (receipt, "I paid", card/Paystack debit). Petty cash in Books.
  - bill — we were invoiced and still owe (contractor/vendor invoice on credit)
  - invoice — we are billing a customer. n8n matches them in Books by name (a short name is enough). Paystack uses the email stored on that Books contact, or an email in this message. NEVER invent an address.
  - lookup — cash position, unpaid invoices/bills, recent spend, P&L
  - preview — hypothetical tax question, "do not record"
  - none — not a Books document, or a photo/receipt with no usable figures
- vendor_name (string, party we pay; a short or partial name is fine)
- customer_name (string, party who pays us; a short or partial name is fine — n8n resolves it in Books)
- email (string, only if the source contains one; NEVER invent; omit if unknown — Books may already have it)
- base_amount (number, tax-exclusive, a single currency)
- currency (NGN, USD, or GBP; default NGN)
- transaction_type (expense | income | contractor_invoice)
- is_taxable_service (boolean) — Books should attach VAT
- is_professional_service (boolean) — professional/consultancy/contract; WHT is applied in Books on vendor payment, not as a second ledger here
- payee_kind (company | individual | unknown)
- mixed_currency (boolean)
- record_expense (boolean) — true only for a real booked expense (already paid)
- date (YYYY-MM-DD if known)
- zoho_account_name: one of Office Supplies, Advertising And Marketing, Lodging, Other Expenses, Uncategorized
- lookup (optional: overview | unpaid | pnl | recent)
- notes (short classifier comment with NO calculated figures)

If an image is attached, read vendor/customer, amounts, dates, currency, and paid vs unpaid from the pixels. Caption text is extra context, not a substitute for the image. If the image is blurry or has no usable figures, set zoho_action to none and ask — NEVER invent amounts.

task_type must be "accounts".
output_text: one short line with no tax arithmetic.
Do not include vat_amount, wht_amount, gross, or net.`

func HandleAccounts(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	images := departments.TakeImages(data)
	user := departments.UserPrompt(req.TaskDescription, data)
	if len(images) > 0 {
		user = "An image is attached (receipt or invoice). Read figures from the image. Do not invent amounts.\n\n" + user
	}
	resp, err := departments.Run(ctx, c, accountsExtractPrompt, user, images...)
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "accounts"
	if err != nil {
		return resp, err
	}
	if resp.Status != contract.StatusSuccess {
		return resp, nil
	}
	return applyAccountsIntent(resp, true, req.TaskDescription), nil
}

func accountsOnly(data map[string]any) bool {
	dept := strings.ToLower(strings.TrimSpace(fmtString(data["department"])))
	if dept == "accounts" {
		return true
	}
	tt := strings.ToLower(strings.TrimSpace(fmtString(data["task_type"])))
	return tt == "accounts"
}

func fmtString(v any) string {
	s, _ := v.(string)
	return s
}
