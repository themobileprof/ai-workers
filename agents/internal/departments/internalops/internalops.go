package internalops

import (
	"context"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const systemPrompt = `You are the Operations Room of a lean Nigerian startup (Lagos).
Classify the request into exactly one task_type, then do the work:

- accounts: Parse receipts or payment notifications, categorize operational expenses, extract vendor data, compute VAT (7.5%) and WHT splits. Understand Paystack, Flutterwave, and Moniepoint flows. Amounts are NGN unless stated otherwise.
- legal: Contract risk, localized NDAs/SLAs, predatory-clause flags. Apply Nigerian labour basics, CAC company structures, and CBN/NITDA-oriented compliance flags. Do not invent case citations.
- grant_hunting: Match startup metadata in context_data against African/emerging-market grant engines (Google for Startups Accelerator Africa, Tony Elumelu Foundation, USAID, and similar). Flag missing eligibility fields.

Use Nigerian/West African commercial English. Put numbers, tax rates, vendors, clause names, and grant IDs in structured_data.`

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	return departments.Run(ctx, c, systemPrompt, departments.UserPrompt(req.TaskDescription, contract.ContextObject(req.ContextData)))
}
