# Third-party tools

Every system this office talks to. Add a section here the day you wire a new vendor — `go test ./internal/admin` fails if compose, `.env.example`, or an n8n credential shows up without a row.

Secrets stay in the VM `.env` or n8n Credentials, never git. Accounts / Books / Paystack **ids** live on desk **Defaults** (`/admin/settings`), not in this file and not in n8n JSON.

## Live

### n8n

- **Id:** `n8n`
- **URL:** https://n8n.io/
- **Role:** Orchestration: webhooks, cron, Zoho/Paystack client, execution logs. Public `/webhook*` and an allowlisted editor at `/home` (plus `/signin*`, `/rest*`, `/assets*`). Not a catch-all on the hostname.
- **Where:** Docker service n8n. Desk iframes `/home`.
- **Secrets:** `N8N_ENCRYPTION_KEY` and `N8N_LICENSE_ACTIVATION_KEY` on the VM `.env`. Per-app tokens live in n8n Credentials, not git.
- **Env:** `N8N_ENCRYPTION_KEY`, `N8N_LICENSE_ACTIVATION_KEY`, `N8N_HOST`, `N8N_PROTOCOL`, `N8N_WEBHOOK_URL`, `N8N_BLOCK_ENV_ACCESS_IN_NODE`

### Zoho Books

- **Id:** `zoho-books`
- **URL:** https://www.zoho.com/books/
- **Role:** Ledger. Expenses, bills, invoices, contacts, customer payments, tax_id. Go never computes VAT/WHT.
- **Where:** n8n OAuth (Books write, Paystack paid, CRM upsert for invoices when there is an email or the stage is qualified+, Ops Telegram `/crm` Books fallback). The CRM story lives on the desk (`GET /internal/v1/crm/contacts`). Org ids and chart-of-accounts ids on the desk Defaults.
- **Secrets:** OAuth tokens in n8n credential Zoho Books. Never `.env`, never the Go worker.
- **n8n credentials:** `Zoho Books`
- **Desk keys:** `zoho.organization_id`, `zoho.paid_through_account_id`, `zoho.default_expense_account_id`, `zoho.account.office_supplies_id`, `zoho.account.advertising_id`, `zoho.account.lodging_id`, `zoho.account.uncategorized_id`, `zoho.deposit_to_account_id`

### Zoho Mail

- **Id:** `zoho-mail`
- **URL:** https://www.zoho.com/mail/
- **Role:** Company mailbox info@themobileprof.com. SMTP out (a pending draft a human asked for). IMAP in (CRM thread summary on Office → CRM; important mail pings owner/BDM).
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
- **Role:** Two bots. Ops (`Telegram account`): office staff identified by People `telegram_id`; prefixes only within that seat; receipts; `/approve` `/kill` for owner/BDM. Academy (`Telegram Academy`): student groups, community mandate, Monday intro. No books, no legal, no HR on the Academy bot.
- **Where:** n8n Ops Telegram (Fetch staff `GET /internal/v1/staff?telegram_id=`), Academy Telegram, Delivery morning ping at 08:30 Lagos (tasks + Coming up + owner/BDM CRM due via `GET /internal/v1/delivery/digest`), inbound info@ important-mail pings.
- **Secrets:** Bot token in n8n credential Telegram account.
- **n8n credentials:** `Telegram account`, `Telegram Academy`

### WhatsApp (Meta Cloud API)

- **Id:** `whatsapp`
- **URL:** https://developers.facebook.com/docs/whatsapp/cloud-api
- **Role:** Customer 1:1 chat and CRM. Bare DMs are Growth; every successful DM with a phone files Office → CRM. Books upsert only with an email or once the stage is qualified+. Office prefixes (`/accounts`, `/ops`, `/legal`, `/seo`, `/today`, `/file`, `/approve`) are refused — staff use ops Telegram. Cloud API numbers cannot join ordinary WhatsApp groups — student rooms are Academy Telegram.
- **Where:** n8n Customer WhatsApp. No admin-phone allowlist.
- **Secrets:** Tokens in n8n WhatsApp credentials.
- **n8n credentials:** `WhatsApp OAuth account`, `WhatsApp account`

### DeepSeek

- **Id:** `deepseek`
- **URL:** https://www.deepseek.com/
- **Role:** Default text LLM for department workers (classifier, chat) and the desk Help? guide. Text-only — no OCR. If a call errors, Go tries Gemini / OpenAI / Anthropic when those keys are set.
- **Where:** Go agents via `LLM_PROVIDER=deepseek`.
- **Secrets:** `DEEPSEEK_API_KEY` (or `LLM_API_KEY`) on the VM `.env` → agents container.
- **Env:** `LLM_PROVIDER`, `LLM_MODEL`, `LLM_API_KEY`, `LLM_BASE_URL`, `DEEPSEEK_API_KEY`, `DEEPSEEK_MODEL`, `DEEPSEEK_BASE_URL`

### Google Gemini

- **Id:** `gemini`
- **URL:** https://ai.google.dev/
- **Role:** Vision OCR for receipt/invoice photos. Default model gemini-3.6-flash. Does not invent amounts when the image is unreadable. Also a text fallback when the primary chat provider errors (desk Help? bot included).
- **Where:** Go agents when `context_data` includes `image_base64`. n8n downloads Telegram/WhatsApp photos.
- **Secrets:** `GEMINI_API_KEY` on the VM `.env` → agents container.
- **Env:** `GEMINI_API_KEY`, `GEMINI_MODEL`, `GEMINI_BASE_URL`

### Fathom

- **Id:** `fathom`
- **URL:** https://fathom.video/
- **Role:** Meeting notetaker. Webhook when a recording is ready → Delivery proposes a week plan. Accept on the Delivery desk writes tasks. Chat never Accepts. Customer WhatsApp is not a chase channel.
- **Where:** n8n workflow Fathom notes. Production URL `https://workers.themobileprof.com/webhook/pm-notes`. Include summary, action items, and transcript on the Fathom webhook. Calendar titles should name the bet slug.
- **Secrets:** `FATHOM_API_KEY` and `FATHOM_WEBHOOK_SECRET` on the VM `.env` → n8n container. Never git. Never the Go worker.
- **Env:** `FATHOM_API_KEY`, `FATHOM_WEBHOOK_SECRET`

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
- **Role:** TLS on 443. `/` and `/site*` plus `/admin*` → agents. `/webhook*` `/form*` → n8n. Editor allowlist `/home*` `/signin*` `/rest*` `/assets*` `/static*` `/types*` → n8n. `/n8n/` redirects to `/home`. No catch-all to n8n. `/departments` and `/internal` stay unpublished.
- **Where:** Native on the host. Site block in `Caddyfile` (repo) → `/etc/caddy/conf.d/workers.caddy`. `/etc/caddy/Caddyfile` only imports `conf.d` so `enterprisellm.ng` can share the VM.
- **Secrets:** None in git. ACME via the public hostname.

## Code-ready

### OpenAI

- **Id:** `openai`
- **URL:** https://platform.openai.com/
- **Role:** Optional text LLM. Set `LLM_PROVIDER=openai` and `OPENAI_API_KEY` (or `LLM_API_KEY`). Also tried as a fallback when the primary provider errors.
- **Where:** Go Completer.
- **Secrets:** `OPENAI_API_KEY` / `LLM_API_KEY` on the VM `.env`.
- **Env:** `OPENAI_API_KEY`, `OPENAI_MODEL`, `OPENAI_BASE_URL`

### Anthropic

- **Id:** `anthropic`
- **URL:** https://www.anthropic.com/
- **Role:** Optional text LLM. Set `LLM_PROVIDER=anthropic` and `ANTHROPIC_API_KEY`. Also tried as a fallback when the primary provider errors.
- **Where:** Go Completer.
- **Secrets:** `ANTHROPIC_API_KEY` on the VM `.env`.
- **Env:** `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`, `ANTHROPIC_BASE_URL`

### Google Search Console

- **Id:** `google-search-console`
- **URL:** https://search.google.com/search-console
- **Role:** Your verified property only: queries, pages, clicks, impressions, position. Free API. Not competitor keywords. n8n pulls ~28 days as `ga4-read`; the SEO clerk only quotes that snapshot.
- **Where:** n8n workflow SEO GSC pull (same `googleApi` credential as GA4, plus `webmasters.readonly`). Property URL on desk Defaults (`seo.gsc_site_url`). Snapshot on `POST /internal/v1/seo/gsc`. The Go worker never holds the JSON key.
- **Secrets:** Same n8n credential Google Analytics (add Search Console scope). Never git, never `.env`, never the Go worker.
- **n8n credentials:** `Google Analytics`
- **Desk keys:** `seo.gsc_site_url`

### Google Analytics (GA4)

- **Id:** `google-analytics`
- **URL:** https://analytics.google.com/
- **Role:** Organic traffic for your GA4 property. Free API. Not rankings, not competitor analysis. n8n pulls last 28 days as the `ga4-read` service account; the SEO clerk only quotes that snapshot.
- **Where:** n8n workflow SEO GA4 pull (`googleApi` service account, not person OAuth). Property id on desk Defaults (`seo.ga4_property_id`). Snapshot on `POST /internal/v1/seo/ga4`. The Go worker never holds the JSON key.
- **Secrets:** Service account JSON in n8n credential Google Analytics. Never git, never `.env`, never the Go worker.
- **n8n credentials:** `Google Analytics`
- **Desk keys:** `seo.ga4_property_id`

### PageSpeed Insights

- **Id:** `pagespeed`
- **URL:** https://developers.google.com/speed/docs/insights/v5/get-started
- **Role:** Core Web Vitals / Lighthouse for a URL you name. Free quota with an API key from Google Cloud. Numbers come from Google — the clerk must not invent them.
- **Where:** Go agents when `PAGESPEED_API_KEY` is set. Key in the VM `.env`.
- **Secrets:** `PAGESPEED_API_KEY` on the VM `.env` → agents container. Never git.
- **Env:** `PAGESPEED_API_KEY`

### Google Drive

- **Id:** `google-drive`
- **URL:** https://drive.google.com/
- **Role:** Company data room (Shared Drive). Ops Telegram `/file` proposes a project folder; Accept on Files; n8n uploads. The Go worker never holds OAuth tokens and never writes Drive itself.
- **Where:** n8n credential Google Drive. Shared Drive id on desk Defaults (`drive.shared_drive_id`).
- **Secrets:** OAuth tokens in n8n only. Never `.env`, never the Go worker.
- **n8n credentials:** `Google Drive`
- **Desk keys:** `drive.shared_drive_id`
