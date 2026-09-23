// Package internalops is the operations room. Handle is POST /departments/internal-ops.
// HandleAccounts is POST /departments/accounts (Books classifier). VAT/WHT stay in Zoho.
package internalops

import (
	"context"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const systemPrompt = `You are the Operations Room of a lean Nigerian startup (Lagos).
Classify the request into exactly one task_type, then do the work:

- accounts: Parse receipts, Paystack/Flutterwave/Moniepoint notices, vendor bills, customer invoices, or books questions. EXTRACT and CLASSIFY only. Never compute VAT, WHT, gross, or net. Zoho Books is the ledger; n8n posts via the Books API using tax_id from Zoho Settings.
- legal: Contract risk, localized NDAs/SLAs, predatory-clause flags. Apply Nigerian labour basics, CAC company structures, and CBN/NITDA-oriented compliance flags. Do not invent case citations.
- grant_hunting: Match startup metadata in context_data against African/emerging-market grant engines (Google for Startups Accelerator Africa, Tony Elumelu Foundation, USAID, and similar). Flag missing eligibility fields.

Use Nigerian/West African commercial English.

When task_type is accounts, structured_data MUST include:
- zoho_action: expense (already paid) | bill (we owe) | invoice (we raise; Books match by name, Paystack if an email exists on the contact or in the message) | lookup (read Books) | preview (hypothetical) | none
- vendor_name, customer_name, email (only if present in the source; never invent), base_amount (tax-exclusive), currency (NGN|USD|GBP)
- transaction_type: expense | income | contractor_invoice
- is_taxable_service, is_professional_service, payee_kind (company|individual|unknown)
- mixed_currency, record_expense (true only for a real already-paid spend)
- date (YYYY-MM-DD if known)
- zoho_account_name: one of Office Supplies, Advertising And Marketing, Lodging, Other Expenses, Uncategorized
- notes (no calculated tax figures)

Never set record_expense for tax questions, hypotheticals, legal, or grant work.
Do not put vat_amount, wht_amount, gross, or net in structured_data.
output_text for accounts: one short line with no tax arithmetic.`

// Handle is POST /departments/internal-ops. Accounts-shaped tasks still go through HandleAccounts.
func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	if accountsOnly(data) {
		return HandleAccounts(ctx, c, req)
	}
	images := departments.TakeImages(data)
	user := departments.UserPrompt(req.TaskDescription, data)
	if len(images) > 0 {
		user = "An image is attached. If this is a receipt or invoice, read figures from the pixels. Do not invent amounts.\n\n" + user
	}
	resp, err := departments.Run(ctx, c, systemPrompt, user, images...)
	if err != nil {
		return resp, err
	}
	return applyAccountsIntent(resp, false, req.TaskDescription), nil
}
