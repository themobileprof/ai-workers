# TheMobileProf workers

One Lagos company, [TheMobileProf Technologies](https://workers.themobileprof.com). n8n is the public door. A Go process holds the workers and the **desk**. Humans stamp. Workers propose.

Live desk: `https://workers.themobileprof.com/admin/`  
Field playbook (signed in): `/admin/docs`

## How to use it

### Company desk (owner / BDM)

Sign in on `/admin/`. You get Board, Projects, People, Defaults, Docs, n8n.

- **People** — add a human, pick an office seat, tick the **handler desks** they actually run. Those workers appear when **that person** signs in, not on the Board. Ticking a desk grants `web_admin`.
- **Projects** — incubated bets (MomLaunchpad, Mechazone, HomeGauge, …). Journey/gate is a stamp. Ask placement writes a proposal; Accept moves it.
- **Defaults** — Zoho org and chart-of-accounts ids. Paystack deposit account may stay blank until you create a **Bank** row in Books. Secrets stay in the VM `.env` / n8n Credentials.

If you also handle Legal or HR, tick those desks on your own People card or the books 403.

### Handler desk

Anyone assigned desks lands on `/admin/desks/{slug}` (or a picker if several). Ask the worker, chat the job, Accept or Turn away. Chat does not stamp. Community handlers also get **Mandates** (`/admin/community`): one brief + catalog per WhatsApp group. Academy LMS is seeded from the public course list. Paste `group_id` from an n8n execution after the Business number is in the room. Monday 09:00 Lagos the worker introduces itself in each matched group.

### Chat prefixes

Do not give each department its own WhatsApp. One ops Telegram, one customer WhatsApp, one `info@`.

| Starts with | Worker |
| --- | --- |
| `/accounts` | accounts (WhatsApp allowlisted) |
| `/ops` | internal-ops (WhatsApp allowlisted) |
| `/legal` | legal (WhatsApp allowlisted) |
| `/hr` | hr (WhatsApp allowlisted) |
| `/growth` | growth |
| `/crm` | crm |
| `/cm` `/community` | community |
| `/validate` | product-dev |
| none, 1:1 | growth |
| none, group | community |

`/approve` / `/kill` on Telegram only send or drop the pending **info@** draft. They do not Accept a legal/HR/project stamp.

Customer-development field work (rates at a mortgage desk, mechanics, a PHC) is a human on a project, not a chat prefix. Bring the names back; `/validate` `analyse_evidence` can score them. The desk still stamps GO/PIVOT/KILL.

Full channel inventory: [`CHANNEL_SETUP.md`](CHANNEL_SETUP.md). Vendors: [`TOOLS.md`](TOOLS.md).

## How the code is laid out

```
agents/cmd/server/          HTTP process (site, /admin, /departments, /internal)
agents/internal/contract/   n8n ↔ Go JSON envelope
agents/internal/departments/{growth,community,crm,legal,hr,productdev}/
agents/internal/departments/internalops/  /ops and /accounts (HandleAccounts)
agents/internal/admin/      desk: templates/, playbook/, Postgres `aiworkers`
                            docs.go catalog · desks.go handler list · store.go migrate
agents/internal/journeys/   idea→PMF playbooks (gates)
agents/internal/llm/        Completer; DeepSeek chat, Gemini vision
n8n/workflows/              imported JSON; credentials stay in n8n
scripts/sync-n8n-workflows.sh
```

n8n calls `http://agents:8000/departments/{name}` on the Docker network with header `X-Internal-Token` (same as `/internal/v1`). Caddy must not publish `/departments` or `/internal`. Body is always:

```json
{ "task_description": "…", "context_data": {} }
```

`context_data` is an object, never a blank expression. Timeout 120s.

Worker reply: `status`, `output_text` (what the human reads), `structured_data` (what n8n may act on). Accounts never compute VAT/WHT — Zoho does. Legal/HR/placement POSTs insert **proposed** rows only.

Internal JSON for n8n (`X-Internal-Token`). Same header on `POST /departments/{name}`. Caddy must not publish `/internal`. None of the POSTs stamp Accept.

| Method | Path |
| --- | --- |
| GET | `/internal/v1/community/mandate` · `/internal/v1/community/mandates` |
| GET | `/internal/v1/whatsapp-accounts` |
| GET | `/internal/v1/settings` |
| GET | `/internal/v1/projects` · `/internal/v1/journeys` |
| POST | `/internal/v1/projects/{id}/proposal` |
| GET, POST | `/internal/v1/legal-drafts` |
| GET | `/internal/v1/hr/roles` |
| POST | `/internal/v1/hr/applications` |
| POST | `/internal/v1/jobs` |

See Desk → Docs → How to call a worker.

## Tests

```
cd agents && go test ./...
cd agents && go test ./... -cover
```

`go test` does not start Postgres or call DeepSeek. Desk HTML, prefixes, the n8n envelope, and worker Handles run against a stub Completer. `./internal/admin` also fails if compose / `.env.example` vendor env or an n8n credential name is missing from `TOOLS.md`. Store migrate and live LLM HTTP stay unhit until you point `ADMIN_DATABASE_URL` at a throwaway `aiworkers` and set keys.

## Adding something

| You add | You also |
| --- | --- |
| A department worker | Extra Go route on this process, `docs.go` row, `playbook/{slug}.html`, n8n HTTP node. Leave a later-box until the workflow is published. |
| A vendor | Section in `TOOLS.md` (env, n8n credential names, desk keys). `go test ./internal/admin` fails if compose / `.env.example` / a credential name is missing. Accounts ids on Defaults, not in workflow JSON. |
| A webhook workflow | JSON under `n8n/workflows/`, then `./scripts/sync-n8n-workflows.sh`. Import deactivates; the script republishes webhooks **and** Execute Workflow children (Books write, CRM upsert, Email outbox). |

## CI / CD

- **CI** on every pull request to `main` and on merge to `main`: `go test ./...` plus a `linux/arm64` binary (`agents/dist/agents`). No Postgres, no DeepSeek.
- **CD** when you **publish a GitHub Release** (or Actions → CD → Run workflow): cross-compile that binary, rsync this tree to `/home/cursor/ai-workers`, `docker compose` with `Dockerfile.runtime` (no golang image on the VM), copy `Caddyfile` to `/etc/caddy/conf.d/workers.caddy` when sudo allows, republish n8n webhooks, `GET /health`. The VM `.env` is never overwritten.

Cut a release after main is green:

```
git tag v0.1.0
git push origin v0.1.0
gh release create v0.1.0 --generate-notes
```

Repo secret **`DEPLOY_SSH_KEY`** is required (ed25519 private key whose public half is in `cursor`’s `authorized_keys` on the VM). Optional: `DEPLOY_HOST` (default `130.61.144.174`), `DEPLOY_USER` (default `cursor`). Port 22 must accept GitHub-hosted runners; key-only, no password.

Host VM, Compose caps, Caddy: [`SERVER_INIT_INSTRUCTIONS.md`](SERVER_INIT_INSTRUCTIONS.md). Cursor architecture prompt: [`CURSOR_INSTRUCTIONS.md`](CURSOR_INSTRUCTIONS.md).
