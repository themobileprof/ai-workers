# Third-party tools

Every system this office talks to. Add a section here the day you wire a new vendor — `go test ./internal/admin` fails if compose, `.env.example`, or an n8n credential shows up without a row.

Secrets stay in the VM `.env` or n8n Credentials, never git. Accounts / Books / Paystack **ids** live on desk **Defaults** (`/admin/settings`), not in this file and not in n8n JSON.

## Live

### n8n

- **Id:** `n8n`
- **URL:** https://n8n.io/
- **Role:** Orchestration: webhooks, cron, visual routing, execution logs. Public `/webhook*` and the editor at `/home`.
- **Where:** Docker service n8n. Desk iframes `/home`.
- **Secrets:** `N8N_ENCRYPTION_KEY` and `N8N_LICENSE_ACTIVATION_KEY` on the VM `.env`. Per-app tokens live in n8n Credentials, not git.
- **Env:** `N8N_ENCRYPTION_KEY`, `N8N_LICENSE_ACTIVATION_KEY`, `N8N_HOST`, `N8N_PROTOCOL`, `N8N_WEBHOOK_URL`

### Zoho Books

- **Id:** `zoho-books`
- **URL:** https://www.zoho.com/books/
- **Role:** Ledger. Expenses, bills, invoices, contacts, customer payments, tax_id. Go never computes VAT/WHT.
- **Where:** n8n OAuth (Books write, Paystack paid, CRM upsert). Org ids and chart-of-accounts ids on the desk Defaults.
- **Secrets:** OAuth tokens in n8n credential Zoho Books. Never `.env`, never the Go worker.
- **n8n credentials:** `Zoho Books`
- **Desk keys:** `zoho.organization_id`, `zoho.paid_through_account_id`, `zoho.default_expense_account_id`, `zoho.account.office_supplies_id`, `zoho.account.advertising_id`, `zoho.account.lodging_id`, `zoho.account.uncategorized_id`, `zoho.deposit_to_account_id`

### Zoho Mail

- **Id:** `zoho-mail`
- **URL:** https://www.zoho.com/mail/
- **Role:** Company mailbox info@themobileprof.com. SMTP out (approved drafts), IMAP in (growth drafts).
- **Where:** n8n credentials. SMTP smtppro.zoho.com:465. IMAP imappro.zoho.com:993.
- **Secrets:** App password in n8n only.
- **n8n credentials:** `Zoho Mail info@`, `Zoho Mail info@ IMAP`

### Paystack

- **Id:** `paystack`
- **URL:** https://paystack.com/
- **Role:** Collect invoice payments (Payment Request). Webhook verifies then n8n books customerpayments in Zoho.
- **Where:** n8n only. Webhook https://workers.themobileprof.com/webhook/paystack-paid. Deposit account id on the desk.
- **Secrets:** `PAYSTACK_SECRET_KEY` on the VM `.env` → n8n container. Never git. Never the Go worker.
- **Env:** `PAYSTACK_SECRET_KEY`
- **Desk keys:** `paystack.currency`, `zoho.deposit_to_account_id`

### Telegram

- **Id:** `telegram`
- **URL:** https://core.telegram.org/
- **Role:** Ops room: prefixes including `/hr` and `/legal`, receipt photos, `/approve` and `/kill` for info@ drafts.
- **Where:** n8n Ops Telegram workflow.
- **Secrets:** Bot token in n8n credential Telegram account.
- **n8n credentials:** `Telegram account`

### WhatsApp (Meta Cloud API)

- **Id:** `whatsapp`
- **URL:** https://developers.facebook.com/docs/whatsapp/cloud-api
- **Role:** Customer and community chat. `/accounts`, `/ops`, `/legal`, and `/hr` are desk-allowlisted. Receipt OCR on allowlisted `/accounts` photos.
- **Where:** n8n Customer WhatsApp. Allowlist from desk People (`whatsapp_accounts`).
- **Secrets:** Tokens in n8n WhatsApp credentials.
- **n8n credentials:** `WhatsApp OAuth account`, `WhatsApp account`

### DeepSeek

- **Id:** `deepseek`
- **URL:** https://www.deepseek.com/
- **Role:** Default text LLM for department workers (classifier, chat). Text-only — no OCR.
- **Where:** Go agents via `LLM_PROVIDER=deepseek`.
- **Secrets:** `DEEPSEEK_API_KEY` (or `LLM_API_KEY`) on the VM `.env` → agents container.
- **Env:** `LLM_PROVIDER`, `LLM_MODEL`, `LLM_API_KEY`, `LLM_BASE_URL`, `DEEPSEEK_API_KEY`, `DEEPSEEK_MODEL`, `DEEPSEEK_BASE_URL`

### Google Gemini

- **Id:** `gemini`
- **URL:** https://ai.google.dev/
- **Role:** Vision OCR for receipt/invoice photos. Default model gemini-3.6-flash. Does not invent amounts when the image is unreadable.
- **Where:** Go agents when `context_data` includes `image_base64`. n8n downloads Telegram/WhatsApp photos.
- **Secrets:** `GEMINI_API_KEY` on the VM `.env` → agents container.
- **Env:** `GEMINI_API_KEY`, `GEMINI_MODEL`, `GEMINI_BASE_URL`

## Host

### PostgreSQL

- **Id:** `postgres`
- **URL:** https://www.postgresql.org/
- **Role:** Two databases on the host: n8n (orchestration) and aiworkers (company desk).
- **Where:** Native systemd on the VM. Never a Docker container.
- **Secrets:** `POSTGRES_*` for n8n. `ADMIN_DATABASE_URL` for the desk. Not in git.
- **Env:** `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `ADMIN_DATABASE_URL`

### Caddy

- **Id:** `caddy`
- **URL:** https://caddyserver.com/
- **Role:** TLS on 443. `/` and `/site*` plus `/admin*` → agents. `/webhook*` → n8n. `/n8n/` redirects to `/home`.
- **Where:** Native on the host. Config in `Caddyfile` (repo) and `/etc/caddy/Caddyfile` (live).
- **Secrets:** None in git. ACME via the public hostname.

## Code-ready

### OpenAI

- **Id:** `openai`
- **URL:** https://platform.openai.com/
- **Role:** Optional text LLM. Set `LLM_PROVIDER=openai` and `OPENAI_API_KEY` (or `LLM_API_KEY`).
- **Where:** Go Completer. Not live unless the env says so.
- **Secrets:** `OPENAI_API_KEY` / `LLM_API_KEY` on the VM `.env`.
- **Env:** `OPENAI_API_KEY`, `OPENAI_MODEL`, `OPENAI_BASE_URL`

### Anthropic

- **Id:** `anthropic`
- **URL:** https://www.anthropic.com/
- **Role:** Optional text LLM. Set `LLM_PROVIDER=anthropic` and `ANTHROPIC_API_KEY`.
- **Where:** Go Completer. Not live unless the env says so.
- **Secrets:** `ANTHROPIC_API_KEY` on the VM `.env`.
- **Env:** `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`, `ANTHROPIC_BASE_URL`
