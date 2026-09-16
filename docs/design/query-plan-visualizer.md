# Query plan visualizer

**Status:** draft for review · **Scope:** SQL Editor · Code references are to `main` at `7664688752`.

Bytebase draws query plans, but few users reach the drawing. This covers the
whole journey: how you ask for a plan, when you get a picture, how you read it,
and how that differs per engine. The UI follows BigQuery, Spanner Studio,
Snowflake and Databricks.

---

## 1. CUJs

| # | Journey | Trigger | Done when |
|---|---|---|---|
| 1 | **Fix a slow query** | A run takes seconds or hits the timeout. | They find the expensive step without leaving the editor. |
| 2 | **Confirm a fix** | An index was added, or the query rewritten. | They compare the costly step across two runs. |
| 3 | **Check before running** | A heavy report or data fix on production. | They know what it scans, without running it. |

Journey 3 is the rarest — most people press Run without checking — so the design
spends its effort on what happens *after* a run. Supabase removed its editor
Explain tab, which only served journey 3, for non-use.

This ordering is a guess, so instrument it before building. Four events settle
it: which of the four triggers fired (§3.1), whether the result drew a graph or
fell back to text, whether Visualize was pressed, and time-to-plan. Unnamed
telemetry never gets built, so these are the names.

---

## 2. Background / problem

The editor explains on 12 engines and draws a picture for 3 (PostgreSQL, SQL
Server, Spanner). Reaching the picture takes five steps: Ctrl/Cmd+E → plan rows
in the grid → click a small "Visualize Explain" link → Bytebase runs `EXPLAIN` a
second time for JSON → a separate browser tab opens.

Four problems, ordered by how many users each blocks:

1. **A typed `EXPLAIN` never gets a picture.** The link appears only when the
   request carried the explain flag, which only Ctrl/Cmd+E sets
   (`SingleResultView.tsx:400`), so typing `EXPLAIN SELECT …` and pressing Run
   returns rows and nothing else. The server already knows it is an explain — it
   parses the statement and demands `bb.sql.explain` (`sql_service.go:1480`) —
   but that never reaches the browser.
2. **Hard to find.** One shortcut, one small link.
3. **Out of context.** The pop-up tab has no database, environment, theme or
   language; every click re-runs `EXPLAIN`; a slow run gets it blocked.
4. **Hard to read.** Opens on the root, costliest operator off-screen.

Out of scope: saved plans, plan comparison, index suggestions, live progress.

---

## 3. Design

### 3.1 Four triggers

| Trigger | Where the control is | Journey | `Query()` calls |
|---|---|---|---|
| Click the **Plan** tab on a result | in the result pane, next to Text | 1, 2 | 0 or 1, per engine (§3.4) |
| Type `EXPLAIN …`, press **Run** (`Ctrl/Cmd+Enter`) | the editor | 1, 2 | 1 |
| Choose **Explain only**, press `Ctrl/Cmd+E`, or right-click → Explain | Run button's dropdown; keyboard; editor context menu | 3 | 1 |
| Click **Explain now** on a History row | next to Copy in the History pane | 1, 2 | 1 |

Explain only goes in the dropdown Run already has, not on the toolbar. Spanner
Studio puts "Results only / Explanation only" in exactly that place, and none of
the four reference products ships a toolbar Explain button. It serves the rarest
journey, so it does not earn permanent space. The keyboard shortcut and the
right-click item are the same action and keep working. It explains **exactly the
statements Run would run** — same selection, same caret rule — so there is no
second scoping rule to learn.

The History trigger matters because the other three assume the query is already
in the editor, while CUJ 1 starts with a slow run — usually noticed later, in
history. It stores nothing: History already keeps the statement, so we explain it
fresh and badge it Estimated (§3.4). It does not reproduce the plan that ran back
then; that needs saved plans, which are out of scope. The action is named
**Explain now** for that reason.

One rule for all four: if the statement already starts with `EXPLAIN`, run it as
typed rather than wrapping it twice. pgAdmin, DBeaver, DataGrip, Workbench and
pgcli all prepend blindly; Workbench turns the resulting error into a dead
"Explain data not available for statement."

### 3.2 The drawing rule

**Rule: draw a picture whenever the engine gave us a machine-readable plan.
Otherwise show the plan as text.**

We can tell from the result, so we never read the user's SQL. PostgreSQL labels
the plan column `json` when you asked for `FORMAT JSON`, `xml` for `FORMAT XML`.
SQL Server's plan always arrives under a fixed column name. SSMS and pgAdmin
both work exactly this way.

| Statement Bytebase ran | Result tab shows | `Query()` calls |
|---|---|---|
| **Explain only** — we build it, so we ask for JSON | picture | 1 |
| Typed `EXPLAIN (FORMAT JSON \| XML)` | picture, drawn from what came back | 1 — nothing extra |
| Typed `EXPLAIN` | text, plus a **Visualize** button | 1, +1 only on Visualize |
| Typed `EXPLAIN ANALYZE` | picture, if we take the exception below; otherwise text with Visualize greyed out | 1 — and it must stay 1 |
| Typed `EXPLAIN (FORMAT YAML)` | text — YAML arrives looking identical to plain text, and no viewer reads it | 1 |

**Cost rule: a picture never costs more than one extra `Query()` call, and never
re-executes a statement that already ran.** Today every Visualize click costs an
extra call on PostgreSQL and SQL Server, because the link re-runs `EXPLAIN` just
to change format; drawing from what came back removes it.

Today's Visualize link already fetches a drawable plan — it is just wired to the
request flag instead of the result. Rewiring it is the smallest fix for
problem 1.

**We do not silently turn a plain `EXPLAIN` into `FORMAT JSON`.** It would change
what the user asked to see, and no SQL editor we surveyed rewrites a typed
`EXPLAIN`. Supabase tried the nearest thing — parsing the text plan — and deleted
it seven months later for non-use.

**Not every result has a Plan tab.** DDL and anything else the engine cannot
explain gets none — an empty tab is worse than no tab. The opposite case is the
one that matters most: a query killed by the workspace query timeout **keeps**
its Plan tab. No measured plan exists, by definition, but an estimated one is
usually the answer to why it died.

**Exception worth deciding: `EXPLAIN ANALYZE`.** It runs the query for real, so a
second `Query()` call would execute it twice and double any writes. Either grey
out Visualize there, or ask for JSON on the first call so no second one is
needed. **Recommend the second, for `ANALYZE` only** — the one place this design
departs from every other product, and the one place the cost rule above would
otherwise break.

### 3.3 Reading the plan

**A tab in the result pane, not a side drawer.** No product surveyed uses a
drawer. BigQuery, Spanner Studio, SSMS, DBeaver, DataGrip, pgAdmin and Navicat
all put the plan in a tab beside the results; BigQuery and Spanner Studio add a
full-screen control on that pane, which is what our Maximize already is.
Snowflake and Databricks escalate to a full page instead, which we rejected in
§4. So: a **Plan** tab beside the results, **Text** one click away — every plan
viewer we checked keeps the raw text next to the picture, so Text is not
optional.

**Text appears only when Plan has a picture.** On the nine engines we cannot draw
(§3.4), the plan *is* text, so a second text tab would say the same thing twice
and leave the reader asking why the Plan tab has no plan in it. One tab there,
named Plan, containing the plan.

1. **Summary panel** — total cost and the most expensive operators, ranked; click
   one and the graph jumps to it. Snowflake and Databricks lead with this,
   because the ranked list names the expensive operator outright, with no graph
   to read first.
2. **Graph** — opens zoomed on the costliest operator, not the root; long
   single-child chains fold into one box with a count. Nodes show their share of
   cost, the top one badged (Spanner); colour follows cost or rows (BigQuery).
3. **Table** — the same operators as sortable rows, like BigQuery.
4. **Warnings on the node** — `plan-model.ts` already computes them
   (`PLAN_FULL_TABLE_SCAN`, `PLAN_FULL_INDEX_SCAN`, a `warnings` array per node)
   and nothing shows them today. The ranked list says which operator costs most;
   a warning says what is wrong with it, which is what CUJ 1 actually wants.
   **Show a warning only when the node is also expensive.** A full scan of a
   10-row lookup table is the correct plan, not a problem; warn on every scan and
   readers learn to ignore the colour, which costs us the warnings that matter.
5. **Planned vs actual rows**, wherever the plan is measured — PostgreSQL under
   `ANALYZE`, SQL Server's actual plan, where `EstimateRows` and `ActualRows` sit
   on one node. Expecting 1,000 rows and getting 1,000,000 explains bad plans
   that cost alone does not, and it is the first thing an expert looks for.

The pane is already big enough: [#21459](https://github.com/bytebase/bytebase/pull/21459)
shipped Maximize, a remembered height and the sidebar stepping aside.

**What changes from today's viewer.** The pop-up page offers five tabs —
Diagram, Grid, Summary, Raw plan, Query. Five tabs for one plan is three too
many: Summary is the answer most readers want and should not be hidden behind a
tab, and Query duplicates the editor they came from.

| Today | Becomes |
|---|---|
| Diagram | **Graph**, opening on the costliest operator instead of the root |
| Grid | **Table** |
| Summary (tab, only when the plan has costs) | **always-visible panel** beside the graph |
| Raw plan | the **Text** tab, beside Plan |
| Query | dropped — the result tab already carries its statement in a tooltip |

Dropping Query is safe only because of that tooltip: the editor may have moved on
to another statement since the result was produced, so it is not a reliable place
to read what a plan belongs to.

Plan or Text is remembered per editor tab, like the result pane's height, so
comparing two runs does not mean re-selecting Plan each time (CUJ 2).

**Several statements.** Every product gives one plan per statement, never one for
the script. We already do, one result tab each, and we keep them in the editor —
only Spanner Studio also does. What is missing is a signal for which statement
matters: SSMS labels each plan with its share of the script's cost. Add that to
the tab label, marked as an estimate.

**Several databases.** Batch mode runs one statement across many databases, which
no reference product has to solve — they are all single-connection. We plan only
the **selected** database, which is the rule the result pane already follows:
selecting a database is what runs its query, because `DatabaseQueryContext`
mounts on selection and executes the pending context then. The plan rides along
with that, so switching databases costs no more than it does today.

**One thing no standalone viewer can do.** A node says `Seq Scan on salary`, and
Bytebase already knows that table — its indexes, its row count. Clicking the node
can show them. pev2, depesz and Paste The Plan are fed a pasted text file and can
never do this. It turns "this scan is expensive" into a next step, with no AI
involved, and it is the strongest reason for the plan to live inside Bytebase
rather than in a tab that could have been any web page.

**Keep the plan portable.** Experts have tools they trust, so a **Copy plan**
action stays, as it is in the viewer today (`PlanCopyButton`) and in every viewer
we surveyed. Dropping it while retiring the pop-up page would be a quiet
regression.

### 3.4 Engine differences

| Engine | Opening the Plan tab on a result | Picture | Measured or estimated |
|---|---|---|---|
| **Spanner** | **no `Query()`** — the plan came with the run | graph | **measured** |
| **SQL Server** | **no `Query()`**, if we ask the right way | graph | **measured** |
| **PostgreSQL** | one `Query()`, on demand, cached | graph | estimated |
| **MySQL, Oracle** | one `Query()`, on demand, cached | text | estimated |
| **7 others** | one `Query()`, on demand, cached | text | estimated |

**Say which one it is, in words.** Measured and estimated carry a badge, and an
estimated plan says *"A new plan, made just now. It may differ from the plan that
ran."* A timestamp alone is not enough: the plan sits in the Plan tab of the
result of the query that was slow, so everything about its placement says it
explains that run. It may not — PostgreSQL chooses a plan from current
statistics, and if the slow run was caused by stale statistics that have since
refreshed, the plan shown is the *good* one, and the reader concludes the query
was never the problem. That is the inverted failure this wording exists to stop.

**Most users see text, not a graph** — 9 of 12 engines have no parser. Showing
their plan as formatted text instead of grid rows is small and helps the
majority, so it should not be sequenced last. MySQL and Oracle are the cheapest
graphs to add later: MySQL already emits structured JSON, and Oracle's plan table
already holds the parent-child links the driver discards.

**Two engines give a measured plan with no extra run.** That is the standard:
BigQuery, Snowflake and Databricks keep a plan for every query because the engine
records one while it runs. Spanner has a third run mode returning rows *and* plan
*and* real timings in **one execution**; SQL Server's `SET STATISTICS XML` does
the same, where `SET SHOWPLAN_XML` only estimates. So this costs one execution —
the one the user already asked for — plus profiling overhead we should measure
first. PostgreSQL and MySQL cannot: on both, `EXPLAIN ANALYZE` returns the plan
*instead of* the rows.

Once the Plan tab matches the pop-up page, delete the page, its token hand-off
and the old link.

### 3.5 Permissions, masking and limits

**Explain is its own permission** — `bb.sql.explain`, not `bb.sql.select`
(`sql_service.go:1480`) — so a user can be allowed to run a query and not to plan
it. For them the Plan tab is **disabled with the reason shown**, not hidden:
hiding it leaves them unable to tell the capability exists or what to ask an
admin for.

**Plans are never masked, deliberately.** Explain skips masking
(`sql_service.go:726`, `:735`), which is sound because a plan echoes the literals
in the user's own statement, never values read from the table. Written down here
because §3.1 adds new routes to a plan, and a reviewer should not have to
rediscover why it is safe.

**Slow and large plans.** The on-demand call can take as long as any query: show
a skeleton, allow cancel, time out rather than hang. Past a node cap the Plan tab
falls back to Text instead of drawing something unreadable — SSMS degrades the
same way, at 2 MB.

**When the plan cannot be drawn or fetched.** A plan we fail to parse falls back
to Text rather than erroring; engines change their output between versions, and
PostgreSQL 18 already reformatted part of it. An explain that the engine itself
rejects shows that error in the tab, like any failed query.

---

## 4. Alternatives

- **Keep the pop-up tab.** Only Snowflake opens one, and theirs is a permanent
  page with a link, not a throwaway that loses the database, theme and language.
- **Explain button on the toolbar.** SSMS, pgAdmin and DBeaver do this; none of
  the four does. Permanent space for the rarest journey.
- **Parse the text plan.** PostgreSQL's text output has no escaping, so a string
  containing a newline can fake a plan node. Supabase reverted it; pgcli declined.
- **Profile every run everywhere.** Free on Spanner and SQL Server; on PostgreSQL
  it would run every statement twice.

---

## 5. Research

| | Spanner Studio | BigQuery | Snowflake | Databricks |
|---|---|---|---|---|
| How you ask | Run ▾: Results / Explanation only | nothing; every job keeps its plan | nothing; every query keeps a profile | nothing; every query keeps a profile |
| Where it shows | Explanation tab beside Results | Execution graph tab | Profile page, new tab | Summary panel, then full profile |
| Bigger view | full-screen control | full-screen control | full page | full page |
| Leads with | "Highest latency" badges | heat by slot time | Most expensive nodes | Top operators |
| Typed `EXPLAIN` | not possible | not possible | raw cell | raw cell |
| Several statements | N plans, in the editor | N plans, per job | N plans, via history | N plans, via history |

All four put the plan beside the result, offer a way to enlarge it, and lead with
the most expensive operators; we follow them on all three. Spanner alone
separates measured from estimated, so we follow it there. For a typed `EXPLAIN`
they are no guide — two cannot express one — so §3.2 follows SSMS and pgAdmin.

**Method.** Code read at `main`; walkthrough of a local build (86 screenshots, 24
problems); 20+ SQL editors and 6 plan viewers checked against docs, source and
live servers, September 2026.
