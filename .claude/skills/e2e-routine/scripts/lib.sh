#!/usr/bin/env bash
# Shared helpers for the e2e routine scripts. Source it; do not run it.

set -o pipefail

E2E_SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "$E2E_SKILL_DIR/../../.." && pwd)"
E2E_RESULTS_ROOT="${E2E_RESULTS_ROOT:-/tmp/e2e-routine}"
E2E_LEDGER_BRANCH="${E2E_LEDGER_BRANCH:-claude/e2e-ledger}"

# The cloud image ships Node 20-22 on PATH; setup-env.sh installs 24 under /opt/node24.
[ -d /opt/node24/bin ] && export PATH="/opt/node24/bin:$PATH"
# Bytebase starts its metadata and sample Postgres with the system `pg_ctl` and
# `initdb` from PATH. Debian/Ubuntu keep them under /usr/lib/postgresql/<ver>/bin.
if ! command -v pg_ctl >/dev/null 2>&1; then
  for d in /usr/lib/postgresql/*/bin; do
    [ -x "$d/pg_ctl" ] && pg_bin="$d"
  done
  [ -n "${pg_bin:-}" ] && export PATH="$pg_bin:$PATH"
fi
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
# The embedded Postgres needs a UTF-8 locale (initdb runs with LC_ALL=en_US.UTF-8).
export LANG="${LANG:-en_US.UTF-8}"
export LC_ALL="${LC_ALL:-en_US.UTF-8}"

log() { printf '[e2e-routine] %s\n' "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

cpu_count() {
  if command -v nproc >/dev/null 2>&1; then nproc; else sysctl -n hw.ncpu; fi
}

# Prints the enterprise license to stdout, never to a log. Order: the
# BYTEBASE_E2E_LICENSE variable, then the dev license committed in
# frontend/.env.dev (dev-signed: the binary must be built WITHOUT the
# `release` tag for it to validate).
e2e_license() {
  if [ -n "${BYTEBASE_E2E_LICENSE:-}" ]; then
    printf '%s' "$BYTEBASE_E2E_LICENSE"
    return 0
  fi
  local f="$REPO_ROOT/frontend/.env.dev"
  [ -f "$f" ] || return 1
  local v
  v="$(sed -n 's/^BB_DEV_ENTERPRISE_LICENSE=//p' "$f" | head -1 | tr -d '"'"'"'"')"
  [ -n "$v" ] || return 1
  printf '%s' "$v"
}

# "yes"/"no" without revealing the value.
e2e_license_available() {
  if e2e_license >/dev/null 2>&1; then echo yes; else echo no; fi
}

# origin/main as fetched now. When the fetch fails it is HEAD, never an older
# origin/main left by an earlier fetch, which would test or skip a stale SHA.
main_sha() {
  if git -C "$REPO_ROOT" fetch -q origin main 2>/dev/null; then
    git -C "$REPO_ROOT" rev-parse origin/main
  else
    log "fetch of origin/main failed; using HEAD"
    git -C "$REPO_ROOT" rev-parse HEAD
  fi
}

run_id_now() { date -u +%Y%m%dT%H%M%SZ; }
