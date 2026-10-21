#!/usr/bin/env python3
"""Turn a Playwright JSON report into the routine's classification buckets.

    summarize.py <results-dir>   -> writes summary.json and summary.md there, prints summary.md

Buckets (see references/triage.md for what to do with each):
  failed        unexpected on every attempt           -> verify: product regression or spec rot
  flaky         failed, then passed on a retry        -> flakes ledger
  lock_flipped  test.fail() lock that now passes      -> the bug was fixed; remove the lock
  lock_holding  test.fail() lock still failing        -> not red
  infra         no test ran, or every test died on a boot/license/browser signature
"""
import json
import os
import re
import sys
from collections import Counter

INFRA_SIGNATURES = [
    ("boot", r"did not become healthy|Could not find available port|bind: address already in use"),
    ("license", r"BYTEBASE_E2E_LICENSE|license"),
    ("binary", r"Bytebase binary not found|does not bundle frontend"),
    ("psql", r"spawn psql ENOENT"),
    ("browser", r"Executable doesn't exist|browserType\.launch"),
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
               "lock_flipped": [], "lock_holding": [], "skipped": [], "infra": None, "errors": []}

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
    for file_title, spec, test in rows:
        bucket = classify(test)
        counts[bucket] += 1
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
        if bucket in ("failed", "flaky", "lock_flipped", "lock_holding", "skipped"):
            summary[bucket].append(entry)

    stats = report.get("stats", {})
    summary["totals"] = {
        "tests": sum(counts.values()),
        "passed": counts["passed"],
        "failed": counts["failed"],
        "flaky": counts["flaky"],
        "lock_flipped": counts["lock_flipped"],
        "lock_holding": counts["lock_holding"],
        "skipped": counts["skipped"],
        "duration_s": round((stats.get("duration") or 0) / 1000),
    }

    failed = summary["failed"]
    if summary["totals"]["tests"] == 0 and summary["errors"]:
        summary["infra"] = {"kind": infra_signature(" ".join(summary["errors"])) or "setup",
                            "detail": summary["errors"][0]}
    elif failed and all(e["infra_signature"] in ("connection", "boot") for e in failed) and len(failed) >= max(5, summary["totals"]["tests"] // 2):
        summary["infra"] = {"kind": "server-lost", "detail": failed[0]["error"]}

    write(results_dir, summary)
    return 0 if not failed and not summary["infra"] else 1


def write(results_dir, s):
    with open(os.path.join(results_dir, "summary.json"), "w") as f:
        json.dump(s, f, indent=2)
    t = s["totals"]
    m = s.get("meta", {})
    lines = [f"# e2e run {m.get('run_id', '?')} @ {m.get('short_sha', '?')}", ""]
    if s["infra"]:
        lines += [f"**INFRA**: {s['infra']['kind']}: {s['infra']['detail']}", ""]
    if t:
        lines += [f"{t['passed']} passed, {t['failed']} failed, {t['flaky']} flaky, {t['lock_flipped']} lock flipped, "
                  f"{t['lock_holding']} lock holding, {t['skipped']} skipped, {t['tests']} total, {t['duration_s']}s", ""]
    for bucket, head in (("failed", "Failed (every attempt)"), ("flaky", "Flaky (passed on retry)"),
                         ("lock_flipped", "Locks that now pass"), ("skipped", "Skipped")):
        if s.get(bucket):
            lines += [f"## {head}", ""]
            for e in s[bucket]:
                sig = f" [{e['infra_signature']}]" if e.get("infra_signature") else ""
                lines.append(f"- `{e['file']}:{e['line']}` {e['title']}{sig}")
                if e.get("error"):
                    lines.append(f"  - {e['error']}")
            lines.append("")
    if s.get("errors"):
        lines += ["## Suite-level errors", ""] + [f"- {e}" for e in s["errors"]] + [""]
    md = "\n".join(lines)
    with open(os.path.join(results_dir, "summary.md"), "w") as f:
        f.write(md)
    print(md)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(2)
    sys.exit(main(sys.argv[1]))
