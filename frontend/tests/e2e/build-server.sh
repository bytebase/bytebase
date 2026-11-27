#!/usr/bin/env bash
# Build the Bytebase server binary the e2e suite boots.
#
#   frontend/tests/e2e/build-server.sh [output]   # default ./bytebase-build/bytebase
#
# The suite tests what release builds ship, but a release build only accepts
# a license signed with the production key, and the suite uses the dev
# license. So this builds a dev binary (no `release` tag) and, through Go's
# -overlay, sets each dev-only product switch in profile_dev.go back to its
# release value. The source tree is not modified. Embed the frontend first:
# `(cd frontend && pnpm release)`.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
out="${1:-$repo/bytebase-build/bytebase}"
profile="$repo/backend/bin/server/cmd/profile_dev.go"

overlay_dir="$(mktemp -d)"
trap 'rm -rf "$overlay_dir"' EXIT

# A dev build forces SQL review V2 on. Dropping that line leaves the release
# value, which turns V2 on with --saas (Bytebase Cloud) and keeps V1 for the
# self-hosted server the suite boots. A missing line means the switch moved:
# fail rather than silently testing V2.
if [ "$(grep -c '^[[:space:]]*p\.SQLReviewV2 = true$' "$profile")" != 1 ]; then
  echo "build-server.sh: expected one 'p.SQLReviewV2 = true' line in $profile; update this script" >&2
  exit 1
fi
sed '/^[[:space:]]*p\.SQLReviewV2 = true$/d' "$profile" >"$overlay_dir/profile_dev.go"
node -e 'require("fs").writeFileSync(process.argv[3], JSON.stringify({ Replace: { [process.argv[1]]: process.argv[2] } }))' \
  "$profile" "$overlay_dir/profile_dev.go" "$overlay_dir/overlay.json"

cpus="$(nproc 2>/dev/null || sysctl -n hw.ncpu)"
mkdir -p "$(dirname "$out")"
cd "$repo"
go build -overlay="$overlay_dir/overlay.json" -tags embed_frontend -ldflags "-w -s" -p="$cpus" \
  -o "$out" ./backend/bin/server/main.go
