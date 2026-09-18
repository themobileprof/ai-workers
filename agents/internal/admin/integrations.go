package admin

// Integration is one third-party system this office talks to.
// When you add a vendor (new env var, n8n credential, or outbound API),
// add a row here. integrations_test.go fails the build if compose/.env.example
// or n8n credentials mention a tool that is missing from this list.
type Integration struct {
	ID           string
	Name         string
	URL          string
	Role         string
	Where        string
	Secrets      string
	Status       string
	ComposeEnvs  []string
	N8nCredNames []string
	DeskKeys     []string
}

const (
	statusLive     = "live"
	statusOptional = "code-ready"
	statusHost     = "host"
)

func allIntegrations() []Integration {
	return []Integration{
		{
			ID:      "n8n",
			Name:    "n8n",
			URL:     "https://n8n.io/",
			Role:    "Orchestration: webhooks, cron, visual routing, execution logs. Public /webhook* and the editor at /home.",
			Where:   "Docker service n8n. Desk iframes /home.",
			Secrets: "N8N_ENCRYPTION_KEY and N8N_LICENSE_ACTIVATION_KEY on the VM .env. Per-app tokens live in n8n Credentials, not git.",
			Status:  statusLive,
			ComposeEnvs: []string{
				"N8N_ENCRYPTION_KEY", "N8N_LICENSE_ACTIVATION_KEY", "N8N_HOST", "N8N_PROTOCOL", "N8N_WEBHOOK_URL",
			},
		},
		{
			ID:      "postgres",
			Name:    "PostgreSQL",
			URL:     "https://www.postgresql.org/",
			Role:    "Two databases on the host: n8n (orchestration) and aiworkers (company desk).",
			Where:   "Native systemd on the VM. Never a Docker container.",
			Secrets: "POSTGRES_* for n8n. ADMIN_DATABASE_URL for the desk. Not in git.",
			Status:  statusHost,
			ComposeEnvs: []string{
				"POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD", "ADMIN_DATABASE_URL",
			},
		},
		{
			ID:      "caddy",
			Name:    "Caddy",
			URL:     "https://caddyserver.com/",
			Role:    "TLS on 443. / and /site* plus /admin* → agents. /webhook* → n8n. /n8n/ redirects to /home.",
			Where:   "Native on the host. Config in Caddyfile (repo) and /etc/caddy/Caddyfile (live).",
			Secrets: "None in git. ACME via the public hostname.",
			Status:  statusHost,
		},
		{
			ID:           "zoho-books",
			Name:         "Zoho Books",
			URL:          "https://www.zoho.com/books/",
			Role:         "Ledger. Expenses, bills, invoices, contacts, customer payments, tax_id. Go never computes VAT/WHT.",
			Where:        "n8n OAuth (Books write, Paystack paid, CRM upsert). Org ids and chart-of-accounts ids on the desk Defaults.",
			Secrets:      "OAuth tokens in n8n credential Zoho Books. Never .env, never the Go worker.",
			Status:       statusLive,
			N8nCredNames: []string{"Zoho Books"},
			DeskKeys: []string{
				"zoho.organization_id",
				"zoho.paid_through_account_id",
				"zoho.default_expense_account_id",
				"zoho.account.office_supplies_id",
				"zoho.account.advertising_id",
				"zoho.account.lodging_id",
				"zoho.account.uncategorized_id",
				"zoho.deposit_to_account_id",
			},
		},
		{
			ID:           "zoho-mail",
			Name:         "Zoho Mail",
			URL:          "https://www.zoho.com/mail/",
			Role:         "Company mailbox info@themobileprof.com. SMTP out (approved drafts), IMAP in (growth drafts).",
			Where:        "n8n credentials. SMTP smtppro.zoho.com:465. IMAP imappro.zoho.com:993.",
			Secrets:      "App password in n8n only.",
			Status:       statusLive,
			N8nCredNames: []string{"Zoho Mail info@", "Zoho Mail info@ IMAP"},
		},
		{
			ID:          "paystack",
			Name:        "Paystack",
			URL:         "https://paystack.com/",
			Role:        "Collect invoice payments (Payment Request). Webhook verifies then n8n books customerpayments in Zoho.",
			Where:       "n8n only. Webhook https://workers.themobileprof.com/webhook/paystack-paid. Deposit account id on the desk.",
			Secrets:     "PAYSTACK_SECRET_KEY on the VM .env → n8n container. Never git. Never the Go worker.",
			Status:      statusLive,
			ComposeEnvs: []string{"PAYSTACK_SECRET_KEY"},
			DeskKeys:    []string{"paystack.currency", "zoho.deposit_to_account_id"},
		},
		{
			ID:           "telegram",
			Name:         "Telegram",
			URL:          "https://core.telegram.org/",
			Role:         "Ops room: prefixes including /hr and /legal, receipt photos, /approve and /kill for info@ drafts.",
			Where:        "n8n Ops Telegram workflow.",
			Secrets:      "Bot token in n8n credential Telegram account.",
			Status:       statusLive,
			N8nCredNames: []string{"Telegram account"},
		},
		{
			ID:           "whatsapp",
			Name:         "WhatsApp (Meta Cloud API)",
			URL:          "https://developers.facebook.com/docs/whatsapp/cloud-api",
			Role:         "Customer and community chat. /accounts, /ops, /legal, and /hr are desk-allowlisted. Receipt OCR on allowlisted /accounts photos.",
			Where:        "n8n Customer WhatsApp. Allowlist from desk People (whatsapp_accounts).",
			Secrets:      "Tokens in n8n WhatsApp credentials.",
			Status:       statusLive,
			N8nCredNames: []string{"WhatsApp OAuth account", "WhatsApp account"},
		},
		{
			ID:          "deepseek",
			Name:        "DeepSeek",
			URL:         "https://www.deepseek.com/",
			Role:        "Default text LLM for department workers (classifier, chat). Text-only — no OCR.",
			Where:       "Go agents via LLM_PROVIDER=deepseek.",
			Secrets:     "DEEPSEEK_API_KEY (or LLM_API_KEY) on the VM .env → agents container.",
			Status:      statusLive,
			ComposeEnvs: []string{"LLM_PROVIDER", "LLM_MODEL", "LLM_API_KEY", "LLM_BASE_URL", "DEEPSEEK_API_KEY", "DEEPSEEK_MODEL", "DEEPSEEK_BASE_URL"},
		},
		{
			ID:          "gemini",
			Name:        "Google Gemini",
			URL:         "https://ai.google.dev/",
			Role:        "Vision OCR for receipt/invoice photos. Default model gemini-3.6-flash. Does not invent amounts when the image is unreadable.",
			Where:       "Go agents when context_data includes image_base64. n8n downloads Telegram/WhatsApp photos.",
			Secrets:     "GEMINI_API_KEY on the VM .env → agents container.",
			Status:      statusLive,
			ComposeEnvs: []string{"GEMINI_API_KEY", "GEMINI_MODEL", "GEMINI_BASE_URL"},
		},
		{
			ID:          "openai",
			Name:        "OpenAI",
			URL:         "https://platform.openai.com/",
			Role:        "Optional text LLM. Set LLM_PROVIDER=openai and OPENAI_API_KEY (or LLM_API_KEY).",
			Where:       "Go Completer. Not live unless the env says so.",
			Secrets:     "OPENAI_API_KEY / LLM_API_KEY on the VM .env.",
			Status:      statusOptional,
			ComposeEnvs: []string{"OPENAI_API_KEY", "OPENAI_MODEL", "OPENAI_BASE_URL"},
		},
		{
			ID:          "anthropic",
			Name:        "Anthropic",
			URL:         "https://www.anthropic.com/",
			Role:        "Optional text LLM. Set LLM_PROVIDER=anthropic and ANTHROPIC_API_KEY.",
			Where:       "Go Completer. Not live unless the env says so.",
			Secrets:     "ANTHROPIC_API_KEY on the VM .env.",
			Status:      statusOptional,
			ComposeEnvs: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "ANTHROPIC_BASE_URL"},
		},
	}
}

func integrationByID(id string) (Integration, bool) {
	for _, it := range allIntegrations() {
		if it.ID == id {
			return it, true
		}
	}
	return Integration{}, false
}
