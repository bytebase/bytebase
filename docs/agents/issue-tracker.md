# Issue tracker: Linear

Issues and PRDs for this repo live in Linear. Linear is the source of truth for incoming requests, triage state, and implementation issues.

Use Linear MCP for Linear operations when available; otherwise use `linctl`. The repo is GitHub-hosted for code review and implementation history: GitHub PRs should reference or attach to the relevant Linear issue, but GitHub Issues and GitHub PRs are not treated as the request tracker or triage queue.

## Target team

Product issues go to Linear team `BYT` (Bytebase). File in another team only when the user names it.

## Conventions

Prefer Linear MCP tools for create/read/update/comment operations when the session exposes them. If Linear MCP is unavailable, use these `linctl` commands:

- **Check the target team**: `linctl team get BYT`.
- **Create an issue**: `linctl issue create --team BYT --title "..." --description "..." --labels needs-triage`.
- **Read an issue**: `linctl issue get <issue-key>`.
- **List issues**: `linctl issue list --team BYT` and filter by triage label/status using the mappings in `triage-labels.md`.
- **Search issues**: `linctl issue search "<query>" --team BYT`.
- **Comment on an issue**: `linctl comment create <issue-key> --body "..."`.
- **Apply / remove triage labels**: use `linctl issue update <issue-key> --labels ...`. `linctl issue update --labels` replaces labels, so fetch the current issue first and pass the full intended label set.
- **Close**: `linctl issue update <issue-key> --state "Done"` or the appropriate canceled/won't-fix workflow state, and leave a closing comment when context is needed.
- **Attach a GitHub PR**: `linctl issue attach <issue-key> --pr <pr-number-or-url>`.

## Pull requests as a triage surface

GitHub PRs are **not** a request surface for triage. They may link to Linear issues, but `/triage` should not pull GitHub PRs into the request queue.

## When a skill says "publish to the issue tracker"

Create a Linear issue in the target team using Linear MCP or `linctl`.

## When a skill says "fetch the relevant ticket"

Fetch the Linear issue referenced by the user, normally by Linear issue key or URL, using Linear MCP or `linctl issue get <issue-key>`.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a Linear issue in the target team with child Linear issues as tickets.

- **Map**: a single Linear issue holding the Notes / Decisions-so-far / Fog body. Label it with the tracker equivalent of `wayfinder:map` if that label exists; otherwise use the issue title or description to identify it as the map.
- **Child ticket**: a Linear issue in the map's team. Record ticket type in a label or description field: `research`, `prototype`, `grilling`, or `task`. Link the child to the map with `linctl issue update <child-key> --parent <map-key>` when Linear parent relationships fit.
- **Blocking**: use Linear issue relations: `linctl issue relation add <child-key> --blocked-by <blocker-key>`. List blockers with `linctl issue relation list <child-key>`.
- **Frontier query**: list open child tickets for the map, drop claimed or blocked tickets, and choose the first remaining ticket in map order.
- **Claim**: assign the Linear issue before starting work: `linctl issue assign <issue-key>` or `linctl issue update <issue-key> --assignee me`.
- **Resolve**: comment with the answer, mark the child ticket done, then append a context pointer to the map's Decisions-so-far.
