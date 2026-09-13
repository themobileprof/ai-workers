# MASTER PROMPT: SYNTHETIC STARTUP MULTI-AGENT HUB

## 1. STRATEGIC GOAL & CONTEXT

I am a technical startup founder and a lone developer operating out of Lagos, Nigeria. I use Cursor as my core coding workstation. My goal is to build a highly synchronized, automated multi-agent "synthetic company."

This system must run multiple background AI departments that talk to each other and external channels, allowing me to run a hyper-lean startup with absolute minimal human overhead.

## 2. SYSTEM ARCHITECTURE & STACK CONFIGURATION

You have full creative and architectural freedom over how the inner software code patterns are structured, but you must respect the following core infrastructural pillars:

* **Host Environment:** Single Oracle Cloud Infrastructure (OCI) Free-Tier Ubuntu VM (`aarch64`).
* **Hardware Constraints:** Exactly 1 OCPU and 6 GB of RAM. Disk is a custom 50–100 GB OCI boot volume.
* **Orchestration Engine:** **n8n**, self-hosted in Docker. Visual router, webhook manager, event processor, cron scheduler, and execution log keeper. Stay on Node/n8n here — replacing it with a custom Go orchestrator loses the visual workflows this system is built around.
* **System Database:** **PostgreSQL, native on the host** (systemd / apt — **not** a Docker container). n8n stores its backend and execution state here. Local beats a remote free tier (Supabase/Neon) for latency, connection-pool stability, and no cold starts.
* **Execution Worker Fleet:** A **Go** HTTP service in Docker. n8n is the only caller. Do **not** publish the worker to the public internet.
* **TLS edge (when a domain exists):** **Caddy** native on the host (Go binary via apt/official repo — **not** a third container). Terminates HTTPS on 443 and reverse-proxies to n8n on localhost. WhatsApp/Telegram Cloud APIs need this.

```
External Inputs: WhatsApp / Telegram / Cron / HTTPS webhooks
        │
        ▼
┌──────────────────────────────────────────┐
│  Caddy (host, Go) :443  — when domain    │
│  bootstrap only: n8n published on :5678  │
└──────────────────┬───────────────────────┘
                   ▼
┌──────────────────────────────────────────┐
│  n8n  (Docker)                           │
└──────────────────┬───────────────────────┘
                   │ agent-network (internal)
                   ▼
┌──────────────────────────────────────────┐
│  agents  (Go static binary, Docker)      │
│  internalops | growth | productdev       │
└──────────────────────────────────────────┘

n8n ──TCP 5432──► PostgreSQL (host systemd, not Docker)
                  5432 must never be in the OCI security list
```

### Why Go for workers (and not Python / FastAPI)

This VM is RAM-bound. n8n (Node) already owns the expensive slice. The worker must stay tiny.

| Choice | Typical RSS on this box | Verdict |
| --- | --- | --- |
| **Go stdlib/chi, static binary** | ~15–40 MB idle; cap **256 MB** | Default. One process, no interpreter, excellent concurrency for outbound LLM HTTP on 1 OCPU. |
| FastAPI + uvicorn + LiteLLM + Pydantic | ~250–500 MB before first request | Rejected. LiteLLM pulls tiktoken and a 100-provider registry we will never use. Extra workers fight the single OCPU. |
| Python sidecar "just for LiteLLM" | Two runtimes, two failure domains | Rejected. Hybrid is worse than either language alone. |
| Go rewrite of n8n | Months of work, lose visual ops | Rejected. n8n stays. |
| Local LLM (Ollama, etc.) | Multi-GB | Forbidden on this VM. Inference is remote HTTP. |

Python is allowed later **only** as an optional sidecar if a job truly needs a Python-only library (heavy OCR/PDF). That is not v1. Do not pre-create an empty Python service.

LLM provider switching does **not** require LiteLLM. A small Go `Completer` interface with OpenAI / Anthropic / DeepSeek clients, selected by `LLM_PROVIDER` + `LLM_API_KEY` in `.env`, is the whole abstraction we need.

Build the worker as `CGO_ENABLED=0 GOOS=linux GOARCH=arm64` (this host is Oracle aarch64). Multi-stage Docker: compile in `golang`, copy a static binary into `gcr.io/distroless/static` or `scratch`. Do not install a Go toolchain on the host.

## 3. CORE CONSIDERATIONS & CONSTRAINTS TO ENFORCE

### A. Memory budget on a 6 GB, 1 OCPU box

This is a **shared** machine, not a dedicated database server. Do not apply "25% of RAM for `shared_buffers`".

| Slice | Limit | Why |
| --- | --- | --- |
| OS + SSH + systemd (+ Caddy later) | ~0.6–0.9 GB | Untouchable |
| PostgreSQL (native) | ~0.4–0.6 GB | `shared_buffers=256MB`, `max_connections=20`, `work_mem=4MB` |
| n8n container | **1536 MB hard cap** | Node is the hungry process |
| Go agents container | **256 MB hard cap** | Plenty for a static binary + a few concurrent LLM calls |
| Headroom / page cache | ~2 GB | OOM safety + outbound LLM HTTP |

**PostgreSQL (host `postgresql.conf`):**

- `shared_buffers = 256MB`
- `work_mem = 4MB`
- `max_connections = 20`
- `maintenance_work_mem = 64MB`
- `effective_cache_size = 512MB`

**n8n pruning (disk, not RAM):**

- `EXECUTIONS_DATA_PRUNE=true`
- `EXECUTIONS_DATA_MAX_AGE=72`
- `EXECUTIONS_DATA_PRUNE_MAX_COUNT=1000`
- Prefer `EXECUTIONS_DATA_SAVE_ON_SUCCESS=none` so successful runs do not pile up.

**Docker:**

- Compose file contains **only n8n and agents**. Postgres is not a compose service. Caddy is not a compose service.
- Isolated bridge `agent-network`. n8n reaches the worker at `http://agents:8000`.
- n8n reaches host Postgres via `host.docker.internal` (`extra_hosts: host.docker.internal:host-gateway`).
- Pin image tags. Never `latest`.
- Do **not** enable n8n queue mode (main + worker + Redis).
- Go worker: **one** process. Do not run a process supervisor with N replicas on 1 OCPU.
- `GOMAXPROCS=1` is acceptable and honest on this CPU.

**Do not install on the host:** Python pip, a Go toolchain, or Node (n8n brings its own Node inside Docker).

### B. Network exposure

- Bootstrap: publish **only** n8n (`5678`). Worker stays on `agent-network` with no `ports:` mapping.
- After Caddy: bind n8n to `127.0.0.1:5678` only; Caddy publishes `443` (and `80` for ACME). Worker still unpublished.
- Postgres `5432` is host-local + Docker subnet in `pg_hba.conf`. Not in UFW. Not in the OCI Security List / NSG.
- n8n must have authentication enabled from first boot plus a stable `N8N_ENCRYPTION_KEY`.
- Local UFW is not enough on OCI: the **VCN security list / NSG** must allow 22 and 5678 (then 80/443; drop 5678 from the cloud firewall once Caddy owns the edge). Never allow 8000 or 5432.

### C. Localized Nigerian/African context

- **Accounts Agent:** VAT and WHT, Paystack / Flutterwave / Moniepoint payment notifications.
- **Legal Agent:** Nigerian labour basics, CAC structures, CBN / NITDA-oriented compliance flags.
- **Growth Agent:** African / emerging-market grant engines (Google for Startups Accelerator Africa, Tony Elumelu Foundation, USAID) and copy that still reads for international investors.

## 4. THE DEPARTMENT FILES & EXECUTION CONTRACT

Group departments into **three** Go packages. Routing lives in `cmd/server`: one HTTP path per department. Each package classifies the sub-task and calls the LLM `Completer`. Do not add a fourth "router agent" process.

Every department handler takes `task_description` and `context_data` (JSON object) and returns:

```json
{
  "status": "success" or "failed",
  "output_text": "Comprehensive Markdown formatted text detailing the agent's work/findings.",
  "structured_data": {}
}
```

The worker does **not** share n8n's Postgres database. n8n owns orchestration state. Workers are stateless. Persistence of agent outputs, if needed later, is an n8n workflow concern or a separate schema — not v1 coupling.

### Layout

```
agents/
  Dockerfile
  go.mod
  cmd/server/main.go
  internal/contract/     # request/response structs
  internal/llm/          # Completer + OpenAI / Anthropic / DeepSeek
  internal/departments/internalops/
  internal/departments/growth/
  internal/departments/productdev/
```

### Module mapping

1. **`internalops` (Operations Room)**
    - *Accounts:* Parse receipt text/images, categorize expenses, extract vendors, compute local tax splits.
    - *Legal:* Contract risk, localized NDAs/SLAs, predatory-clause flags.
    - *Grant Hunting:* Startup metadata vs grant eligibility parameters.
2. **`growth` (Front Office)**
    - *Marketing:* Platform-customized social copy (Twitter/X, LinkedIn) with local market context.
    - *Sales & Support:* Incoming queries (WhatsApp Business / live chat); calendar booking metadata on high intent.
3. **`productdev` (Engineering Lab)**
    - *Product Ops / QA:* Code diffs or logs from GitHub webhooks; bugs and leaked secrets.
    - *Customer Success:* Cohort telemetry → churn risk and retention copy.
    - *Product validation:* Port of the idea-validation loop (hypotheses → intern mission → evidence → GO/PIVOT/KILL). n8n holds project state; the worker only proposes. Set `context_data.action` to one of `generate_hypotheses`, `generate_plan`, `generate_mission`, `generate_interview_guide`, `analyse_evidence`, `update_hypothesis`, `recommend_next_experiment`, `generate_decision_report`.

## 5. RECOMMENDED TECH STACK SPECIFICATIONS

- **Go 1.23+**, stdlib `net/http` (chi only if routing would otherwise get messy).
- **LLM:** small `Completer` interface; remote HTTP only; swap providers via `.env`.
- **Caddy** on the host when a domain is ready (Go, native, automatic TLS).
- n8n + Docker Compose v2 + native PostgreSQL.

Host packages required: Docker Engine + Compose v2 plugin, Git, native PostgreSQL. Caddy only after DNS exists.

### n8n workflows as code

Do **not** ask the founder to click nodes. Workflows live in `n8n/workflows/*.json` (stable `id` per file) and are imported with `scripts/sync-n8n-workflows.sh`. Credentials (Telegram, WhatsApp, SMTP) stay in the n8n UI / Postgres — never in git.

- Compose mounts `./n8n/workflows` read-only at `/home/node/workflows`.
- Owner assignment: `n8n/instance.json` (`userId` / `projectId`).
- Re-importing the same `id` updates the workflow. Import deactivates unless you publish: `N8N_PUBLISH=id1,id2 ./scripts/sync-n8n-workflows.sh`.
- Manual-trigger smoke tests do not need publishing. Webhook/Telegram/WhatsApp/cron flows **must** be published so production URLs work.
- n8n HTTP Request nodes call `http://agents:8000/departments/{internal-ops,growth,product-dev}`. First body must be **static JSON** (`context_data` an object). Expressions only after a trigger exists.

---

### YOUR NEXT STEP

When asked to implement (not before), generate:

1. Root `docker-compose.yml` — n8n + Go `agents` only, memory caps, `agent-network`, host Postgres via `host.docker.internal`.
2. `agents/Dockerfile` (multi-stage, `linux/arm64` static binary) and `agents/go.mod`.
3. `agents/cmd/server/main.go` — department routes, timeouts, health endpoint.
4. Department packages + `internal/llm` provider switch, contract-compliant JSON.

Follow `SERVER_INIT_INSTRUCTIONS.md` for host Postgres, firewall, and boot order.
