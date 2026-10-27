#!/usr/bin/env bash
# Shared helpers for the e2e routine scripts. Source it; do not run it.

set -o pipefail

E2E_SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "$E2E_SKILL_DIR/../../.." && pwd)"
# One results root per checkout: run-dir.sh keeps its CURRENT pointer there,
# and two worktrees running at once must not take over each other's run.
E2E_RESULTS_ROOT="${E2E_RESULTS_ROOT:-/tmp/e2e-routine/$(basename "$REPO_ROOT")-$(printf '%s' "$REPO_ROOT" | cksum | cut -d' ' -f1)}"
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

# The commit to test: origin/main as fetched now. When the fetch fails it is
# HEAD, never an older origin/main left by an earlier fetch, which would test
# or skip a stale SHA. Not for "is X on main": see main_ref_strict.
main_sha() {
  if git -C "$REPO_ROOT" fetch -q origin main 2>/dev/null; then
    git -C "$REPO_ROOT" rev-parse origin/main
  else
    log "fetch of origin/main failed; using HEAD"
    git -C "$REPO_ROOT" rev-parse HEAD
  fi
}

# The main to judge membership against (references/ledger.md): fetched now,
# else the local origin/main, since an older main is still main. Never HEAD,
# which would let any branch pass. Exit 1 when there is no origin/main at all.
main_ref_strict() {
  git -C "$REPO_ROOT" fetch -q origin main 2>/dev/null \
    || log "fetch of origin/main failed; checking against the local origin/main"
  git -C "$REPO_ROOT" rev-parse -q --verify 'origin/main^{commit}'
}

# A shallow clone lacks the history that ancestry checks and attribution
# need; deepen it once to E2E_HISTORY_DAYS of main.
ensure_history() {
  [ "$(git -C "$REPO_ROOT" rev-parse --is-shallow-repository 2>/dev/null)" = true ] || return 0
  git -C "$REPO_ROOT" fetch -q --shallow-since="${E2E_HISTORY_DAYS:-120} days ago" origin main 2>/dev/null \
    || log "could not deepen the shallow clone; older ledger entries may not resolve"
}

# on_main <sha> [main]: the SHA is a commit on main. A SHA this clone lacks is
# fetched first (GitHub serves any published commit), so an absent object is
# never read as "not on main"; one GitHub does not have fails. In a shallow
# clone a fetched commit is not yet linked to main, so main is deepened to
# just before the commit's date and the check is made again.
on_main() {
  local sha="$1" main="${2:-}" when
  [ -n "$sha" ] || return 1
  [ -n "$main" ] || main="$(main_ref_strict)" || return 1
  ensure_history
  git -C "$REPO_ROOT" cat-file -e "$sha^{commit}" 2>/dev/null \
    || git -C "$REPO_ROOT" fetch -q origin "$sha" 2>/dev/null || return 1
  git -C "$REPO_ROOT" merge-base --is-ancestor "$sha" "$main" && return 0
  [ "$(git -C "$REPO_ROOT" rev-parse --is-shallow-repository 2>/dev/null)" = true ] || return 1
  when="$(git -C "$REPO_ROOT" log -1 --format=%ct "$sha")" || return 1
  git -C "$REPO_ROOT" fetch -q --shallow-since="$((when - 86400))" origin main 2>/dev/null || return 1
  git -C "$REPO_ROOT" merge-base --is-ancestor "$sha" "$main"
}

# What the binary is built from; the e2e tree is excluded because specs are
# read at run time and an adaptation changes only them.
PRODUCT_PATHSPEC=(frontend backend proto go.mod go.sum ':!frontend/tests/e2e')

# The main commit the checkout's product files come from (references/ledger.md):
# HEAD when it is on main, the merge base when the commits above it, and any
# uncommitted change, touch only the e2e tree or .claude/. Exit 1 when the
# product files match no main commit.
tested_sha() {
  local main base
  main="$(main_ref_strict)" || return 1
  ensure_history
  base="$(git -C "$REPO_ROOT" merge-base HEAD "$main" 2>/dev/null)" || return 1
  git -C "$REPO_ROOT" diff --quiet "$base" HEAD -- "${PRODUCT_PATHSPEC[@]}" || return 1
  [ -z "$(git -C "$REPO_ROOT" status --porcelain -- "${PRODUCT_PATHSPEC[@]}" | head -1)" ] || return 1
  echo "$base"
}

run_id_now() { date -u +%Y%m%dT%H%M%SZ; }
