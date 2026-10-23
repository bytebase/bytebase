#!/usr/bin/env bash
# Build the frontend-embedded Bytebase binary the harness boots.
#
#   build.sh [results-dir]
#
# Skips the work when ./bytebase-build/bytebase already matches HEAD and the
# server build script (stamp file) and the tree is clean. The Go step is
# frontend/tests/e2e/build-server.sh: a dev build, so the dev license in
# frontend/.env.dev validates, with dev-only product switches set to their
# release values. Records timings in <results-dir>/build-timing.txt.
set -eu
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
cd "$REPO_ROOT"

RESULTS="${1:-$E2E_RESULTS_ROOT/build}"
mkdir -p "$RESULTS" bytebase-build
STAMP=bytebase-build/.built-sha
BIN=bytebase-build/bytebase
SERVER_BUILD=frontend/tests/e2e/build-server.sh
# A change to the server build script changes the binary without a commit.
stamp="$(git rev-parse HEAD) $(git hash-object "$SERVER_BUILD")"
# Any uncommitted change that can reach the binary invalidates the cached
# build: the whole frontend tree (package.json, lockfile, Tailwind and Vite
# config, plugins), not only frontend/src. The e2e specs do not reach it.
dirty="$(git status --porcelain -- frontend backend proto go.mod go.sum ':!frontend/tests/e2e' | head -1)"

timing="$RESULTS/build-timing.txt"
if [ -x "$BIN" ] && [ -f "$STAMP" ] && [ "$(cat "$STAMP")" = "$stamp" ] && [ -z "$dirty" ] && [ "${FORCE_BUILD:-0}" != 1 ]; then
  log "binary already built from $stamp; skipping (FORCE_BUILD=1 to rebuild)"
  # Every run directory gets a timing record, a reused binary included.
  printf '%-16s %5ss\n' "cached" 0 >"$timing"
  exit 0
fi

: >"$timing"
phase() { # phase <name> <cmd...>
  local name="$1"; shift
  local t0=$(date +%s)
  log "$name ..."
  "$@"
  local dt=$(( $(date +%s) - t0 ))
  printf '%-16s %5ss\n' "$name" "$dt" | tee -a "$timing" >&2
}

phase pnpm-install pnpm --dir frontend install --frozen-lockfile
phase pnpm-release pnpm --dir frontend release
phase go-build bash "$SERVER_BUILD" "$BIN"
echo "$stamp" >"$STAMP"
log "built $BIN from $stamp"
