#!/usr/bin/env bash
# Import n8n/workflows/*.json into the running n8n container.
# From this laptop: rsync then SSH. On the VM: import only.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="${N8N_SSH_HOST:-oci-ai-workers}"
REMOTE_DIR="${N8N_REMOTE_DIR:-/home/cursor/ai-workers}"
# Execute Workflow refuses inactive children. Webhooks also stay dark after import.
DEFAULT_PUBLISH="paystackPaid000001,opsTelegram00001,custWhatsApp0001,inbdEmailImap0001,booksWriteDoc0001,crmUpsertCont0001,emailOutbox000001,commWeeklyPing01,acadTelegram00001"
PUBLISH="${N8N_PUBLISH:-$DEFAULT_PUBLISH}"

instance_field() {
  python3 - "$1" "$2" <<'PY'
import json, sys
print(json.load(open(sys.argv[1])).get(sys.argv[2]) or "")
PY
}

# n8n 2.x import --userId tries to re-own credentials/workflows and dies when they
# already belong to that user ("can't be re-owned"). --projectId is the 2.x owner.
# Re-own on an existing id is an update, not a failure.
import_one() {
  local cpath="$1"
  local project_id="$2"
  local out st=0
  if [[ -n "$project_id" ]]; then
    out="$(docker compose exec -T n8n n8n import:workflow --input="$cpath" --projectId="$project_id" 2>&1)" || st=$?
  else
    out="$(docker compose exec -T n8n n8n import:workflow --input="$cpath" 2>&1)" || st=$?
  fi
  printf '%s\n' "$out"
  if [[ "$st" -eq 0 ]]; then
    return 0
  fi
  if [[ "$out" == *"can't be re-owned"* ]]; then
    echo "already owned; retry $cpath without owner flags" >&2
    if ! docker compose exec -T n8n n8n import:workflow --input="$cpath"; then
      echo "still owned; $cpath already on this instance" >&2
    fi
    return 0
  fi
  return "$st"
}

import_on_this_host() {
  local root="$1"
  local project_id
  project_id="$(instance_field "$root/n8n/instance.json" projectId)"
  cd "$root"

  if ! docker compose exec -T n8n test -d /home/node/workflows; then
    echo "n8n container is missing /home/node/workflows. Recreate n8n after updating docker-compose.yml." >&2
    exit 1
  fi

  local f base
  for f in "$root/n8n/workflows/"*.json; do
    base="$(basename "$f")"
    echo "import $base"
    import_one "/home/node/workflows/$base" "$project_id"
  done

  if [[ -n "$PUBLISH" ]]; then
    IFS=',' read -r -a ids <<< "$PUBLISH"
    for id in "${ids[@]}"; do
      [[ -z "$id" ]] && continue
      docker compose exec -T n8n n8n publish:workflow --id="$id"
    done
  fi

  docker compose exec -T n8n n8n list:workflow
}

on_vm() {
  [[ -d /home/cursor/ai-workers/n8n/workflows ]] && [[ "$(id -un)" == "cursor" ]]
}

if on_vm; then
  import_on_this_host /home/cursor/ai-workers
  exit 0
fi

rsync -az --delete \
  -e "ssh -o BatchMode=yes" \
  "$ROOT/n8n/" \
  "${REMOTE_HOST}:${REMOTE_DIR}/n8n/"
rsync -az \
  -e "ssh -o BatchMode=yes" \
  "$ROOT/scripts/sync-n8n-workflows.sh" \
  "${REMOTE_HOST}:${REMOTE_DIR}/scripts/sync-n8n-workflows.sh"

ssh -o BatchMode=yes "$REMOTE_HOST" "chmod +x ${REMOTE_DIR}/scripts/sync-n8n-workflows.sh && N8N_PUBLISH=$(printf '%q' "$PUBLISH") ${REMOTE_DIR}/scripts/sync-n8n-workflows.sh"
