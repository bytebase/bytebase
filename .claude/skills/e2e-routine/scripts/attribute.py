#!/usr/bin/env python3
"""Rank the product commits that could explain a red spec, and name an owner.

    attribute.py --spec frontend/tests/e2e/sql-editor/sql-editor-result.spec.ts \
                 --since <last-green-sha> [--text "Request export" ...] [--json]

Candidates are the non-merge commits in <since>..origin/main that touch the
area's source paths (owners.json), the component files that render any quoted
UI text (found through the locale files), or whose diff adds/removes the text
itself (git log -S). Score = 3 per text hit + 1 per path hit. The owner is the
top candidate's author mapped through owners.json to a Linear user name, else
the area default.
"""
import argparse
import glob
import json
import os
import re
import subprocess
import sys

SKILL_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
REPO = os.path.abspath(os.path.join(SKILL_DIR, "..", "..", ".."))
# Text searches (git log -S) look only at product code and copy. A docs, test
# or routine commit that merely mentions a UI word must not become the culprit.
PRODUCT_PATHS = ["frontend/src", "backend", "proto"]


def git(*args):
    return subprocess.run(["git", "-C", REPO, *args], capture_output=True, text=True).stdout


def commits(rng, paths=(), pickaxe=None):
    cmd = ["log", "--no-merges", "--format=%H%x1f%an%x1f%ad%x1f%s", "--date=short", rng]
    if pickaxe:
        cmd += ["-S", pickaxe, "--pickaxe-regex"]
    if paths:
        cmd += ["--", *paths]
    out = {}
    for line in git(*cmd).splitlines():
        sha, author, date, subject = line.split("\x1f")
        out[sha] = {"sha": sha[:10], "author": author, "date": date, "subject": subject,
                    "pr": (re.search(r"\(#(\d+)\)\s*$", subject) or [None, None])[1]}
    return out


def locale_matches(value, text):
    """True when the quoted UI text is this locale value, with i18next
    placeholders ({{count}}) allowed to stand for anything."""
    value, text = value.strip(), text.strip()
    if value.lower() == text.lower():
        return True
    if "{{" not in value:
        return False
    parts = [p for p in re.split(r"(\{\{[^}]*\}\})", value) if p]
    # A value made only of placeholders ("{{bound}} {{audit}}") would match any
    # text; it carries no literal words to identify the string by.
    if not any(not p.startswith("{{") and p.strip() for p in parts):
        return False
    pattern = "".join(".+" if part.startswith("{{") else re.escape(part) for part in parts)
    return re.fullmatch(pattern, text, re.I) is not None


def locale_keys(text):
    """Keys whose English value is the quoted text. The main bundle's keys are
    used as is; a nested bundle (locales/dynamic, sql-review, subscription) is
    merged by buildTranslation() under its directory name, so its keys carry
    that name as a prefix."""
    keys = set()
    locales = os.path.join(REPO, "frontend/src/locales")
    for f in glob.glob(os.path.join(locales, "**", "en*.json"), recursive=True):
        try:
            data = json.load(open(f))
        except Exception:
            continue
        sub = os.path.relpath(os.path.dirname(f), locales)
        namespace = "" if sub == "." else sub.replace(os.sep, ".")
        stack = [(namespace, data)]
        while stack:
            prefix, node = stack.pop()
            if isinstance(node, dict):
                for k, v in node.items():
                    stack.append((f"{prefix}.{k}" if prefix else k, v))
            elif isinstance(node, str) and locale_matches(node, text):
                keys.add(prefix)
    return keys


E2E_ROOT = "frontend/tests/e2e"
TEST_FILE = re.compile(r"(\.test\.(ts|tsx|mjs)$|/test-utils/)")


def files_using(needle):
    """Product files under frontend/src that contain needle: the components
    that render a string. Locale JSON and unit tests mention it without
    rendering it, so a commit to them is not a culprit."""
    out = subprocess.run(["git", "-C", REPO, "grep", "-l", "-F", needle, "--", "frontend/src"],
                         capture_output=True, text=True).stdout
    return [l for l in out.splitlines() if not l.endswith(".json") and not TEST_FILE.search(l)]


def repo_spec_path(spec):
    """The spec as a repo-relative path. Playwright's JSON report gives it
    relative to the test root (sql-editor/x.spec.ts); git needs the full path."""
    spec = spec.lstrip("./")
    if spec.startswith("tests/e2e/"):
        spec = "frontend/" + spec
    if not spec.startswith(E2E_ROOT + "/") and os.path.exists(os.path.join(REPO, E2E_ROOT, spec)):
        spec = f"{E2E_ROOT}/{spec}"
    return spec


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--spec", required=True)
    ap.add_argument("--since", required=True)
    # HEAD is the tested checkout. origin/main can move during a 40-60 minute
    # run, and a commit merged after the test cannot have caused its failure.
    ap.add_argument("--head", default="HEAD")
    ap.add_argument("--text", action="append", default=[])
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args()
    a.spec = repo_spec_path(a.spec)

    owners = json.load(open(os.path.join(SKILL_DIR, "owners.json")))
    rng = f"{a.since}..{a.head}"
    # A path outside the e2e tree is a product source file (a compile error
    # names one): commits to that file are the decisive evidence, and its area
    # is the one whose sources contain it.
    source_mode = not a.spec.startswith(E2E_ROOT + "/")
    if source_mode:
        area = next((name for name, cfg in owners["areas"].items()
                     if any(a.spec == s or a.spec.startswith(s.rstrip("/") + "/") for s in cfg["sources"])),
                    "(none)")
    else:
        area = os.path.basename(os.path.dirname(a.spec))
    area_cfg = owners["areas"].get(area, {"sources": [], "owner": None})

    score, info = {}, {}
    def add(found, pts, why):
        for sha, c in found.items():
            info.setdefault(sha, c).setdefault("why", []).append(why)
            score[sha] = score.get(sha, 0) + pts

    if source_mode:
        add(commits(rng, [a.spec]), 3, f"changes {a.spec}, the file that fails")
    # An empty path list would make git log return every commit in the range.
    if area_cfg["sources"]:
        add(commits(rng, area_cfg["sources"]), 1, f"touches {area} sources")
    text_files = {}
    for t in a.text:
        keys = locale_keys(t)
        files = set()
        for k in keys:
            files.update(files_using(f'"{k}"'))
            files.update(files_using(f"'{k}'"))
        files.update(f for f in files_using(t) if f.startswith("frontend/src"))
        text_files[t] = {"keys": sorted(keys), "files": sorted(files)}
        if files:
            add(commits(rng, sorted(files)), 2, f"touches a file rendering '{t}'")
        add(commits(rng, PRODUCT_PATHS, pickaxe=re.escape(t)), 3, f"adds/removes '{t}'")
        for k in keys:
            add(commits(rng, PRODUCT_PATHS, pickaxe=re.escape(k)), 3, f"adds/removes key {k}")

    # Nothing decisive inside the window: the spec may have been red for longer
    # than the ledger knows. Widen to the span between the spec file's own last
    # change and the window's start, so a culprit from weeks ago still surfaces.
    widened = None
    if not any(s >= 3 for s in score.values()):
        spec_base = git("log", "-1", "--format=%H", "--", a.spec).strip()
        is_older = subprocess.run(["git", "-C", REPO, "merge-base", "--is-ancestor", spec_base, a.since],
                                  capture_output=True).returncode == 0
        if spec_base and is_older and spec_base != git("rev-parse", a.since).strip():
            widened = f"{spec_base}..{a.since}"
            note = f"widened to the spec's last change {spec_base[:10]}"
            if area_cfg["sources"]:
                add(commits(widened, area_cfg["sources"]), 1, f"touches {area} sources ({note})")
            for t in a.text:
                files = text_files[t]["files"]
                if files:
                    add(commits(widened, files), 2, f"touches a file rendering '{t}' ({note})")
                add(commits(widened, PRODUCT_PATHS, pickaxe=re.escape(t)), 3, f"adds/removes '{t}' ({note})")
                for k in text_files[t]["keys"]:
                    add(commits(widened, PRODUCT_PATHS, pickaxe=re.escape(k)), 3, f"adds/removes key {k} ({note})")

    ranked = sorted(info.values(), key=lambda c: -score[[s for s, v in info.items() if v is c][0]])
    for c in ranked:
        c["score"] = score[[s for s, v in info.items() if v is c][0]]
        c["linear"] = owners["github_to_linear"].get(c["author"])
    total = int(git("rev-list", "--count", rng).strip() or 0)

    owner, basis = None, None
    if ranked and ranked[0]["score"] >= 3 and ranked[0]["linear"]:
        owner, basis = ranked[0]["linear"], f"author of {ranked[0]['sha']} (#{ranked[0]['pr']}) score {ranked[0]['score']}"
    elif ranked and ranked[0]["score"] >= 3 and ranked[0]["author"] in owners["github_to_linear"]:
        owner, basis = area_cfg.get("owner"), f"top commit by a bot/unmapped author ({ranked[0]['author']}); area default"
    elif area_cfg.get("owner"):
        owner, basis = area_cfg["owner"], "area default (no single commit explains it)"
    else:
        owner, basis = None, "unassigned: no area default and no decisive commit"

    result = {"spec": a.spec, "area": area, "range": rng, "range_widened": widened, "commits_in_range": total,
              "texts": text_files, "candidates": ranked[:10], "owner": owner, "owner_basis": basis,
              "framework_owner": owners.get("framework_owner")}
    if a.json:
        print(json.dumps(result, indent=2))
        return
    print(f"spec {a.spec}  area {area}  range {rng} ({total} commits)" + (f"  widened {widened}" if widened else ""))
    for t, v in text_files.items():
        print(f"text '{t}': keys {v['keys'] or '-'}; files {v['files'] or '-'}")
    print("candidates (score, sha, date, author -> linear, PR, subject):")
    for c in ranked[:10]:
        print(f"  {c['score']:>2}  {c['sha']} {c['date']} {c['author']} -> {c['linear']}  #{c['pr']}  {c['subject'][:80]}")
        for w in c["why"]:
            print(f"       - {w}")
    if not ranked:
        print("  (none)")
    print(f"owner: {owner}  ({basis})")


if __name__ == "__main__":
    main()
