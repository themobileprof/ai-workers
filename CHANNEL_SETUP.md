# Channel setup (Telegram, WhatsApp, Email)

n8n is the only public door. Agents still talk over HTTP JSON. Chat apps are inbound/outbound edges.

| Channel | Role | Identity |
| --- | --- | --- |
| Telegram | You (ops room): watch jobs, approve, poke a department | One bot, one private chat or group |
| WhatsApp | Customers, community, field intern | One Business number |
| Email | Formal humans: grants, investors, NDAs, invoices | One sending domain |

Do not give each department its own WhatsApp or bot. n8n routes to the Go workers on the **Docker network** at `http://agents:8000/departments/{internal-ops,growth,product-dev,community,crm}`. That hostname only works inside an n8n **HTTP Request** node (or `docker compose exec n8n ...`). It is not a browser URL.

Webhook origin is already `https://workers.themobileprof.com/` (`N8N_WEBHOOK_URL`). Production URLs look like `https://workers.themobileprof.com/webhook/<id>`. Test URLs contain `webhook-test` and only work while Listen is on. Meta and Telegram must get the **production** URL, and the workflow must be **published/active**.

Store tokens in n8n **Credentials**, not in git. `.env` on the VM is for Postgres, encryption, LLM, and license only.

---

## Live inventory

Keep this section in sync with every channel change. IDs are not secrets.

### n8n credentials

| Name | Type | ID | Used by |
| --- | --- | --- | --- |
| Telegram account | `telegramApi` | `GD7WqGif3v6QlSbc` | Ops Telegram |
| WhatsApp OAuth account | `whatsAppTriggerApi` | `Oaps5dWBa76PVrD9` | Customer WhatsApp trigger |
| WhatsApp account | `whatsAppApi` | `TpVfkoyKY6eZDl87` | Customer WhatsApp send |
| Zoho Books | `oAuth2Api` | `ByEtw1MmiDqYxQWI` | Smoke: Zoho Books; `/ops` expenses; CRM contact upsert |
| Zoho Mail info@ | `smtp` | `cvlEAE8BvvQ87wew` | Smoke: send email; Email outbox approved sends |
| Zoho Mail info@ IMAP | `imap` | `w6VyL4iDdwX5XgIM` | Inbound info@ |

SMTP host in the credential (not in git as a secret): `smtppro.zoho.com:465` SSL, user `info@themobileprof.com`. App password stays in n8n only.

### Workflows in repo

| File | n8n id | Active | What it does |
| --- | --- | --- | --- |
| `n8n/workflows/smoke-growth.json` | `smkGrwthHttp0001` | no | Manual POST growth worker |
| `n8n/workflows/smoke-community.json` | `smkCommHttp0001` | no | Manual POST community worker |
| `n8n/workflows/smoke-zoho-books.json` | `smkZohoBooks0001` | no | GET Zoho orgs + chart of accounts |
| `n8n/workflows/smoke-email.json` | `smkEmailSmtp0001` | no | Send one text mail From/To `info@themobileprof.com` |
| `n8n/workflows/smoke-crm.json` | `smkCrmBooks00001` | no | GET Zoho Books contacts (read-only) |
| `n8n/workflows/crm-upsert.json` | `crmUpsertCont0001` | no (sub-workflow) | Upsert a Books customer from `record_lead` |
| `n8n/workflows/ops-telegram.json` | `opsTelegram00001` | yes | Ops room; prefixes; `/ops` expenses; `/crm` leads; `/approve` `/kill` email drafts |
| `n8n/workflows/customer-whatsapp.json` | `custWhatsApp0001` | yes | Customer WhatsApp; same prefixes; `/ops` expenses; CRM upsert on high intent |
| `n8n/workflows/email-outbox.json` | `emailOutbox000001` | no (sub-workflow) | Stores one pending draft; SMTP send on `/approve` |
| `n8n/workflows/inbound-email.json` | `inbdEmailImap0001` | yes | IMAP INBOX → growth draft → Telegram; upserts Books contact from sender |

UI leftover (not in git): `thnYcpkfxCvFveX8` “My workflow”.

### Chat prefixes (Telegram + WhatsApp)

| Message starts with | Department |
| --- | --- |
| `/community` or `/cm` | community |
| `/growth` | growth |
| `/crm` | crm |
| `/ops` | internal-ops |
| `/validate` | product-dev |
| none, 1:1 | growth |
| none, group | community |

### Zoho Books (not Mail)

- API: `https://www.zohoapis.com` (US DC). Org **TheMobileProf Technologies**, `organization_id` `939049468`, currency NGN.
- Write path: `/ops` → worker `structured_data.record_expense` + numeric `amount` → `POST /books/v3/expenses`.
- CRM path: `/crm` or high-intent growth/email → upsert **Books contact** (`POST`/`PUT /books/v3/contacts`). This is the CRM. Do **not** add a Zoho CRM OAuth app unless you buy Zoho CRM; Books contacts already sit on the existing credential.
- Defaults: expense account Other Expenses `1300646000000000460`, paid through Petty Cash `1300646000000000361`. Also mapped: Office Supplies `…400`, Advertising `…403`, Lodging `…32023`, Uncategorized `…35005`.
- Do not put Zoho tokens in `.env` or the Go worker.

### Email (Zoho Mail)

- Company address: **`info@themobileprof.com`**. From display name: `TheMobileProf`.
- Outbound: n8n **Send Email** + credential `Zoho Mail info@`. Paid Zoho Mail SMTP is `smtppro.zoho.com` / `465` SSL (fallback `smtp.zoho.com` if a send fails).
- Inbound: **Inbound info@** IMAP (`imappro.zoho.com:993`, credential `Zoho Mail info@ IMAP`) → growth worker drafts a reply → **Email outbox** holds **one** pending draft → Telegram notify. Nothing is sent until `/approve` in Ops Telegram. `/kill` drops it. A new inbound mail overwrites the pending draft. The sender is upserted as a Zoho Books contact.
- Skips mail From `info@themobileprof.com` (loop) and subjects matching `n8n smoke`.
- Message the ops bot once (`/help`) after deploy so outbox learns `ops_chat_id`; otherwise the draft is still stored and `/approve` still works, but you will not get the Telegram card.
- Replies are a new message with `Re:` subject (n8n SMTP does not set `In-Reply-To`).
- Do not add a second public From (`workers@`, `hello@`); one address is the company.

---

## 0. Shared pieces (do once)

### Prove the worker with static JSON

The repo already ships `n8n/workflows/smoke-growth.json`. Import it with `scripts/sync-n8n-workflows.sh`, then in the UI open **Smoke: growth HTTP** and Execute. Do not use `{{ }}` expressions on a manual trigger — Execute workflow has no input, and empty expressions produce invalid JSON (`"context_data": }`).

If you rebuild the node by hand instead:

- Method: `POST`
- URL: `http://agents:8000/departments/growth`
- Authentication: None
- Send Body: on
- Body Content Type: JSON
- Specify Body: Using JSON
- JSON (paste exactly):

```json
{
  "task_description": "Write a short LinkedIn post about a WhatsApp bookkeeping assistant for Lagos mechanics.",
  "context_data": {}
}
```

- Options → Timeout: at least `120000` ms (2 minutes)

Execute step. You should see `status`, `output_text`, and `structured_data`. Swap the path for `/departments/internal-ops` or `/departments/product-dev` as needed.

### After a Trigger exists (Telegram / WhatsApp / Webhook)

Then map fields. `context_data` must stay a JSON **object**, never a blank expression.

Use **Specify Body: Using JSON**:

```json
{
  "task_description": "{{ $json.message.text }}",
  "context_data": {
    "channel": "telegram",
    "chat_id": "{{ $json.message.chat.id }}"
  }
}
```

Telegram’s payload shape is `message.text` and `message.chat.id`. WhatsApp nodes differ — inspect the trigger output Schema tab and adjust. If you need to pass an object from a previous node, stringify it so the body stays valid JSON:

```json
{
  "task_description": "{{ $json.task_description }}",
  "context_data": {{ JSON.stringify($json.context_data || {}) }}
}
```

Never write `"context_data": {{ $json.context_data }}` unless you have already confirmed that field is a non-empty object.

Create a second small workflow later: **Notify ops** — Telegram send of `output_text` (truncated) + `status`. Every customer-facing flow should call it at the end so you see what went out.

---

## 1. Telegram first (about 20 minutes)

Fastest loop. Proves inbound → worker → outbound on HTTPS.

1. In Telegram, talk to [@BotFather](https://t.me/BotFather) → `/newbot` → copy the token.
2. Message the bot once (so a `chat_id` exists), or create a private group, add the bot, send a message.
3. n8n → **Credentials** → Telegram API → paste token. The repo workflow **Ops Telegram** (`n8n/workflows/ops-telegram.json`) is imported and published from there — do not rebuild the nodes by hand.
4. Message the bot `/help`, then `/growth` plus a task. Same prefixes as WhatsApp: `/cm`, `/ops`, `/validate`. Groups default to community; 1:1 defaults to growth.
5. Confirm Telegram registered the production hook:

```text
https://api.telegram.org/bot<TOKEN>/getWebhookInfo
```

`url` must contain `/webhook/` not `/webhook-test/`. `last_error_message` must be empty.

`/approve` and `/kill` send or drop the pending **info@** draft. Validation GO/PIVOT/KILL can reuse the same commands later.

---

## 2. Email (`info@themobileprof.com`)

Stay on **Zoho Mail** (paid year). n8n uses SMTP, not the Zoho Mail API. Books OAuth is a different credential.

### Outbound (live)

Credential **Zoho Mail info@** (`cvlEAE8BvvQ87wew`):

1. Mailbox `info@themobileprof.com` exists in Zoho Mail Admin.
2. App password from [Zoho Accounts](https://accounts.zoho.com/) → Security → App passwords (no spaces). Not the web login password.
3. SMTP: `smtppro.zoho.com`, port `465`, SSL/TLS on, user `info@themobileprof.com`. Leave Client Host Name empty.
4. IMAP must be enabled on the mailbox before inbound (Settings → Mail Accounts → IMAP). Server later: `imappro.zoho.com:993`.

**Smoke: send email** (`n8n/workflows/smoke-email.json`): open it in the UI and Execute. It sends a plain-text message From/To `info@themobileprof.com`. Check that inbox (and spam). It is inactive on purpose — do not publish.

Approved customer replies use the same SMTP node inside **Email outbox**. Departments must not each open SMTP.

### Inbound (live)

1. Enable IMAP on the mailbox (Zoho Mail → Settings → Mail Accounts → IMAP).
2. Credential **Zoho Mail info@ IMAP** (`w6VyL4iDdwX5XgIM`): `imappro.zoho.com`, `993`, SSL, user `info@themobileprof.com`, same app password as SMTP.
3. **Inbound info@** is published. Send a mail **to** `info@` from a non-`info@` address (your Gmail). Telegram should get a draft. `/approve` sends; `/kill` drops.
4. First, `/help` the ops bot so it records your chat id.

---

## 3. WhatsApp last (Meta, half a day)

HTTPS is already in place. WhatsApp Cloud API still needs a Meta app.

1. [Meta for Developers](https://developers.facebook.com/) → app type **Business** → add **WhatsApp**.
2. WhatsApp → API Setup: note **Phone number ID**, **WhatsApp Business Account ID**, temporary token (then a system user permanent token).
3. n8n **Credentials** → WhatsApp Business Cloud (access token + IDs).
4. Workflow **Customer WhatsApp**:
   - **WhatsApp Trigger** (or Webhook if you wire Meta by hand).
   - Subscribe to `messages`.
   - Ignore echo/status payloads (`statuses` vs `messages`).
   - HTTP Request → `http://agents:8000/departments/growth` for chat; Switch to `internal-ops` if the text looks like a receipt/Paystack/VAT.
   - **WhatsApp** send text: `output_text` (keep it short; WhatsApp is not Markdown-friendly — a Code node that strips headings helps).
5. In Meta webhook config, paste n8n’s **production** callback URL and verify token. Publish the n8n workflow **before** Meta’s verify GET, or verification fails.
6. For messages you send first (outside the 24h window) you need a **template** approved in WhatsApp Manager. Session replies inside 24h can be free-form.

Use the test number first. Move to a real Nigerian number when the test loop works. One number = the company.

On that number, n8n routes by prefix (then default **growth** for 1:1, **community** for group chats):

| Message starts with | Department |
| --- | --- |
| `/community` or `/cm` | community |
| `/growth` | growth |
| `/crm` | crm |
| `/ops` | internal-ops |
| `/validate` | product-dev |

Telegram community uses the same worker path once a bot credential exists. Do not create a second WhatsApp number for community.

Intern field missions: same number, or a second **internal** WhatsApp later. Do not mix intern debriefs and customer support in one thread without a prefix (`MISSION:` vs customer).

---

## 4. How the three fit together

```
Customer WhatsApp/Email ──► n8n ──► Go department ──► n8n ──► same channel
                                      │
                                      └──► Telegram (you see a copy)
You Telegram ──► n8n ──► Go department ──► Telegram
Agents among themselves: n8n HTTP only (never WhatsApp/Telegram as the bus)
```

Validation loop: Telegram `/validate` → `POST .../product-dev` with `context_data.action`. n8n stores the returned `structured_data` (hypotheses, mission) in workflow static data or a Data Table, then WhatsApp/email the intern the mission text.

When a new flow is described in Cursor, add or edit a JSON file under `n8n/workflows/`, run `scripts/sync-n8n-workflows.sh`, and (for webhooks) publish. Tokens still go in n8n **Credentials**, not in those JSON files.

---

## 5. Order to actually click

1. Import **Smoke: growth HTTP** (`scripts/sync-n8n-workflows.sh`) and Execute it. Confirm `output_text`.
2. Telegram bot + Ops workflow + getWebhookInfo clean.
3. **Smoke: send email** — Execute; confirm mail in `info@`.
4. WhatsApp is already live; keep using prefixes.
5. Email inbound is live: mail `info@` → Telegram draft → `/approve`.
6. Only then: intern mission on WhatsApp.

If a channel fails, check Caddy (`https://workers.themobileprof.com`), workflow published, and production vs test webhook URL before touching the Go worker.
