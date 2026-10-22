#!/usr/bin/env bash
# Run the licensed Playwright suite detached from the calling shell, so a
# Bash-tool timeout cannot kill it, and wait for it.
#
#   run-suite.sh start [run-id] [-- playwright args...]   # detach; prints the results dir
#   run-suite.sh wait  <results-dir> [max-minutes]        # block with a heartbeat; exits with the suite's code
#   run-suite.sh status <results-dir>                     # one-line progress
#
# Layout of <results-dir>: playwright.log, server.log, report.json (Playwright
# JSON reporter), test-results/ (traces, screenshots, error-context.md),
# exit-code, meta.json. Every test gets three attempts (--retries=2): red only
# when all three fail, flaky when a retry passes. A run is capped at
# E2E_GLOBAL_TIMEOUT_MS (default 2 h; a healthy run takes about 1 h) because a
# night of hung tests otherwise multiplies 3 attempts by each test's timeout.
# The license comes from lib.sh's e2e_license and is exported only into the
# detached process.
set -u
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

cmd="${1:-}"; shift || true
case "$cmd" in
start)
  run_id="${1:-$(run_id_now)}"; [ "${1:-}" = "--" ] && run_id="$(run_id_now)" || shift || true
  [ "${1:-}" = "--" ] && shift
  RESULTS="$E2E_RESULTS_ROOT/$run_id"
  mkdir -p "$RESULTS"
  cd "$REPO_ROOT"
  [ -x bytebase-build/bytebase ] || die "bytebase-build/bytebase missing; run build.sh first"
  license="$(e2e_license)" || die "no license available (BYTEBASE_E2E_LICENSE or frontend/.env.dev)"

  # Wrapper so the disposable server's stdout/stderr land in server.log; the
  # harness spawns BYTEBASE_BIN with stdio ignored.
  # As root (cloud sessions), Bytebase runs embedded Postgres as the `bytebase`
  # user, which must be able to traverse the harness's mkdtemp data dir (0700).
  cat >"$RESULTS/bytebase-wrapper.sh" <<EOF
#!/usr/bin/env bash
if [ "\$(id -u)" = 0 ]; then
  prev=""
  for a in "\$@"; do
    if [ "\$prev" = "--data" ] && [ -d "\$a" ]; then chmod 755 "\$a"; fi
    prev="\$a"
  done
fi
exec "$REPO_ROOT/bytebase-build/bytebase" "\$@" >>"$RESULTS/server.log" 2>&1
EOF
  chmod +x "$RESULTS/bytebase-wrapper.sh"

  cat >"$RESULTS/meta.json" <<EOF
{"run_id":"$run_id","sha":"$(git rev-parse HEAD)","short_sha":"$(git rev-parse --short HEAD)","started_at":"$(date -u +%FT%TZ)","host":"$(hostname)","args":"$*"}
EOF

  runner="$RESULTS/runner.sh"
  cat >"$runner" <<EOF
#!/usr/bin/env bash
cd "$REPO_ROOT/frontend" || exit 97
export BYTEBASE_BIN="$RESULTS/bytebase-wrapper.sh"
export BYTEBASE_STARTUP_TIMEOUT="\${BYTEBASE_STARTUP_TIMEOUT:-900000}"
export PLAYWRIGHT_JSON_OUTPUT_NAME="$RESULTS/report.json"
export LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8
export PATH="$PATH"
./node_modules/.bin/playwright test --reporter=list,json --retries=2 --global-timeout="\${E2E_GLOBAL_TIMEOUT_MS:-7200000}" --output="$RESULTS/test-results" $*
code=\$?
echo "\$code" >"$RESULTS/exit-code"
date -u +%FT%TZ >"$RESULTS/finished-at"
exit \$code
EOF
  chmod +x "$runner"

  # setsid (Linux) or a perl setsid (macOS) puts the run in its own session so
  # the Bash tool's process-group kill on timeout does not reach it. On macOS,
  # caffeinate keeps the machine from idle-sleeping under the run: a laptop
  # that slept mid-run once stretched 120 s timeouts to 1000 s each.
  if command -v setsid >/dev/null 2>&1; then
    BYTEBASE_E2E_LICENSE="$license" setsid nohup "$runner" >"$RESULTS/playwright.log" 2>&1 < /dev/null &
  elif command -v caffeinate >/dev/null 2>&1; then
    BYTEBASE_E2E_LICENSE="$license" nohup caffeinate -i perl -MPOSIX -e 'POSIX::setsid() or die; exec @ARGV' "$runner" >"$RESULTS/playwright.log" 2>&1 < /dev/null &
  else
    BYTEBASE_E2E_LICENSE="$license" nohup perl -MPOSIX -e 'POSIX::setsid() or die; exec @ARGV' "$runner" >"$RESULTS/playwright.log" 2>&1 < /dev/null &
  fi
  echo $! >"$RESULTS/runner-pid"
  log "started run $run_id (pid $(cat "$RESULTS/runner-pid")); results in $RESULTS"
  echo "$RESULTS"
  ;;
wait)
  RESULTS="${1:?results dir}"; max_min="${2:-120}"
  deadline=$(( $(date +%s) + max_min*60 ))
  while [ ! -f "$RESULTS/exit-code" ]; do
    if [ "$(date +%s)" -ge "$deadline" ]; then log "gave up after $max_min min"; exit 98; fi
    sleep 60
    "$0" status "$RESULTS"
  done
  code="$(cat "$RESULTS/exit-code")"
  log "suite finished with exit code $code"
  tail -n 5 "$RESULTS/playwright.log"
  exit "$code"
  ;;
status)
  RESULTS="${1:?results dir}"
  lg="$RESULTS/playwright.log"
  [ -f "$lg" ] || { echo "no log yet"; exit 0; }
  done_n=$(grep -cE '^\s+[✓✘-]\s' "$lg" 2>/dev/null || echo 0)
  fail_n=$(grep -cE '^\s+✘\s' "$lg" 2>/dev/null || echo 0)
  last=$(grep -E '^\s+[✓✘-]\s' "$lg" | tail -1 | cut -c1-110)
  state=$([ -f "$RESULTS/exit-code" ] && echo "done(exit $(cat "$RESULTS/exit-code"))" || echo running)
  echo "$(date -u +%H:%M:%SZ) $state lines=$done_n red=$fail_n last: $last"
  ;;
*)
  echo "usage: run-suite.sh start [run-id] [-- playwright args] | wait <dir> [max-min] | status <dir>" >&2; exit 2 ;;
esac
