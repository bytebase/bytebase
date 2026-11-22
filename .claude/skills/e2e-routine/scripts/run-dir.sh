#!/usr/bin/env bash
# The current run's results directory, kept in a file because each phase runs
# in a fresh shell: a variable set in Phase 0 is gone by Phase 2.
#
#   run-dir.sh new   create a run directory, make it current, print its path
#   run-dir.sh       print the current run directory
#   run-dir.sh id    print the current run id (the directory's name)
set -u
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

pointer="$E2E_RESULTS_ROOT/CURRENT"
case "${1:-}" in
new)
  mkdir -p "$E2E_RESULTS_ROOT"
  dir="$E2E_RESULTS_ROOT/$(run_id_now)"
  mkdir -p "$dir"
  printf '%s\n' "$dir" > "$pointer"
  echo "$dir"
  ;;
""|id)
  [ -s "$pointer" ] || die "no current run; start one with run-dir.sh new"
  dir="$(cat "$pointer")"
  [ -d "$dir" ] || die "current run directory $dir is gone"
  if [ "${1:-}" = id ]; then basename "$dir"; else echo "$dir"; fi
  ;;
*) echo "usage: run-dir.sh new | run-dir.sh | run-dir.sh id" >&2; exit 2 ;;
esac
