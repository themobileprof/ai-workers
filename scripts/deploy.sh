#!/usr/bin/env bash
# Apply this checkout on the OCI VM. GitHub Release CD and the laptop both call this.
# Never rsync .env — secrets stay on the host.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="${DEPLOY_SSH_HOST:-${N8N_SSH_HOST:-oci-ai-workers}}"
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

apply_on_host() {
  local root="$1"
  cd "$root"
  docker compose up -d --build --remove-orphans
  wait_health
  ./scripts/sync-n8n-workflows.sh
}

on_vm() {
  [[ -d /home/cursor/ai-workers/docker-compose.yml ]] && [[ "$(id -un)" == "cursor" ]]
}

if on_vm; then
  apply_on_host /home/cursor/ai-workers
  exit 0
fi

rsync -az --delete \
  --exclude '.git/' \
  --exclude '.env' \
  --exclude '.cursor/' \
  -e "ssh $SSH_OPTS" \
  "$ROOT/" \
  "${REMOTE_HOST}:${REMOTE_DIR}/"

ssh $SSH_OPTS "$REMOTE_HOST" \
  "chmod +x ${REMOTE_DIR}/scripts/deploy.sh ${REMOTE_DIR}/scripts/sync-n8n-workflows.sh && ${REMOTE_DIR}/scripts/deploy.sh"
