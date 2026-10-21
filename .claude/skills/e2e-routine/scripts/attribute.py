#!/usr/bin/env python3
"""Rank the product commits that could explain a red spec, and name an owner.

    attribute.py --spec frontend/tests/e2e/sql-editor/sql-editor-result.spec.ts \
                 --since <last-green-sha> [--text "Request export" ...] [--json]

Candidates are the non-merge commits in <since>..origin/main that touch the
area's source paths (owners.json), the component files that render any quoted
UI text (found through the locale files), or whose diff adds/removes the text
itself (git log -S). Score = 3 per text hit + 1 per path hit. The owner is the
top candidate's author mapped through owners.json, else the area default.
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
    pattern = "".join(".+" if part.startswith("{{") else re.escape(part)
                      for part in re.split(r"(\{\{[^}]*\}\})", value) if part)
    return re.fullmatch(pattern, text, re.I) is not None


def locale_keys(text):
    keys = set()
    for f in glob.glob(os.path.join(REPO, "frontend/src/locales/en*.json")):
        try:
            data = json.load(open(f))
        except Exception:
            continue
        stack = [("", data)]
        while stack:
            prefix, node = stack.pop()
            if isinstance(node, dict):
                for k, v in node.items():
                    stack.append((f"{prefix}.{k}" if prefix else k, v))
            elif isinstance(node, str) and locale_matches(node, text):
                keys.add(prefix)
    return keys


def files_using(needle):
    out = subprocess.run(["git", "-C", REPO, "grep", "-l", "-F", needle, "--", "frontend/src"],
                         capture_output=True, text=True).stdout
    return [l for l in out.splitlines() if not l.endswith(".json")]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--spec", required=True)
    ap.add_argument("--since", required=True)
    ap.add_argument("--head", default="origin/main")
    ap.add_argument("--text", action="append", default=[])
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args()

    owners = json.load(open(os.path.join(SKILL_DIR, "owners.json")))
    area = os.path.basename(os.path.dirname(a.spec))
    area_cfg = owners["areas"].get(area, {"sources": [], "owner": None})
    rng = f"{a.since}..{a.head}"

    score, info = {}, {}
    def add(found, pts, why):
        for sha, c in found.items():
            info.setdefault(sha, c).setdefault("why", []).append(why)
            score[sha] = score.get(sha, 0) + pts

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
        add(commits(rng, (), pickaxe=re.escape(t)), 3, f"adds/removes '{t}'")
        for k in keys:
            add(commits(rng, (), pickaxe=re.escape(k)), 3, f"adds/removes key {k}")

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

    result = {"spec": a.spec, "area": area, "range": rng, "commits_in_range": total,
              "texts": text_files, "candidates": ranked[:10], "owner": owner, "owner_basis": basis,
              "framework_owner": owners.get("framework_owner")}
    if a.json:
        print(json.dumps(result, indent=2))
        return
    print(f"spec {a.spec}  area {area}  range {rng} ({total} commits)")
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
