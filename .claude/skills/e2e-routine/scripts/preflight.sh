#!/usr/bin/env bash
# Preflight for an e2e routine run. Prints one line per check; exits 1 when a
# hard prerequisite is missing. Never prints the license.
set -u
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
cd "$REPO_ROOT" || exit 1

fail=0
ok()   { printf 'OK    %-28s %s\n' "$1" "$2"; }
warn() { printf 'WARN  %-28s %s\n' "$1" "$2"; }
bad()  { printf 'FAIL  %-28s %s\n' "$1" "$2"; fail=1; }

v=$(node --version 2>/dev/null || true)
case "$v" in v2[4-9].*|v[3-9]*) ok node "$v" ;; "") bad node "missing" ;; *) warn node "$v (CI uses 24.18.1; build_init.sh requires >= 24.15)" ;; esac

v=$(pnpm --version 2>/dev/null || true)
[ -n "$v" ] && ok pnpm "$v (packageManager pins 11.17.0; pnpm self-switches)" || bad pnpm "missing"

v=$(go version 2>/dev/null | awk '{print $3}')
[ -n "$v" ] && ok go "$v (go.mod 1.27.1; GOTOOLCHAIN=$GOTOOLCHAIN)" || bad go "missing"

command -v psql >/dev/null && ok psql "$(psql --version)" || bad psql "missing (apt install postgresql-client)"
if command -v pg_ctl >/dev/null && command -v initdb >/dev/null; then
  ok pg_ctl/initdb "$(command -v pg_ctl) ($(pg_ctl --version))"
else
  bad pg_ctl/initdb "missing from PATH; Bytebase boots Postgres with them (apt install postgresql-16; lib.sh adds /usr/lib/postgresql/*/bin)"
fi

if [ "$(id -u)" = 0 ]; then
  id bytebase >/dev/null 2>&1 && ok root-user "running as root; 'bytebase' user exists for embedded Postgres" \
    || bad root-user "running as root and no 'bytebase' user: addgroup --gid 113 --system bytebase && adduser --uid 113 --system --ingroup bytebase bytebase"
else
  ok user "$(id -un) (non-root)"
fi

[ "$(e2e_license_available)" = yes ] && ok license "available (env or frontend/.env.dev; value not shown)" || bad license "no BYTEBASE_E2E_LICENSE and no BB_DEV_ENTERPRISE_LICENSE in frontend/.env.dev"

if [ -n "${BYTEBASE_BROWSER_CHANNEL:-}" ]; then
  ok browser "channel=$BYTEBASE_BROWSER_CHANNEL"
else
  cache=""
  for d in "$HOME/.cache/ms-playwright" "$HOME/Library/Caches/ms-playwright" "${PLAYWRIGHT_BROWSERS_PATH:-}"; do
    [ -n "$d" ] && [ -d "$d" ] && cache="$d" && break
  done
  if [ -n "$cache" ] && ls "$cache" 2>/dev/null | grep -q '^chromium'; then
    ok chromium "$(ls "$cache" | grep '^chromium' | tr '\n' ' ')"
  else
    warn chromium "no cached build; the harness downloads it (needs cdn.playwright.dev allowed)"
  fi
fi

for p in 18234 18236 18237; do
  if command -v lsof >/dev/null && lsof -nP -iTCP:$p -sTCP:LISTEN >/dev/null 2>&1; then
    warn "port $p" "in use: $(lsof -nP -iTCP:$p -sTCP:LISTEN | awk 'NR==2{print $1, $2}') (harness falls back to the next range; do NOT kill it)"
  fi
done
[ -f /tmp/bytebase-e2e-pid ] && warn pid-file "/tmp/bytebase-e2e-pid exists: $(cat /tmp/bytebase-e2e-pid)" || ok pid-file "none"

df_out=$(df -Pk . | awk 'NR==2{printf "%.1f GiB free", $4/1024/1024}')
ok disk "$df_out"
ok cpus "$(cpu_count)"
if [ -r /proc/meminfo ]; then ok memory "$(awk '/MemTotal/{printf "%.1f GiB", $2/1024/1024}' /proc/meminfo)"; fi

git fetch -q origin main 2>/dev/null && ok origin/main "$(git rev-parse --short origin/main) $(git log -1 --format=%cd --date=short origin/main)" || warn origin/main "fetch failed; using HEAD $(git rev-parse --short HEAD)"

[ -x ./bytebase-build/bytebase ] && ok binary "present ($(stat -c %y ./bytebase-build/bytebase 2>/dev/null || stat -f %Sm ./bytebase-build/bytebase))" || ok binary "absent (build.sh builds it)"

exit $fail
