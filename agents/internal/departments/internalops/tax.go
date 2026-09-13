package internalops

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

const (
	actionNone    = "none"
	actionExpense = "expense"
	actionBill    = "bill"
	actionInvoice = "invoice"
	actionLookup  = "lookup"
	actionPreview = "preview"
)

// Extract is the LLM-classified payload. Zoho Books owns tax math and ledgers.
type Extract struct {
	VendorName            string
	CustomerName          string
	BaseMinor             int64
	Currency              string
	TransactionType       string
	IsTaxableService      bool
	IsProfessionalService bool
	PayeeKind             string
	MixedCurrency         bool
	RecordExpense         bool
	Date                  string
	ZohoAccountName       string
	Notes                 string
	ZohoAction            string
	LookupKind            string
	Reference             string
}

func parseExtract(sd map[string]any) Extract {
	if sd == nil {
		sd = map[string]any{}
	}
	e := Extract{
		VendorName:            firstString(sd, "vendor_name", "vendor"),
		CustomerName:          firstString(sd, "customer_name", "customer"),
		Currency:              normalizeCurrency(firstString(sd, "currency")),
		TransactionType:       normalizeTxn(firstString(sd, "transaction_type")),
		IsTaxableService:      asBool(sd["is_taxable_service"]),
		IsProfessionalService: asBool(sd["is_professional_service"]),
		PayeeKind:             normalizePayee(firstString(sd, "payee_kind", "entity_type", "payee_type")),
		MixedCurrency:         asBool(sd["mixed_currency"]),
		RecordExpense:         asBool(sd["record_expense"]),
		Date:                  firstString(sd, "date"),
		ZohoAccountName:       firstString(sd, "zoho_account_name"),
		Notes:                 firstString(sd, "notes"),
		ZohoAction:            strings.ToLower(strings.TrimSpace(firstString(sd, "zoho_action"))),
		LookupKind:            strings.ToLower(strings.TrimSpace(firstString(sd, "lookup", "lookup_kind"))),
		Reference:             firstString(sd, "reference", "reference_number"),
	}
	if major, ok := asFloat(sd["base_amount"]); ok {
		e.BaseMinor = toMinor(major)
	} else if major, ok := asFloat(sd["amount"]); ok {
		e.BaseMinor = toMinor(major)
	}
	if e.Currency == "" {
		e.Currency = "NGN"
	}
	if e.TransactionType == "" {
		e.TransactionType = "expense"
	}
	if e.PayeeKind == "" {
		e.PayeeKind = "unknown"
	}
	if e.CustomerName == "" && e.TransactionType == "income" {
		e.CustomerName = e.VendorName
	}
	return e
}

func resolveAction(e Extract, task string) (action, skip string) {
	task = strings.ToLower(task)
	if e.MixedCurrency {
		return actionNone, "mixed currencies in the source text — not posting until a single currency is confirmed"
	}
	if allowedAction(e.ZohoAction) {
		action = e.ZohoAction
	} else {
		action = inferAction(e, task)
	}
	switch action {
	case actionLookup:
		return actionLookup, ""
	case actionPreview:
		if e.BaseMinor <= 0 {
			return actionNone, "no numeric amount to preview against Zoho tax settings"
		}
		return actionPreview, ""
	case actionExpense, actionBill, actionInvoice:
		if e.BaseMinor <= 0 {
			return actionNone, "no numeric base amount extracted"
		}
		if e.Currency != "NGN" {
			return actionNone, "Zoho org is NGN — non-NGN documents are not auto-posted"
		}
		if action == actionInvoice && strings.TrimSpace(e.CustomerName) == "" && strings.TrimSpace(e.VendorName) == "" {
			return actionNone, "invoice needs a customer name"
		}
		if action == actionBill && strings.TrimSpace(e.VendorName) == "" {
			return actionNone, "bill needs a vendor name"
		}
		return action, ""
	default:
		if skip != "" {
			return actionNone, skip
		}
		return actionNone, "not a Zoho Books write or lookup"
	}
}

func inferAction(e Extract, task string) string {
	if lookupRequested(task, e.LookupKind) {
		return actionLookup
	}
	if previewRequested(task, e) {
		return actionPreview
	}
	if e.TransactionType == "income" {
		if e.RecordExpense || strings.Contains(task, "invoice") || strings.Contains(task, "raise") {
			return actionInvoice
		}
		return actionNone
	}
	if !e.RecordExpense {
		return actionNone
	}
	if e.TransactionType == "contractor_invoice" && !alreadyPaid(task) {
		return actionBill
	}
	return actionExpense
}

func lookupRequested(task, kind string) bool {
	switch kind {
	case "pnl", "unpaid", "recent", "cash", "overview":
		return true
	}
	keys := []string{
		"p&l", "profit and loss", "profit & loss", "cash position",
		"outstanding", "unpaid", "what do we owe", "who owes us",
		"recent expenses", "what did we spend", "balance sheet",
		"books summary", "show invoices", "show bills",
	}
	for _, k := range keys {
		if strings.Contains(task, k) {
			return true
		}
	}
	return false
}

func previewRequested(task string, e Extract) bool {
	if e.RecordExpense {
		return false
	}
	if strings.Contains(task, "do not record") || strings.Contains(task, "don't record") ||
		strings.Contains(task, "hypothetical") || strings.Contains(task, "do not post") {
		return true
	}
	if e.BaseMinor > 0 && (strings.Contains(task, "what is vat") || strings.Contains(task, "what's vat") ||
		strings.Contains(task, "what is wht") || strings.Contains(task, "tax on")) {
		return true
	}
	return false
}

func alreadyPaid(task string) bool {
	keys := []string{"i paid", "we've paid", "we paid", "receipt", "paystack", "flutterwave", "moniepoint", "debit alert"}
	for _, k := range keys {
		if strings.Contains(task, k) {
			return true
		}
	}
	return false
}

func allowedAction(s string) bool {
	switch s {
	case actionNone, actionExpense, actionBill, actionInvoice, actionLookup, actionPreview:
		return true
	default:
		return false
	}
}

func applyAccountsIntent(resp contract.Response, force bool, task string) contract.Response {
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	if resp.Status != contract.StatusSuccess {
		return resp
	}
	taskType := strings.ToLower(strings.TrimSpace(fmt.Sprint(resp.StructuredData["task_type"])))
	if !force && taskType != "accounts" {
		return resp
	}
	resp.StructuredData["task_type"] = "accounts"
	ext := parseExtract(resp.StructuredData)
	action, skip := resolveAction(ext, task)
	overlayAccounts(resp.StructuredData, ext, action, skip)
	resp.OutputText = formatAudit(ext, action, skip, resp.OutputText)
	return resp
}

func overlayAccounts(sd map[string]any, e Extract, action, skip string) {
	sd["tax_engine"] = "zoho_books"
	sd["zoho_action"] = action
	sd["vendor"] = e.VendorName
	sd["vendor_name"] = e.VendorName
	if e.CustomerName != "" {
		sd["customer_name"] = e.CustomerName
	}
	sd["currency"] = e.Currency
	sd["transaction_type"] = e.TransactionType
	sd["is_taxable_service"] = e.IsTaxableService
	sd["is_professional_service"] = e.IsProfessionalService
	sd["payee_kind"] = e.PayeeKind
	sd["mixed_currency"] = e.MixedCurrency
	sd["base_amount"] = fromMinor(e.BaseMinor)
	sd["amount"] = fromMinor(e.BaseMinor)
	sd["record_expense"] = action == actionExpense
	sd["record_bill"] = action == actionBill
	sd["record_invoice"] = action == actionInvoice
	if e.LookupKind != "" {
		sd["lookup"] = e.LookupKind
	} else if action == actionLookup {
		sd["lookup"] = "overview"
	}
	if e.Date != "" {
		sd["date"] = e.Date
	} else {
		delete(sd, "date")
	}
	if e.ZohoAccountName != "" {
		sd["zoho_account_name"] = e.ZohoAccountName
	}
	if e.Reference != "" {
		sd["reference"] = e.Reference
	}
	delete(sd, "vat_amount")
	delete(sd, "wht_amount")
	delete(sd, "gross_amount")
	delete(sd, "net_payable")
	delete(sd, "vat_rate")
	delete(sd, "wht_rate")
	delete(sd, "wht_assumed")
	delete(sd, "expense_description")
	if skip != "" {
		sd["post_skip_reason"] = skip
	} else {
		delete(sd, "post_skip_reason")
	}
}

func formatAudit(e Extract, action, skip, llmNote string) string {
	var b strings.Builder
	b.WriteString("ACCOUNTS — classifier only. Zoho Books owns VAT, WHT, and the ledger.\n")
	if e.VendorName != "" {
		b.WriteString("Party: ")
		b.WriteString(e.VendorName)
		b.WriteByte('\n')
	}
	if e.CustomerName != "" && e.CustomerName != e.VendorName {
		b.WriteString("Customer: ")
		b.WriteString(e.CustomerName)
		b.WriteByte('\n')
	}
	if e.BaseMinor > 0 {
		b.WriteString("Base (ex-tax, extracted): ")
		b.WriteString(fmtMoney(e.BaseMinor, e.Currency))
		b.WriteByte('\n')
	}
	b.WriteString("Type: ")
	b.WriteString(e.TransactionType)
	if e.IsTaxableService {
		b.WriteString(" · taxable")
	}
	if e.IsProfessionalService {
		b.WriteString(" · professional (withhold on payment in Books)")
	}
	b.WriteByte('\n')
	b.WriteByte('\n')
	switch action {
	case actionExpense:
		acct := e.ZohoAccountName
		if acct == "" {
			acct = "Other Expenses"
		}
		b.WriteString("Zoho Books: WILL POST a paid expense to ")
		b.WriteString(acct)
		b.WriteString(". Tax comes from Books Settings, not this worker.")
	case actionBill:
		b.WriteString("Zoho Books: WILL POST a vendor bill (we owe). Not marked paid. WHT is applied when you record the vendor payment in Books.")
	case actionInvoice:
		b.WriteString("Zoho Books: WILL CREATE a draft customer invoice. Not emailed.")
	case actionLookup:
		b.WriteString("Zoho Books: WILL READ unpaid invoices/bills, recent expenses, and cash P&L.")
	case actionPreview:
		b.WriteString("Zoho Books: preview only — tax % from Settings, nothing posted.")
	default:
		b.WriteString("Zoho Books: not writing. ")
		if skip != "" {
			b.WriteString(skip)
			b.WriteByte('.')
		} else {
			b.WriteString("Not a booked document.")
		}
	}
	note := strings.TrimSpace(e.Notes)
	if note == "" {
		note = strings.TrimSpace(stripTaxGuesses(llmNote))
	}
	if len(note) > 400 {
		note = note[:400] + "…"
	}
	if note != "" {
		b.WriteString("\n\nNote: ")
		b.WriteString(note)
	}
	return strings.TrimSpace(b.String())
}

func stripTaxGuesses(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "vat") || strings.Contains(lower, "wht") || strings.Contains(lower, "withholding") {
		return ""
	}
	return s
}

func toMinor(major float64) int64 {
	if major <= 0 || math.IsNaN(major) || math.IsInf(major, 0) {
		return 0
	}
	return int64(math.Round(major * 100))
}

func fromMinor(minor int64) float64 {
	return float64(minor) / 100.0
}

func fmtMoney(minor int64, currency string) string {
	cur := currency
	if cur == "" {
		cur = "NGN"
	}
	neg := ""
	if minor < 0 {
		neg = "-"
		minor = -minor
	}
	return fmt.Sprintf("%s%s %d.%02d", neg, cur, minor/100, minor%100)
}

func normalizeCurrency(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.TrimPrefix(s, "₦")
	switch {
	case s == "" || s == "NAIRA" || s == "NGN" || s == "N":
		return "NGN"
	case strings.Contains(s, "USD") || s == "DOLLAR" || s == "DOLLARS" || s == "$" || s == "US$":
		return "USD"
	case strings.Contains(s, "GBP") || s == "POUND" || s == "POUNDS" || s == "STERLING" || s == "£":
		return "GBP"
	default:
		if len(s) == 3 {
			return s
		}
		return "NGN"
	}
}

func normalizeTxn(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	switch s {
	case "income", "revenue", "inflow":
		return "income"
	case "contractor_invoice", "contractor", "consultancy", "invoice":
		return "contractor_invoice"
	case "expense", "spend", "payment", "receipt":
		return "expense"
	default:
		return s
	}
}

func normalizePayee(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "individual" || s == "person" || s == "sole_proprietor" || s == "freelancer":
		return "individual"
	case s == "company" || s == "corporate" || s == "ltd" || s == "llc" || s == "entity":
		return "company"
	default:
		return "unknown"
	}
}

func firstString(sd map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := sd[k]; ok {
			if t, ok := v.(string); ok {
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "yes" || s == "1"
	case float64:
		return t != 0
	default:
		return false
	}
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return 0, false
		}
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case jsonNumber:
		f, err := t.Float64()
		return f, err == nil
	case string:
		s := strings.TrimSpace(t)
		s = strings.ReplaceAll(s, ",", "")
		s = strings.TrimPrefix(s, "₦")
		s = strings.TrimPrefix(s, "NGN")
		s = strings.TrimPrefix(s, "$")
		s = strings.TrimPrefix(s, "£")
		s = strings.TrimSpace(s)
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

type jsonNumber interface {
	Float64() (float64, error)
}
