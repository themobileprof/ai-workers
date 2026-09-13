package internalops

import (
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func TestPaidReceiptMapsToExpense(t *testing.T) {
	e := Extract{
		VendorName:       "Shoprite",
		BaseMinor:        10_000_000,
		Currency:         "NGN",
		TransactionType:  "expense",
		RecordExpense:    true,
		IsTaxableService: true,
	}
	action, skip := resolveAction(e, "I paid Shoprite 100000 for office supplies")
	if action != actionExpense || skip != "" {
		t.Fatalf("action=%s skip=%s", action, skip)
	}
}

func TestContractorUnpaidMapsToBill(t *testing.T) {
	e := Extract{
		VendorName:            "Acme Consulting Ltd",
		BaseMinor:             10_000_000,
		Currency:              "NGN",
		TransactionType:       "contractor_invoice",
		RecordExpense:         true,
		IsProfessionalService: true,
		ZohoAction:            "bill",
	}
	action, skip := resolveAction(e, "Record this contractor invoice from Acme Consulting Ltd for 100000")
	if action != actionBill || skip != "" {
		t.Fatalf("action=%s skip=%s", action, skip)
	}
}

func TestIncomeInvoiceNeedsCustomer(t *testing.T) {
	e := Extract{
		BaseMinor:       5_000_000,
		Currency:        "NGN",
		TransactionType: "income",
		RecordExpense:   true,
		ZohoAction:      "invoice",
		CustomerName:    "Lagos Motors",
	}
	action, skip := resolveAction(e, "Raise an invoice to Lagos Motors for 50000")
	if action != actionInvoice || skip != "" {
		t.Fatalf("action=%s skip=%s", action, skip)
	}
}

func TestHypotheticalIsPreviewNotPost(t *testing.T) {
	e := Extract{
		VendorName:       "Acme Consulting Ltd",
		BaseMinor:        10_000_000,
		Currency:         "NGN",
		TransactionType:  "contractor_invoice",
		IsTaxableService: true,
		RecordExpense:    false,
	}
	action, _ := resolveAction(e, "Hypothetical only, do not record: what is VAT and WHT on 100000")
	if action != actionPreview {
		t.Fatalf("action=%s", action)
	}
}

func TestLookupCashPosition(t *testing.T) {
	e := Extract{Currency: "NGN", LookupKind: "overview"}
	action, skip := resolveAction(e, "What is our cash position and unpaid bills?")
	if action != actionLookup || skip != "" {
		t.Fatalf("action=%s skip=%s", action, skip)
	}
}

func TestUSDDoesNotPost(t *testing.T) {
	e := Extract{
		BaseMinor:     10_000_00,
		Currency:      "USD",
		RecordExpense: true,
		ZohoAction:    "expense",
	}
	action, skip := resolveAction(e, "I paid $100 for AWS")
	if action != actionNone || skip == "" {
		t.Fatalf("action=%s skip=%s", action, skip)
	}
}

func TestApplyIntentStripsLLMTaxAndSetsZohoAction(t *testing.T) {
	resp := contract.Response{
		Status:     contract.StatusSuccess,
		OutputText: "VAT is 99999 because I guessed.",
		StructuredData: map[string]any{
			"task_type":               "accounts",
			"vendor_name":             "Acme Consulting Ltd",
			"base_amount":             100000.0,
			"currency":                "NGN",
			"transaction_type":        "contractor_invoice",
			"is_taxable_service":      true,
			"is_professional_service": true,
			"payee_kind":              "company",
			"record_expense":          true,
			"zoho_action":             "bill",
			"vat_amount":              99999.0,
			"wht_amount":              1.0,
			"zoho_account_name":       "Other Expenses",
		},
	}
	got := applyAccountsIntent(resp, false, "Record this contractor invoice from Acme Consulting Ltd for 100000")
	if got.StructuredData["zoho_action"] != "bill" {
		t.Fatalf("zoho_action=%v", got.StructuredData["zoho_action"])
	}
	if got.StructuredData["tax_engine"] != "zoho_books" {
		t.Fatal("tax_engine")
	}
	if got.StructuredData["record_bill"] != true {
		t.Fatal("record_bill")
	}
	if got.StructuredData["record_expense"] != false {
		t.Fatal("expense should be false for unpaid contractor")
	}
	if _, ok := got.StructuredData["vat_amount"]; ok {
		t.Fatal("vat_amount must not be computed here")
	}
	if strings.Contains(got.OutputText, "99999") {
		t.Fatalf("LLM tax leaked: %s", got.OutputText)
	}
	if !strings.Contains(got.OutputText, "WILL POST a vendor bill") {
		t.Fatalf("missing bill notice: %s", got.OutputText)
	}
}

func TestInvoiceEmailPassthrough(t *testing.T) {
	resp := contract.Response{
		Status: contract.StatusSuccess,
		StructuredData: map[string]any{
			"task_type":     "accounts",
			"zoho_action":   "invoice",
			"customer_name": "Apex Motors",
			"base_amount":   250000.0,
			"currency":      "NGN",
			"email":         "billing@apexmotors.ng",
		},
	}
	got := applyAccountsIntent(resp, true, "Raise an invoice to Apex Motors 250000 NGN billing@apexmotors.ng")
	if got.StructuredData["email"] != "billing@apexmotors.ng" {
		t.Fatalf("email=%v", got.StructuredData["email"])
	}
	if got.StructuredData["customer_email"] != "billing@apexmotors.ng" {
		t.Fatalf("customer_email=%v", got.StructuredData["customer_email"])
	}
	if !strings.Contains(got.OutputText, "billing@apexmotors.ng") {
		t.Fatalf("audit missing email: %s", got.OutputText)
	}
}

func TestInvoiceJunkEmailDropped(t *testing.T) {
	resp := contract.Response{
		Status: contract.StatusSuccess,
		StructuredData: map[string]any{
			"task_type":     "accounts",
			"zoho_action":   "invoice",
			"customer_name": "Apex Motors",
			"base_amount":   250000.0,
			"currency":      "NGN",
			"email":         "not-an-email",
		},
	}
	got := applyAccountsIntent(resp, true, "Raise an invoice to Apex Motors 250000 NGN")
	if _, ok := got.StructuredData["email"]; ok {
		t.Fatalf("junk email kept: %v", got.StructuredData["email"])
	}
	if !strings.Contains(got.OutputText, "No customer email") {
		t.Fatalf("audit: %s", got.OutputText)
	}
}

func TestLegalPathUntouched(t *testing.T) {
	resp := contract.Response{
		Status:     contract.StatusSuccess,
		OutputText: "NDA looks fine.",
		StructuredData: map[string]any{
			"task_type": "legal",
			"clause":    "termination",
		},
	}
	got := applyAccountsIntent(resp, false, "review this NDA")
	if got.OutputText != "NDA looks fine." {
		t.Fatalf("legal output rewritten: %s", got.OutputText)
	}
	if _, ok := got.StructuredData["tax_engine"]; ok {
		t.Fatal("tax_engine on legal")
	}
}
