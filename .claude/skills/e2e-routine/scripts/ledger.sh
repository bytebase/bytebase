#!/usr/bin/env bash
# The run ledger lives on an orphan branch (default claude/e2e-ledger) of this
# repository so a run can read "when did we last run, on what SHA, with what
# result" without a second repository. Format: references/ledger.md.
#
#   ledger.sh last-sha                      # prints the SHA of the last recorded run, or nothing
#   ledger.sh last-result                   # green | red | infra | incomplete, or nothing
#   ledger.sh show                          # prints last-run.json
#   ledger.sh flakes                        # prints flakes.json
#   ledger.sh last-green                    # prints the attribution baseline SHA
#   ledger.sh record <results-dir> [note]   # appends the run and pushes the branch
set -u
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
cd "$REPO_ROOT" || exit 1

remote_ref="refs/remotes/origin/$E2E_LEDGER_BRANCH"
# A dry run (E2E_LEDGER_PUSH=0) commits here instead of pushing, and reads it
# back while the variable is 0 (references/ledger.md).
dry_ref="refs/e2e-routine/ledger-dry-run"
ident=(-c user.name="e2e-routine" -c user.email="e2e-routine@bytebase.com")
fetch_ledger() { git fetch -q origin "+refs/heads/$E2E_LEDGER_BRANCH:$remote_ref" 2>/dev/null; }
dry_run() { [ "${E2E_LEDGER_PUSH:-1}" != 1 ]; }
has_ref() { git rev-parse -q --verify "$1^{commit}" >/dev/null; }
read_ref() {
  fetch_ledger
  if dry_run && has_ref "$dry_ref"; then echo "$dry_ref"; else echo "$remote_ref"; fi
}

case "${1:-}" in
last-sha) git show "$(read_ref):last-run.json" 2>/dev/null | python3 -c 'import json,sys; print(json.load(sys.stdin).get("sha",""))' 2>/dev/null ;;
last-result) git show "$(read_ref):last-run.json" 2>/dev/null | python3 -c 'import json,sys; print(json.load(sys.stdin).get("result",""))' 2>/dev/null ;;
show)     git show "$(read_ref):last-run.json" 2>/dev/null || echo '{}' ;;
flakes)   git show "$(read_ref):flakes.json" 2>/dev/null || echo '{}' ;;
last-green)
  # The attribution baseline (references/ledger.md): the newest green run on
  # main, else the newest run on main, else main 60 days back. Computed on each
  # call so a phase never depends on a variable an earlier shell set.
  r="$(read_ref)"
  main="$(main_ref_strict)" || die "no origin/main to check the ledger against"
  # Every entry as "<result> <sha>", newest first; on_main judges each one.
  entries="$(python3 - "$r" <<'PY'
import json, subprocess, sys
ref = sys.argv[1]
def git(*a):
    return subprocess.run(["git", *a], capture_output=True, text=True).stdout
runs = []
for path in git("ls-tree", "--name-only", f"{ref}:runs").split():
    try:
        runs.append(json.loads(git("show", f"{ref}:runs/{path}")))
    except ValueError:
        pass
for e in sorted(runs, key=lambda e: (e.get("date", ""), e.get("run_id", "")), reverse=True):
    if e.get("sha"):
        print(e.get("result", "?"), e["sha"])
PY
)"
  green="" newest=""
  while read -r result s; do
    [ -n "$s" ] && on_main "$s" "$main" || continue
    [ -n "$newest" ] || newest="$s"
    if [ "$result" = green ]; then green="$s"; break; fi
  done <<<"$entries"
  sha="${green:-$newest}"
  [ -n "$sha" ] || sha="$(git rev-list -1 --before='60 days ago' "$main")"
  [ -n "$sha" ] || die "no baseline: no ledger entry on main and no main commit 60 days back"
  echo "$sha"
  ;;
record)
  RESULTS="${2:?results dir}"; note="${3:-}"
  [ -f "$RESULTS/summary.json" ] || die "$RESULTS/summary.json missing; run summarize.py first"
  sha="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("meta", {}).get("sha", ""))' "$RESULTS/summary.json")"
  # Only a main commit may enter the ledger (references/ledger.md); a dry run
  # from a branch records with a warning, since it never reaches the remote.
  if ! on_main "$sha"; then
    dry_run || die "run tested ${sha:-no SHA}, which is not a commit on main; not recorded"
    log "WARNING: ${sha:-no SHA} is not on main; recording on the dry-run ref only"
  fi
  tmp="$(mktemp -d)"
  if dry_run && has_ref "$dry_ref"; then
    git worktree add -q --detach "$tmp" "$dry_ref"
  elif fetch_ledger || has_ref "$remote_ref"; then
    # A failed fetch keeps the last fetched ledger rather than starting an
    # empty one; a push from it is then rejected, never a rewrite.
    git worktree add -q --detach "$tmp" "$remote_ref"
  else
    root="$(git "${ident[@]}" commit-tree "$(git hash-object -t tree /dev/null)" -m "e2e ledger root")"
    git worktree add -q --detach "$tmp" "$root"
  fi
  python3 - "$RESULTS" "$tmp" "$note" <<'PY'
import json, os, sys, datetime
results, dest, note = sys.argv[1:4]
s = json.load(open(os.path.join(results, "summary.json")))
meta, t = s.get("meta", {}), s.get("totals", {})
sha = meta.get("sha", "")
day = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%d")
run_url = ""
sid = os.environ.get("CLAUDE_CODE_REMOTE_SESSION_ID")
if sid:
    run_url = f"https://claude.ai/code/session_{sid}" if not sid.startswith("session_") else f"https://claude.ai/code/{sid}"
if s.get("infra"):
    result = "infra"
elif s.get("incomplete"):
    result = "incomplete"
elif t.get("failed"):
    result = "red"
else:
    result = "green"
entry = {"run_id": meta.get("run_id"), "sha": sha, "date": day, "result": result, "totals": t,
         "failed": [f"{e['file']}::{e['title']}" for e in s.get("failed", [])],
         "flaky": [f"{e['file']}::{e['title']}" for e in s.get("flaky", [])],
         "lock_unstable": [f"{e['file']}::{e['title']}" for e in s.get("lock_unstable", [])],
         "infra": s.get("infra"), "note": note, "run_url": run_url}
os.makedirs(os.path.join(dest, "runs"), exist_ok=True)
# The run id keeps a retry of the same SHA on the same day from overwriting
# the earlier entry.
json.dump(entry, open(os.path.join(dest, "runs", f"{day}-{sha[:7]}-{entry['run_id']}.json"), "w"), indent=2)
json.dump(entry, open(os.path.join(dest, "last-run.json"), "w"), indent=2)
fp = os.path.join(dest, "flakes.json")
flakes = json.load(open(fp)) if os.path.exists(fp) else {"runs": [], "tests": {}}
flakes["runs"] = (flakes["runs"] + [sha[:7]])[-20:]
for key in entry["flaky"]:
    hist = flakes["tests"].setdefault(key, [])
    hist.append(sha[:7])
    flakes["tests"][key] = hist[-20:]
json.dump(flakes, open(fp, "w"), indent=2)
if not os.path.exists(os.path.join(dest, "README.md")):
    open(os.path.join(dest, "README.md"), "w").write(
        "# e2e routine ledger\n\nMachine state for `.claude/skills/e2e-routine`: `last-run.json`, one `runs/<date>-<sha7>-<run_id>.json` per run, `flakes.json` (per-test flake history over the last 20 runs). Written by `ledger.sh record`; nothing here is hand-edited.\n")
print(f"recorded {result} run {entry['run_id']} @ {sha[:7]}")
PY
  (
    cd "$tmp" && git add -A && git "${ident[@]}" commit -q -m "e2e ledger: $(date -u +%F) ${sha:0:10}" \
      && if dry_run; then
           git update-ref "$dry_ref" HEAD
           log "E2E_LEDGER_PUSH=0: not pushing; recorded on $dry_ref, which ledger.sh reads while E2E_LEDGER_PUSH=0"
           git show --stat --format='%h %s' HEAD
         else git push -q origin "HEAD:refs/heads/$E2E_LEDGER_BRANCH"; fi
  ) || { status=$?; log "ledger NOT recorded (commit or push exit $status); the run dir still holds summary.json"; }
  git worktree remove --force "$tmp" 2>/dev/null || rm -rf "$tmp"
  git worktree prune
  exit "${status:-0}"
  ;;
*) echo "usage: ledger.sh last-sha | last-result | show | flakes | last-green | record <results-dir> [note]" >&2; exit 2 ;;
esac
