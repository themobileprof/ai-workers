# REMOTE SERVER CONTEXT & SETUP INSTRUCTIONS

## 1. Environment State

- You are running inside a Remote SSH workspace directly on an Oracle Cloud (OCI) Ubuntu server (`130.61.144.174`, **aarch64**).
- **Work user:** `cursor` (not `ubuntu`). `ubuntu` is the cloud-init bootstrap account; all stack work happens as `cursor` (`ssh cursor@130.61.144.174` or Host `oci-ai-workers`).
- System spec limits: 1 OCPU, 6 GB RAM.

## 2. Objective

Act as an automated DevOps engineer. Construct, configure, and boot the stack described in `CURSOR_INSTRUCTIONS.md`.

**Hard rules:**

- PostgreSQL runs as a native systemd service on the host. It must never be a Docker container and must never appear in `docker-compose.yml`.
- The worker fleet is a **Go** static binary, not FastAPI/Python.
- Do not install a Go toolchain, Python, or Node on the host. Compile Go **inside** the image. n8n brings Node inside its own image.
- Caddy, when added, is native on the host (like Postgres), not a Compose service.

## 3. Mandatory Steps

### Step A: System Inspection & Packages

1. Verify `apt` is available. Prefer `sudo apt update` and install only what we need. Do **not** run an unattended `apt upgrade -y` unless I explicitly ask — kernel upgrades on a 1 OCPU box can stall SSH and demand a reboot.

2. Install:

   - Docker Engine + **Compose v2 plugin** (`docker-compose-v2` / `docker compose`), not the legacy Python `docker-compose` v1 package.
   - `git`
   - `postgresql` and `postgresql-contrib`
   - `ufw` if missing

   Do **not** install `golang-go`, `python3-pip`, or `nodejs` on the host.

3. Enable Docker on boot: `sudo systemctl enable --now docker`.

4. Add `cursor` to the `docker` group after Docker is installed, then use a new login/session so it takes effect.

5. Enable Postgres on boot: `sudo systemctl enable --now postgresql`.

### Step B: Native PostgreSQL (host, not Docker)

n8n will connect from a container using `host.docker.internal` (Compose `extra_hosts: host.docker.internal:host-gateway`).

1. Create a dedicated role and database (example names — put the password in a root `.env`, never commit it):

   ```bash
   sudo -u postgres psql -c "CREATE USER n8n WITH PASSWORD 'CHANGE_ME';"
   sudo -u postgres psql -c "CREATE DATABASE n8n OWNER n8n;"
   ```

2. Tune `/etc/postgresql/*/main/postgresql.conf` for a **shared** 6 GB box (not a dedicated DB server):

   ```
   listen_addresses = '*'
   max_connections = 20
   shared_buffers = 256MB
   work_mem = 4MB
   maintenance_work_mem = 64MB
   effective_cache_size = 512MB
   ```

   `listen_addresses = '*'` is acceptable **only** because 5432 stays closed on UFW and on the OCI Security List. Binding solely to `127.0.0.1` would block Docker.

3. Restrict `/etc/postgresql/*/main/pg_hba.conf` to localhost and Docker bridges:

   ```
   local   all    postgres     peer
   host    n8n    n8n          127.0.0.1/32    scram-sha-256
   host    n8n    n8n          ::1/128         scram-sha-256
   host    n8n    n8n          172.16.0.0/12   scram-sha-256
   ```

4. `sudo systemctl restart postgresql` and confirm `systemctl is-active postgresql`.

### Step C: Networking & Firewall

Publish **only** n8n on bootstrap. The Go worker is internal. Postgres is internal.

OCI Ubuntu images put an `INPUT REJECT icmp-host-prohibited` rule **in front of UFW**. UFW alone will not open 5678 or Docker→Postgres. Persist ACCEPT rules before that REJECT (`/usr/local/sbin/ai-workers-iptables.sh` + `ai-workers-iptables.service`): TCP 5678 from anywhere (bootstrap), TCP 5432 from `172.16.0.0/12` only.

```bash
sudo ufw allow OpenSSH
sudo ufw allow 5678/tcp
sudo ufw allow from 172.16.0.0/12 to any port 5432 proto tcp
sudo ufw --force enable
```

Do **not** open port `8000`. Do **not** allow 5432 from the public internet.

When a domain is pointed at this instance, install **Caddy** on the host, allow `80`/`443`, proxy to `127.0.0.1:5678`, rebind n8n to localhost, and remove 5678 from iptables/UFW and the OCI Security List.

The VCN **Security List / NSG** must allow TCP 5678 (and later 80/443). Do not allow 8000 or 5432 there.

### Step D: File Generation & Workspace Setup

Create application files in this folder:

1. `docker-compose.yml` with **two** services only:

   - **n8n** — pinned image tag (not `latest`); log pruning env vars; `N8N_ENCRYPTION_KEY`; Postgres at `host.docker.internal` mapped to the compose bridge gateway `172.28.0.1` (not Docker `host-gateway`/`docker0`, which is down when nothing uses the default bridge); memory limit **1536M**; publish `5678:5678`; join `agent-network`.
   - **agents** (Go) — build `./agents` for **linux/arm64**; memory limit **256M**; `GOMAXPROCS=1`; **no** `ports:` mapping; join `agent-network`. n8n calls `http://agents:8000`.

   Do not add a `postgres` service. Do not add Caddy as a service. Do not use n8n queue mode / Redis.

2. `agents/` Go layout: `Dockerfile` (multi-stage static binary), `go.mod`, `cmd/server/main.go`, `internal/contract`, `internal/llm`, `internal/departments/{internalops,growth,productdev}` as specified in `CURSOR_INSTRUCTIONS.md`.

### Step E: Execution & Boot Validation

1. `docker compose up --build -d` (Compose v2 syntax).

2. Health checks — **two containers + one host service**, not three containers:

   - `docker compose ps` — `n8n` and `agents` running.
   - `systemctl is-active postgresql` — `active`.
   - From the n8n container, Postgres is reachable at `host.docker.internal:5432`.
   - Go worker is reachable from n8n at `http://agents:8000` (health route) and is **not** listening on the public NIC.

3. If the VM is under memory pressure, do not raise Postgres `shared_buffers`. Lower the n8n memory cap or prune executions first. Do not "fix" pressure by adding Python workers.
