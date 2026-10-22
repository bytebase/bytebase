#!/usr/bin/env bash
# The run ledger lives on an orphan branch (default claude/e2e-ledger) of this
# repository so a run can read "when did we last run, on what SHA, with what
# result" without a second repository. Format: references/ledger.md.
#
#   ledger.sh last-sha                      # prints the SHA of the last recorded run, or nothing
#   ledger.sh last-result                   # green | red | infra | incomplete, or nothing
#   ledger.sh show                          # prints last-run.json
#   ledger.sh flakes                        # prints flakes.json
#   ledger.sh record <results-dir> [note]   # appends the run and pushes the branch
set -u
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
cd "$REPO_ROOT" || exit 1

ref="refs/remotes/origin/$E2E_LEDGER_BRANCH"
fetch_ledger() { git fetch -q origin "+refs/heads/$E2E_LEDGER_BRANCH:$ref" 2>/dev/null; }

case "${1:-}" in
last-sha) fetch_ledger; git show "$ref:last-run.json" 2>/dev/null | python3 -c 'import json,sys; print(json.load(sys.stdin).get("sha",""))' 2>/dev/null ;;
last-result) fetch_ledger; git show "$ref:last-run.json" 2>/dev/null | python3 -c 'import json,sys; print(json.load(sys.stdin).get("result",""))' 2>/dev/null ;;
show)     fetch_ledger; git show "$ref:last-run.json" 2>/dev/null || echo '{}' ;;
flakes)   fetch_ledger; git show "$ref:flakes.json" 2>/dev/null || echo '{}' ;;
record)
  RESULTS="${2:?results dir}"; note="${3:-}"
  [ -f "$RESULTS/summary.json" ] || die "$RESULTS/summary.json missing; run summarize.py first"
  tmp="$(mktemp -d)"
  if fetch_ledger; then
    git worktree add -q --detach "$tmp" "$ref"
  else
    root="$(git commit-tree "$(git hash-object -t tree /dev/null)" -m "e2e ledger root")"
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
         "infra": s.get("infra"), "note": note, "run_url": run_url}
os.makedirs(os.path.join(dest, "runs"), exist_ok=True)
json.dump(entry, open(os.path.join(dest, "runs", f"{day}-{sha[:7]}.json"), "w"), indent=2)
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
        "# e2e routine ledger\n\nMachine state for `.claude/skills/e2e-routine`: `last-run.json`, one `runs/<date>-<sha7>.json` per run, `flakes.json` (per-test flake history over the last 20 runs). Written by `ledger.sh record`; nothing here is hand-edited.\n")
print(f"recorded {result} run {entry['run_id']} @ {sha[:7]}")
PY
  (
    cd "$tmp" && git add -A && git -c user.name="e2e-routine" -c user.email="e2e-routine@bytebase.com" commit -q -m "e2e ledger: $(date -u +%F) $(git -C "$REPO_ROOT" rev-parse --short HEAD)" \
      && if [ "${E2E_LEDGER_PUSH:-1}" = 1 ]; then git push -q origin "HEAD:refs/heads/$E2E_LEDGER_BRANCH"; else log "E2E_LEDGER_PUSH=0: not pushing; inspect with git show $(git rev-parse --short HEAD)"; git show --stat --format='%h %s' HEAD; fi
  ) || log "ledger push failed; the run dir still holds summary.json"
  git worktree remove --force "$tmp" 2>/dev/null || rm -rf "$tmp"
  git worktree prune
  ;;
*) echo "usage: ledger.sh last-sha | last-result | show | flakes | record <results-dir> [note]" >&2; exit 2 ;;
esac
