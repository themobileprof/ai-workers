#!/usr/bin/env bash
# Create the aiworkers Postgres role/database and merge admin env into .env.
# Run on the VM as cursor (uses sudo for postgres). Does not print secrets.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${ADMIN_ENV_FILE:-$ROOT/.env}"
DB_NAME="${ADMIN_DB_NAME:-aiworkers}"
DB_USER="${ADMIN_DB_USER:-aiworkers}"

require_sudo() {
  if ! sudo -n true 2>/dev/null; then
    echo "Need passwordless sudo (or a sudo prompt) to create the Postgres role." >&2
    exit 1
  fi
}

pg_hba_file() {
  sudo -u postgres psql -tAc "SHOW hba_file" | tr -d '[:space:]'
}

ensure_pg_hba() {
  local hba="$1"
  local line
  local changed=0
  for line in \
    "host    ${DB_NAME}    ${DB_USER}    127.0.0.1/32    scram-sha-256" \
    "host    ${DB_NAME}    ${DB_USER}    ::1/128         scram-sha-256" \
    "host    ${DB_NAME}    ${DB_USER}    172.16.0.0/12   scram-sha-256"; do
    if ! sudo grep -qxF "$line" "$hba"; then
      printf '\n%s\n' "$line" | sudo tee -a "$hba" >/dev/null
      changed=1
    fi
  done
  if [[ "$changed" -eq 1 ]]; then
    sudo systemctl reload postgresql
  fi
}

env_has() {
  local key="$1"
  [[ -f "$ENV_FILE" ]] && grep -qE "^${key}=" "$ENV_FILE"
}

append_env() {
  local key="$1"
  local value="$2"
  if env_has "$key"; then
    return
  fi
  mkdir -p "$(dirname "$ENV_FILE")"
  touch "$ENV_FILE"
  chmod 600 "$ENV_FILE"
  printf '\n%s=%s\n' "$key" "$value" >> "$ENV_FILE"
}

require_sudo

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE — copy .env.example first." >&2
  exit 1
fi

TOKEN="$(python3 - <<'PY'
import secrets
print(secrets.token_hex(32))
PY
)"
BOOT_PASS="$(python3 - <<'PY'
import secrets, string
alphabet = string.ascii_letters + string.digits
print("".join(secrets.choice(alphabet) for _ in range(20)))
PY
)"

if ! env_has "ADMIN_DATABASE_URL"; then
  PASS="$(python3 - <<'PY'
import secrets, string
alphabet = string.ascii_letters + string.digits
print("".join(secrets.choice(alphabet) for _ in range(28)))
PY
)"
  if sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" | grep -q 1; then
    sudo -u postgres psql -v ON_ERROR_STOP=1 \
      -c "ALTER USER ${DB_USER} WITH PASSWORD '${PASS}';"
  else
    sudo -u postgres psql -v ON_ERROR_STOP=1 \
      -c "CREATE USER ${DB_USER} WITH PASSWORD '${PASS}';"
  fi
  append_env "ADMIN_DATABASE_URL" "postgres://${DB_USER}:${PASS}@host.docker.internal:5432/${DB_NAME}?sslmode=disable"
elif ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" | grep -q 1; then
  echo "ADMIN_DATABASE_URL is set but role ${DB_USER} is missing. Fix Postgres or unset the URL." >&2
  exit 1
fi

if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" | grep -q 1; then
  sudo -u postgres psql -v ON_ERROR_STOP=1 -c "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};"
fi

ensure_pg_hba "$(pg_hba_file)"

append_env "ADMIN_BOOTSTRAP_NAME" "Samuel"
append_env "ADMIN_BOOTSTRAP_PHONE" "2348033954301"
append_env "ADMIN_BOOTSTRAP_EMAIL" "info@themobileprof.com"
append_env "ADMIN_BOOTSTRAP_PASSWORD" "$BOOT_PASS"
append_env "INTERNAL_API_TOKEN" "$TOKEN"
append_env "ADMIN_COOKIE_SECURE" "true"

echo "Admin Postgres database ${DB_NAME} is ready."
echo "Login phone: 2348033954301"
echo "Password is ADMIN_BOOTSTRAP_PASSWORD in ${ENV_FILE} (not printed)."
echo "Grant web_admin / whatsapp_accounts on the desk after first sign-in if you add people."
