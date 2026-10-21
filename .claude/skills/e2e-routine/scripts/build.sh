#!/usr/bin/env bash
# Build the frontend-embedded Bytebase binary the harness boots.
#
#   build.sh [results-dir]
#
# Skips the work when ./bytebase-build/bytebase already matches HEAD (stamp
# file) and the tree is clean. Builds WITHOUT the `release` tag on purpose: the
# dev license in frontend/.env.dev is signed with the dev key, which only a
# non-release build accepts. Records timings in <results-dir>/build-timing.txt.
set -eu
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
cd "$REPO_ROOT"

RESULTS="${1:-$E2E_RESULTS_ROOT/build}"
mkdir -p "$RESULTS" bytebase-build
STAMP=bytebase-build/.built-sha
BIN=bytebase-build/bytebase
head="$(git rev-parse HEAD)"
dirty="$(git status --porcelain -- frontend/src backend proto go.mod go.sum | head -1)"

if [ -x "$BIN" ] && [ -f "$STAMP" ] && [ "$(cat "$STAMP")" = "$head" ] && [ -z "$dirty" ] && [ "${FORCE_BUILD:-0}" != 1 ]; then
  log "binary already built from $head; skipping (FORCE_BUILD=1 to rebuild)"
  exit 0
fi

timing="$RESULTS/build-timing.txt"
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
phase go-build go build -tags embed_frontend -ldflags "-w -s" -p="$(cpu_count)" -o "$BIN" ./backend/bin/server/main.go
echo "$head" >"$STAMP"
log "built $BIN from $head"
