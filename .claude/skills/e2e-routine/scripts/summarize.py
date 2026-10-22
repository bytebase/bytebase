#!/usr/bin/env python3
"""Turn a Playwright JSON report into the routine's classification buckets.

    summarize.py <results-dir>   -> writes summary.json and summary.md there, prints summary.md

Buckets (see references/triage.md for what to do with each):
  failed        unexpected on every attempt           -> verify: product regression or spec rot
  flaky         failed, then passed on a retry        -> flakes ledger
  lock_flipped  test.fail() lock that now passes      -> the bug was fixed; remove the lock
  lock_holding  test.fail() lock still failing        -> not red
  infra         no test ran, or every test died on a boot/license/browser signature
                (a lost server whose server.log holds a Go crash is a failed row instead)
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


# Infra kinds that only say the client lost the server; server.log says why.
SERVER_LOSS_KINDS = ("server-lost", "boot", "connection")

# What the Go runtime prints when the process dies: the message at the start
# of a line, then a goroutine dump. Errors the server logs and survives carry
# stack text inside a log record, never at the start of a line.
CRASH_RE = re.compile(r"^(?:panic|fatal error): .*$", re.M)
GOROUTINE_RE = re.compile(r"^goroutine \d+ \[", re.M)
FRAME_RE = re.compile(r"^\t(\S+?\.go:\d+)", re.M)


def server_crash(results_dir):
    """The first unrecovered crash in server.log: its message line and the
    first backend frame of its stack (the file attribution starts from)."""
    sp = os.path.join(results_dir, "server.log")
    if not os.path.exists(sp):
        return None
    text = open(sp, errors="replace").read()
    for m in CRASH_RE.finditer(text):
        dump = GOROUTINE_RE.search(text, m.end())
        if not dump:
            continue
        # Only the crashing goroutine's frames: its block ends at the next dump.
        nxt = GOROUTINE_RE.search(text, dump.end())
        block = text[dump.end():nxt.start() if nxt else len(text)]
        frame = next(("backend/" + p.split("/backend/", 1)[1] for p in FRAME_RE.findall(block)
                      if "/pkg/mod/" not in p and "/backend/" in p), None)
        return {"line": m.group(0)[:300], "frame": frame}
    return None


def first_line(msg):
    msg = re.sub(r"\x1b\[[0-9;]*m", "", msg or "")
    for line in msg.splitlines():
        line = line.strip()
        if line:
            return line[:300]
    return ""


def pick_evidence(results):
    """The files that show the failure the entry reports: the screenshots (one
    per open page) and aria snapshot of the latest failed attempt that has
    them, and the trace of a failed attempt (Playwright traces the first retry
    only). A passing retry's trace shows the pass, so it is never the evidence."""
    failed = [(i, r) for i, r in enumerate(results) if r.get("status") not in ("passed", "skipped")]

    def latest(name):
        for i, r in reversed(failed):
            paths = sorted({a["path"] for a in r.get("attachments", [])
                            if a.get("name") == name and a.get("path") and os.path.exists(a["path"])})
            if paths:
                return i, paths
        return None, []

    shot_attempt, screenshots = latest("screenshot")
    _, error_context = latest("error-context")
    _, trace = latest("trace")
    return {"attempt": shot_attempt, "screenshots": screenshots,
            "error_context": error_context[0] if error_context else None,
            "trace": trace[0] if trace else None}


NO_EVIDENCE = {"attempt": None, "screenshots": [], "error_context": None, "trace": None}


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
        # The latest attempt that carries an error: for a flaky test the final
        # attempt passed and has none, and the failure is the evidence.
        errored = [r for r in results if (r.get("error") or {}).get("message")]
        err = first_line(errored[-1]["error"]["message"]) if errored else ""
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
            "evidence": pick_evidence(results),
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
    # connection or boot signature; a missing browser, psql or toolchain fails
    # them with that signature instead. Judge it by the share of failures that
    # carry any environment signature, not by their share of the suite: a
    # server lost late leaves most tests passed. A failure of the setup project
    # alone is enough, since every other test depends on it. Failures without a
    # signature stay in `failed` and are still triaged one by one.
    signed = [e for e in failed if e["infra_signature"]]
    setup_lost = bool(signed) and len(signed) == len(failed) and any(
        e["file"].endswith("framework/setup-project.ts") for e in signed)
    setup_signature = infra_signature(" ".join(summary["errors"])) if summary["errors"] else None
    if summary["totals"]["tests"] == 0 and summary["errors"] and setup_signature:
        summary["infra"] = {"kind": setup_signature, "detail": summary["errors"][0]}
    elif summary["totals"]["tests"] == 0 and summary["errors"]:
        # Global setup failed with no known environment signature: one of its
        # product calls (signup, project creation, sample instance) broke. It is
        # a failure to triage and attribute like any other, not infra.
        summary["failed"] = [{"file": "framework/global-setup.ts", "title": "global setup", "line": 0,
                              "error": summary["errors"][0], "infra_signature": None, "attempts": 1,
                              "statuses": ["failed"], "evidence": NO_EVIDENCE}]
        failed = summary["failed"]
        summary["totals"]["failed"] = 1
    elif setup_lost or (len(signed) >= SERVER_LOST_MIN_FAILURES and len(signed) >= SERVER_LOST_SHARE * len(failed)):
        dominant = Counter(e["infra_signature"] for e in signed).most_common(1)[0][0]
        kind = "server-lost" if dominant in ("connection", "boot") else dominant
        summary["infra"] = {"kind": kind,
                            "detail": f"{len(signed)} of {len(failed)} failures carry an environment signature ({dominant}); first: {signed[0]['error']}",
                            "first_test": f"{signed[0]['file']}:{signed[0]['line']}"}
        # These casualties prove nothing about their tests, so they leave the
        # triage queue; failures without a signature stay in it.
        summary["lost_to_infra"] = signed
        summary["failed"] = [e for e in failed if e not in signed]
        failed = summary["failed"]
        summary["totals"]["failed"] = len(failed)
        summary["totals"]["lost_to_infra"] = len(signed)
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
                                          "statuses": ["failed"], "evidence": NO_EVIDENCE})
            failed = summary["failed"]
            summary["totals"]["failed"] = len(failed)

    # Losing the server is only an environment failure when the server did not
    # crash on its own: a Go panic in server.log is a product failure. The
    # casualties still prove nothing about their own tests, so they stay out of
    # triage; one row for the crash takes their place.
    crash = server_crash(results_dir)
    summary["server_crash"] = crash
    lost_server = summary["infra"]
    if crash and lost_server and lost_server["kind"] in SERVER_LOSS_KINDS:
        summary["infra"] = None
        casualties = summary.pop("lost_to_infra", [])
        summary["totals"].pop("lost_to_infra", None)
        if casualties:
            summary["lost_to_crash"] = casualties
            summary["totals"]["lost_to_crash"] = len(casualties)
        summary["failed"].insert(0, {
            "file": "(server)", "title": "Bytebase server crashed", "line": 0, "error": crash["line"],
            "frame": crash["frame"], "first_test": lost_server.get("first_test"), "infra_signature": None,
            "attempts": 1, "statuses": ["failed"],
            "evidence": casualties[0]["evidence"] if casualties else NO_EVIDENCE})
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
            "not_run": summary["totals"]["not_run"],
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


def evidence_line(ev, results_dir):
    if not ev["screenshots"]:
        return "none (read error-context.md and playwright.log)" if ev["error_context"] else "none"

    def show(path):
        rel = os.path.relpath(path, results_dir)
        return path if rel.startswith("..") else rel
    return ", ".join(f"`{show(p)}`" for p in ev["screenshots"]) + f" (attempt {ev['attempt'] + 1})"


def write(results_dir, s):
    with open(os.path.join(results_dir, "summary.json"), "w") as f:
        json.dump(s, f, indent=2)
    t = s["totals"]
    m = s.get("meta", {})
    lines = [f"# e2e run {m.get('run_id', '?')} @ {m.get('short_sha', '?')}", ""]
    if s["infra"]:
        lines += [f"**INFRA**: {s['infra']['kind']}: {s['infra']['detail']}", ""]
    if s.get("server_crash"):
        c = s["server_crash"]
        lines += [f"**SERVER CRASH** in server.log: `{c['line']}` at `{c['frame'] or 'no backend frame'}`", ""]
    if s.get("incomplete"):
        lines += [f"**INCOMPLETE**: {s['incomplete']['reason']}; {s['incomplete']['not_run']} test(s) never ran. Not a green run.", ""]
    if t:
        lines += [f"{t['passed']} passed, {t['failed']} failed, {t['flaky']} flaky, {t['lock_flipped']} lock flipped, "
                  f"{t['lock_holding']} lock holding, {t.get('not_run', 0)} did not run, "
                  f"{t.get('skipped_deliberate', t['skipped'])} skipped on purpose, {t['tests']} total, {t['duration_s']}s", ""]
    for bucket, head in (("failed", "Failed (every attempt)"), ("flaky", "Flaky (passed on retry)"),
                         ("lost_to_infra", "Lost to the environment (not triaged; they rerun once it is fixed)"),
                         ("lost_to_crash", "Lost to the server crash (not triaged; they rerun once the crash is fixed)"),
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
                if bucket in ("failed", "flaky", "lock_flipped"):
                    lines.append(f"  - screenshots: {evidence_line(e.get('evidence') or NO_EVIDENCE, results_dir)}")
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
                              "error": detail, "infra_signature": None, "attempts": 1,
                              "evidence": NO_EVIDENCE}]
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
