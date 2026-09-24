#!/usr/bin/env bash
# Apply this checkout on the OCI VM. GitHub Release CD and the laptop both call this.
# Never rsync .env — secrets stay on the host.
# Agents are a linux/arm64 binary built on CI or the laptop. The VM must not compile Go.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [[ -n "${DEPLOY_SSH_HOST:-}" ]]; then
  REMOTE_HOST="$DEPLOY_SSH_HOST"
elif [[ -n "${DEPLOY_HOST:-}" ]]; then
  REMOTE_HOST="${DEPLOY_USER:-cursor}@${DEPLOY_HOST}"
else
  REMOTE_HOST="${N8N_SSH_HOST:-oci-ai-workers}"
fi
REMOTE_DIR="${DEPLOY_REMOTE_DIR:-${N8N_REMOTE_DIR:-/home/cursor/ai-workers}}"
SSH_OPTS="-o BatchMode=yes -o ServerAliveInterval=15 -o ServerAliveCountMax=8"

wait_health() {
  local i
  for i in $(seq 1 45); do
    if docker compose exec -T n8n wget -qO- http://agents:8000/health 2>/dev/null | grep -q '"status":"ok"'; then
      docker compose exec -T n8n wget -qO- http://agents:8000/health
      echo
      return 0
    fi
    sleep 2
  done
  echo "agents health did not return ok" >&2
  docker compose ps >&2 || true
  docker compose logs --tail=80 agents >&2 || true
  return 1
}

ensure_agents_binary() {
  local out="$ROOT/agents/dist/agents"
  if [[ -f "$out" ]]; then
    return 0
  fi
  if ! command -v go >/dev/null; then
    echo "need $out (CI linux/arm64 build) or a Go toolchain to cross-compile" >&2
    exit 1
  fi
  mkdir -p "$ROOT/agents/dist"
  echo "cross-compiling linux/arm64 agents → $out"
  (cd "$ROOT/agents" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/agents ./cmd/server)
}

install_caddy() {
  local src="$1/Caddyfile"
  local dest="/etc/caddy/conf.d/workers.caddy"
  if [[ ! -f "$src" ]]; then
    return 0
  fi
  if [[ -f "$dest" ]] && cmp -s "$src" "$dest"; then
    return 0
  fi
  if ! sudo -n true 2>/dev/null; then
    echo "Caddyfile changed. Copy $src to $dest, then: sudo caddy validate --config /etc/caddy/Caddyfile && sudo systemctl reload caddy" >&2
    return 0
  fi
  sudo cp "$src" "$dest"
  sudo caddy validate --config /etc/caddy/Caddyfile
  sudo systemctl reload caddy
  echo "reloaded caddy ($dest)"
}

compose_up() {
  local root="$1"
  local bin="$root/agents/dist/agents"
  if [[ ! -f "$bin" ]]; then
    echo "missing $bin — ship a linux/arm64 binary; do not compile Go on this VM" >&2
    exit 1
  fi
  chmod +x "$bin" || true
  docker compose -f docker-compose.yml -f docker-compose.deploy.yml up -d --build --remove-orphans
}

apply_on_host() {
  local root="$1"
  cd "$root"
  compose_up "$root"
  wait_health
  install_caddy "$root"
  ./scripts/sync-n8n-workflows.sh
}

on_vm() {
  [[ -f /home/cursor/ai-workers/docker-compose.yml ]] && [[ "$(id -un)" == "cursor" ]]
}

if on_vm; then
  apply_on_host /home/cursor/ai-workers
  exit 0
fi

ensure_agents_binary

echo "rsync to ${REMOTE_HOST}:${REMOTE_DIR}"

rsync -az --delete \
  --exclude '.git/' \
  --exclude '.env' \
  --exclude '.cursor/' \
  -e "ssh $SSH_OPTS" \
  "$ROOT/" \
  "${REMOTE_HOST}:${REMOTE_DIR}/"

ssh $SSH_OPTS "$REMOTE_HOST" \
  "chmod +x ${REMOTE_DIR}/scripts/deploy.sh ${REMOTE_DIR}/scripts/sync-n8n-workflows.sh && ${REMOTE_DIR}/scripts/deploy.sh"
