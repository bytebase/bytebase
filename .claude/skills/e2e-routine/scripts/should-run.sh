#!/usr/bin/env bash
# Decide whether tonight's run is worth doing.
#
#   should-run.sh            -> prints a JSON decision; exit codes:
#     0  RUN          main moved and the diff touches product or e2e code, or the
#                     last run of this SHA was infra or incomplete (not covered)
#     3  NO_CHANGE    origin/main is the SHA the ledger already recorded
#     4  NOT_RELEVANT main moved but only docs/CI/workflow files changed
#   FORCE=1 should-run.sh    -> always RUN (manual fire)
#
# The ledger is read from origin/<E2E_LEDGER_BRANCH>:last-run.json (see
# references/ledger.md). A missing ledger means "never ran": RUN.
set -u
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
cd "$REPO_ROOT" || exit 1

head="$(main_sha)"
last="$("$E2E_SKILL_DIR/scripts/ledger.sh" last-sha 2>/dev/null || true)"

emit() {
  local decision="$1" reason="$2" count="${3:-0}" relevant="${4:-0}"
  printf '{"decision":"%s","reason":"%s","head_sha":"%s","last_sha":"%s","commits":%s,"relevant_paths":%s}\n' \
    "$decision" "$reason" "$head" "${last:-}" "$count" "$relevant"
}

if [ "${FORCE:-0}" = 1 ]; then emit RUN forced; exit 0; fi
if [ -z "$last" ]; then emit RUN "no ledger entry yet"; exit 0; fi
# A SHA counts as covered only when its last run actually tested it: an
# infra failure or a run cut short proves nothing, so the same SHA runs again.
last_result="$("$E2E_SKILL_DIR/scripts/ledger.sh" last-result 2>/dev/null || true)"
case "$last_result" in
  infra|incomplete) emit RUN "last run on $last was $last_result; retrying"; exit 0 ;;
esac
if [ "$last" = "$head" ]; then emit NO_CHANGE "origin/main unchanged since the last run"; exit 3; fi
if ! git cat-file -e "$last^{commit}" 2>/dev/null; then
  git fetch -q origin "$last" 2>/dev/null || { emit RUN "last SHA $last not fetchable; running"; exit 0; }
fi

count=$(git rev-list --count "$last..$head" 2>/dev/null || echo 0)
# Everything that is built into the binary or drives the suite counts: the
# whole frontend, backend and proto trees and the Go module files. Excluded are
# only files that cannot reach the build: contributor docs, Go tests and
# frontend unit tests. Other Markdown is product input (guide pages loaded by
# import.meta.glob, prompts and skills pulled in by go:embed), so it stays.
# The list names what to drop, not what to keep.
relevant=$(git diff --name-only "$last" "$head" -- 'frontend' 'backend' 'proto' 'go.mod' 'go.sum' \
  | grep -vE '(^|/)(README|AGENTS|CLAUDE|CHANGELOG|CONTRIBUTING)\.md$' \
  | grep -vE '^backend/.*_test\.go$' \
  | grep -vE '^frontend/(src|scripts)/.*\.test\.(ts|tsx|mjs)$' \
  | wc -l | tr -d ' ')

if [ "$relevant" = 0 ]; then emit NOT_RELEVANT "$count commit(s) since $last touch no product or e2e code" "$count" 0; exit 4; fi
emit RUN "$count commit(s) since $last, $relevant relevant path(s)" "$count" "$relevant"
exit 0
