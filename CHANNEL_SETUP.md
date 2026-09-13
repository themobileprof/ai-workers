# Channel setup (Telegram, WhatsApp, Email)

n8n is the only public door. Agents still talk over HTTP JSON. Chat apps are inbound/outbound edges.

| Channel | Role | Identity |
| --- | --- | --- |
| Telegram | You (ops room): watch jobs, approve, poke a department | One bot, one private chat or group |
| WhatsApp | Customers, community, field intern | One Business number |
| Email | Formal humans: grants, investors, NDAs, invoices | One sending domain |

Do not give each department its own WhatsApp or bot. n8n routes to the Go workers on the **Docker network** at `http://agents:8000/departments/{internal-ops,growth,product-dev,community}`. That hostname only works inside an n8n **HTTP Request** node (or `docker compose exec n8n ...`). It is not a browser URL.

Webhook origin is already `https://workers.themobileprof.com/` (`N8N_WEBHOOK_URL`). Production URLs look like `https://workers.themobileprof.com/webhook/<id>`. Test URLs contain `webhook-test` and only work while Listen is on. Meta and Telegram must get the **production** URL, and the workflow must be **published/active**.

Store tokens in n8n **Credentials**, not in git. `.env` on the VM is for Postgres, encryption, LLM, and license only.

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
3. n8n → **Credentials** → Telegram API → paste token.
4. New workflow **Ops Telegram**:
   - **Telegram Trigger** (updates: message)
   - **Switch** or IF: if text starts with `/growth`, `/ops`, `/validate` pick the department URL; else default `growth` (sales/support).
   - **Set** node (optional): `task_description` = `{{ $json.message.text }}`.
   - HTTP Request as in section 0, using the Telegram JSON body (not the empty-input template).
   - **Telegram** send message to `{{ $json.message.chat.id }}` with `{{ $json.output_text }}` from the HTTP node (`{{ $('HTTP Request').item.json.output_text }}`).
5. **Publish** the workflow. Do not leave it on Listen-only.
6. Confirm Telegram registered the production hook:

```text
https://api.telegram.org/bot<TOKEN>/getWebhookInfo
```

`url` must contain `/webhook/` not `/webhook-test/`. `last_error_message` must be empty.

Commands worth adding later: `/approve`, `/kill` for validation decisions — still n8n IF nodes, still one bot.

---

## 2. Email outbound, then inbound

### Outbound (founder → customer)

Pick one provider. Resend or Amazon SES is simpler than raw Gmail SMTP (app passwords, blocking).

1. Verify `themobileprof.com` (SPF, DKIM, a sending domain).
2. n8n **Credentials** → SMTP or Resend API.
3. Workflow **Send email**: From something like `workers@themobileprof.com` (or `hello@`). Body from the worker `output_text`. Attachments only when Accounts/Legal produced a file later.
4. Trigger it from other workflows (grant copy, NDA draft, invoice note) — do not let every department SMTP itself.

### Inbound (customer emails you)

Do this after outbound works.

- Resend inbound, or Google Workspace “forward to webhook”, or Mailgun route → n8n **Webhook** production URL.
- Strip HTML to text, HTTP POST `growth` (or Switch on To: address: `grants@` → internal-ops grant hunting).
- Reply via the outbound credential so the thread stays on your domain.

Unattended inbox → LLM → send is how you get embarrassing mail. First version: Telegram notify **you** with the draft; you `/approve` then n8n sends.

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
3. Email outbound to yourself.
4. WhatsApp test number + one customer-style ping.
5. Email inbound + Telegram approve-before-send.
6. Only then: intern mission on WhatsApp.

If a channel fails, check Caddy (`https://workers.themobileprof.com`), workflow published, and production vs test webhook URL before touching the Go worker.
