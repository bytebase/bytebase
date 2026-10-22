#!/usr/bin/env python3
"""Turn a Playwright JSON report into the routine's classification buckets.

    summarize.py <results-dir>   -> writes summary.json and summary.md there, prints summary.md

Buckets (see references/triage.md for what to do with each):
  failed        unexpected on every attempt           -> verify: product regression or spec rot
  flaky         failed, then passed on a retry        -> flakes ledger
  lock_flipped  test.fail() lock that now passes      -> the bug was fixed; remove the lock
  lock_holding  test.fail() lock still failing        -> not red
  infra         no test ran, or every test died on a boot/license/browser signature
  incomplete    the run was cut short (global timeout, crash): tests never ran, never green

Exit code: 0 green, 1 red or infra, 2 no report, 3 incomplete.

    summarize.py --pre-run-failure <results-dir> <kind> <detail> [infra|product]
        records a failure before the suite ran (preflight, build) so ledger.sh
        can record it; kind is e.g. preflight or build. infra (default) is
        retried next run; product (a product file does not compile) is red.
"""
import json
import os
import re
import subprocess
import sys
from collections import Counter

SERVER_LOST_MIN_FAILURES = 5
SERVER_LOST_SHARE = 0.8

INFRA_SIGNATURES = [
    ("boot", r"did not become healthy|Could not find available port|bind: address already in use"),
    # The messages that mean the license itself is missing or unusable. Not
    # the bare word: every error from the license upload API carries it in
    # the URL, and a regression in that API is a product failure.
    ("license", r"No enterprise license: set BYTEBASE_E2E_LICENSE|token signature is invalid|license has expired"),
    ("binary", r"Bytebase binary not found|does not bundle frontend"),
    ("psql", r"spawn psql ENOENT"),
    ("browser", r"Executable doesn't exist|browserType\.launch|playwright install"),
    ("toolchain", r"ERR_PNPM_|Command failed: pnpm|ENOSPC|EAI_AGAIN|Could not reach https?://registry"),
    ("connection", r"fetch failed|ECONNREFUSED|ERR_CONNECTION_REFUSED"),
]


def first_line(msg):
    msg = re.sub(r"\x1b\[[0-9;]*m", "", msg or "")
    for line in msg.splitlines():
        line = line.strip()
        if line:
            return line[:300]
    return ""


def walk(suite, file_title, out):
    file_title = suite.get("file") or file_title
    for spec in suite.get("specs", []):
        for test in spec.get("tests", []):
            out.append((file_title, spec, test))
    for child in suite.get("suites", []):
        walk(child, file_title, out)


def classify(test):
    status = test.get("status")  # expected | unexpected | flaky | skipped
    expected = test.get("expectedStatus")  # passed | failed | skipped
    if status == "skipped":
        return "skipped"
    if status == "flaky":
        return "flaky"
    if expected == "failed":
        return "lock_holding" if status == "expected" else "lock_flipped"
    return "passed" if status == "expected" else "failed"


def infra_signature(text):
    for name, pat in INFRA_SIGNATURES:
        if re.search(pat, text or "", re.I):
            return name
    return None


def main(results_dir):
    report_path = os.path.join(results_dir, "report.json")
    meta = {}
    mp = os.path.join(results_dir, "meta.json")
    if os.path.exists(mp):
        meta = json.load(open(mp))
    exit_code = None
    ep = os.path.join(results_dir, "exit-code")
    if os.path.exists(ep):
        exit_code = int(open(ep).read().strip() or 0)

    summary = {"meta": meta, "exit_code": exit_code, "totals": {}, "failed": [], "flaky": [],
               "lock_flipped": [], "lock_holding": [], "skipped": [], "not_run": [], "infra": None,
               "incomplete": None, "errors": []}

    if not os.path.exists(report_path):
        log_tail = ""
        lp = os.path.join(results_dir, "playwright.log")
        if os.path.exists(lp):
            log_tail = open(lp, errors="replace").read()[-4000:]
        summary["infra"] = {"kind": infra_signature(log_tail) or "no-report",
                            "detail": first_line(log_tail.splitlines()[-1] if log_tail else "no report.json and no log")}
        write(results_dir, summary)
        return 2

    report = json.load(open(report_path))
    summary["errors"] = [first_line(e.get("message", "")) for e in report.get("errors", [])]
    rows = []
    for suite in report.get("suites", []):
        walk(suite, suite.get("file"), rows)

    counts = Counter()
    # A test skipped without a skip/fixme annotation never ran: its serial
    # group stopped at an earlier failure (or the run was cut short). It is not
    # covered, so it is counted and listed apart from deliberate skips, with
    # the failure that blocked it. Re-running the same SHA would block it again;
    # it runs once that failure is fixed.
    last_failed_in_file = {}
    for file_title, spec, test in rows:
        bucket = classify(test)
        counts[bucket] += 1
        if bucket == "failed":
            last_failed_in_file[file_title] = f"{file_title}:{spec.get('line')} {spec.get('title')}"
        deliberate = any(a.get("type") in ("skip", "fixme") for a in test.get("annotations", []))
        if bucket == "skipped":
            counts["not_run" if not deliberate else "skipped_deliberate"] += 1
        results = test.get("results", [])
        last = results[-1] if results else {}
        err = first_line((last.get("error") or {}).get("message", "")) if last else ""
        attachments = [a.get("path") for r in results for a in r.get("attachments", []) if a.get("path")]
        entry = {
            "file": file_title,
            "title": spec.get("title"),
            "line": spec.get("line"),
            "project": test.get("projectName"),
            "attempts": len(results),
            "statuses": [r.get("status") for r in results],
            "duration_ms": sum(r.get("durationMs", r.get("duration", 0)) or 0 for r in results),
            "error": err,
            "infra_signature": infra_signature(err),
            "attachments": attachments,
            "annotations": [a.get("type") for a in test.get("annotations", [])],
        }
        if bucket == "skipped" and not deliberate:
            entry["blocked_by"] = last_failed_in_file.get(file_title)
            summary["not_run"].append(entry)
        elif bucket in ("failed", "flaky", "lock_flipped", "lock_holding", "skipped"):
            summary[bucket].append(entry)

    stats = report.get("stats", {})
    summary["totals"] = {
        # not_run and skipped_deliberate split `skipped`; they are not extra tests.
        "tests": sum(n for k, n in counts.items() if k not in ("not_run", "skipped_deliberate")),
        "passed": counts["passed"],
        "failed": counts["failed"],
        "flaky": counts["flaky"],
        "lock_flipped": counts["lock_flipped"],
        "lock_holding": counts["lock_holding"],
        "skipped": counts["skipped"],
        "not_run": counts["not_run"],
        "skipped_deliberate": counts["skipped_deliberate"],
        "duration_s": round((stats.get("duration") or 0) / 1000),
    }

    failed = summary["failed"]
    # The disposable server dying mid-run fails every test after it with a
    # connection or boot signature. Judge that by the share of failures that
    # carry the signature, not by their share of the suite: a server lost late
    # leaves most tests passed. Failures without the signature stay in
    # `failed` and are still triaged one by one.
    signed = [e for e in failed if e["infra_signature"] in ("connection", "boot")]
    setup_signature = infra_signature(" ".join(summary["errors"])) if summary["errors"] else None
    if summary["totals"]["tests"] == 0 and summary["errors"] and setup_signature:
        summary["infra"] = {"kind": setup_signature, "detail": summary["errors"][0]}
    elif summary["totals"]["tests"] == 0 and summary["errors"]:
        # Global setup failed with no known environment signature: one of its
        # product calls (signup, project creation, sample instance) broke. It is
        # a failure to triage and attribute like any other, not infra.
        summary["failed"] = [{"file": "framework/global-setup.ts", "title": "global setup", "line": 0,
                              "error": summary["errors"][0], "infra_signature": None, "attempts": 1,
                              "statuses": ["failed"], "attachments": []}]
        failed = summary["failed"]
        summary["totals"]["failed"] = 1
    elif len(signed) >= SERVER_LOST_MIN_FAILURES and len(signed) >= SERVER_LOST_SHARE * len(failed):
        summary["infra"] = {"kind": "server-lost",
                            "detail": f"{len(signed)} of {len(failed)} failures lost the server; first: {signed[0]['error']}",
                            "first_test": f"{signed[0]['file']}:{signed[0]['line']}"}
        # The server's casualties prove nothing about their tests, so they
        # leave the triage queue; failures without the signature stay in it.
        summary["lost_to_server"] = signed
        summary["failed"] = [e for e in failed if e not in signed]
        failed = summary["failed"]
        summary["totals"]["failed"] = len(failed)
        summary["totals"]["lost_to_server"] = len(signed)
    elif summary["errors"]:
        # Errors outside any test after tests ran (global teardown, a worker
        # crash): Playwright fails the run for them. One with an environment
        # signature is infra; otherwise each becomes a failed row to triage.
        if setup_signature:
            summary["infra"] = {"kind": setup_signature, "detail": summary["errors"][0]}
        else:
            for err in summary["errors"]:
                summary["failed"].append({"file": "(outside any test)", "title": "suite-level error", "line": 0,
                                          "error": err, "infra_signature": None, "attempts": 1,
                                          "statuses": ["failed"], "attachments": []})
            failed = summary["failed"]
            summary["totals"]["failed"] = len(failed)

    # A run the global timeout (or a crash) cut short proves nothing about the
    # tests it did not reach, so it is "incomplete", never green. The evidence
    # is the timeout line, or a nonzero exit with no failure to explain it and
    # tests that never ran. A flipped lock also exits nonzero with no failure,
    # but every test ran, so it is not incomplete.
    log_text = ""
    lp = os.path.join(results_dir, "playwright.log")
    if os.path.exists(lp):
        log_text = open(lp, errors="replace").read()
    # Playwright's runner reports the global timeout as
    # "Timed out waiting <N>s for the <task> to run"; the task name varies by
    # phase ("entire test run", "test suite", ...).
    timed_out = re.search(r"Timed out waiting [\d.]+s for the ", log_text) is not None
    cut_short = (exit_code not in (0, None) and not failed and not summary["lock_flipped"]
                 and summary["totals"]["not_run"] > 0)
    if not summary["infra"] and (timed_out or cut_short):
        summary["incomplete"] = {
            "reason": "global timeout" if timed_out else f"exit code {exit_code} with no failed test",
            "not_run": summary["totals"]["skipped"],
        }
    else:
        summary["incomplete"] = None

    # Green requires Playwright's own verdict. A nonzero exit that nothing
    # above explains (no failure, flipped lock, infra or incomplete) is never
    # recorded green: it is infra, with the exit code as the evidence.
    explained = failed or summary["lock_flipped"] or summary["infra"] or summary["incomplete"]
    if exit_code not in (0, None) and not explained:
        summary["infra"] = {"kind": "unexplained-exit",
                            "detail": f"playwright exited {exit_code} with every test passing and no reported error"}

    write(results_dir, summary)
    if summary["infra"]:
        return 1
    if summary["incomplete"]:
        return 3
    return 0 if not failed else 1


def write(results_dir, s):
    with open(os.path.join(results_dir, "summary.json"), "w") as f:
        json.dump(s, f, indent=2)
    t = s["totals"]
    m = s.get("meta", {})
    lines = [f"# e2e run {m.get('run_id', '?')} @ {m.get('short_sha', '?')}", ""]
    if s["infra"]:
        lines += [f"**INFRA**: {s['infra']['kind']}: {s['infra']['detail']}", ""]
    if s.get("incomplete"):
        lines += [f"**INCOMPLETE**: {s['incomplete']['reason']}; {s['incomplete']['not_run']} test(s) never ran. Not a green run.", ""]
    if t:
        lines += [f"{t['passed']} passed, {t['failed']} failed, {t['flaky']} flaky, {t['lock_flipped']} lock flipped, "
                  f"{t['lock_holding']} lock holding, {t.get('not_run', 0)} did not run, "
                  f"{t.get('skipped_deliberate', t['skipped'])} skipped on purpose, {t['tests']} total, {t['duration_s']}s", ""]
    for bucket, head in (("failed", "Failed (every attempt)"), ("flaky", "Flaky (passed on retry)"),
                         ("lost_to_server", "Lost to the server (not triaged; rerun once the server is fixed)"),
                         ("lock_flipped", "Locks that now pass"),
                         ("not_run", "Did not run (not covered by this run)"), ("skipped", "Skipped on purpose")):
        if s.get(bucket):
            lines += [f"## {head}", ""]
            for e in s[bucket]:
                sig = f" [{e['infra_signature']}]" if e.get("infra_signature") else ""
                lines.append(f"- `{e['file']}:{e['line']}` {e['title']}{sig}")
                if e.get("error"):
                    lines.append(f"  - {e['error']}")
                if bucket == "not_run":
                    lines.append(f"  - blocked by: {e.get('blocked_by') or 'the run ending early'}")
            lines.append("")
    if s.get("errors"):
        lines += ["## Suite-level errors", ""] + [f"- {e}" for e in s["errors"]] + [""]
    md = "\n".join(lines)
    with open(os.path.join(results_dir, "summary.md"), "w") as f:
        f.write(md)
    print(md)


def record_pre_run_failure(results_dir, kind, detail, cause="infra"):
    """A failure before the suite ran (preflight, build) has no report. Write
    the same meta.json and summary.json a run would, so ledger.sh can record
    it. cause=infra (the default) sets `infra`, and the next run retries the
    SHA. cause=product is a product file that does not compile: it is one
    failed entry, the run is red, and it goes through Phase 5 like any product
    regression."""
    os.makedirs(results_dir, exist_ok=True)
    mp = os.path.join(results_dir, "meta.json")
    if os.path.exists(mp):
        meta = json.load(open(mp))
    else:
        repo = os.path.abspath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "..", ".."))
        sha = subprocess.run(["git", "-C", repo, "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip()
        meta = {"run_id": os.path.basename(os.path.normpath(results_dir)), "sha": sha, "short_sha": sha[:10]}
        json.dump(meta, open(mp, "w"))
    summary = {"meta": meta, "exit_code": None, "totals": {}, "failed": [], "flaky": [],
               "lock_flipped": [], "lock_holding": [], "skipped": [], "not_run": [],
               "infra": None, "incomplete": None, "errors": []}
    if cause == "product":
        summary["failed"] = [{"file": f"({kind})", "title": f"{kind} of a product file failed", "line": 0,
                              "error": detail, "infra_signature": None, "attempts": 1}]
        summary["totals"] = {"tests": 0, "passed": 0, "failed": 1, "flaky": 0, "lock_flipped": 0,
                             "lock_holding": 0, "skipped": 0, "not_run": 0, "skipped_deliberate": 0,
                             "duration_s": 0}
    else:
        summary["infra"] = {"kind": kind, "detail": detail}
    write(results_dir, summary)
    return 1


if __name__ == "__main__":
    if sys.argv[1:2] == ["--pre-run-failure"] and len(sys.argv) in (5, 6):
        cause = sys.argv[5] if len(sys.argv) == 6 else "infra"
        if cause not in ("infra", "product"):
            print("cause must be infra or product", file=sys.stderr)
            sys.exit(2)
        sys.exit(record_pre_run_failure(sys.argv[2], sys.argv[3], sys.argv[4], cause))
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(2)
    sys.exit(main(sys.argv[1]))
