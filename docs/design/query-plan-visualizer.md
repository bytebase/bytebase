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
| 2 | **Confirm a fix** | An index was added, or the query rewritten. | They compare the costly step against a plan captured before the change. |
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

Also deferred: **explaining a statement from Query History**. It is the right
entry point — CUJ 1 starts with a slow run, usually noticed later — but
`QueryHistory` stores only `database` and `statement`, not the schema or data
source the run used. Explaining `SELECT * FROM orders` from history could
silently plan a different table under a different search path, and an Estimated
badge does not warn about a changed target. It needs history to record execution
context first.

---

## 3. Design

### 3.1 Four triggers

| Trigger | Where the control is | Journey | Executes? |
|---|---|---|---|
| Click the **Plan** tab on a result | in the result pane, next to Text | 1, 2 | no |
| Type `EXPLAIN …`, press **Run** (`Ctrl/Cmd+Enter`) | the editor | 1, 2 | as typed |
| Choose **Explain**, press `Ctrl/Cmd+E`, or right-click → Explain | Run button's dropdown; keyboard; editor context menu | 3 | **no** |
| Choose **Explain analyze** | Run button's dropdown | 1, 2 | **yes** |

Both explain actions go in the dropdown Run already has, not on the toolbar.
Spanner Studio puts "Results only / Explanation only" in exactly that place, and
none of the four reference products ships a toolbar Explain button. They explain
**exactly the statements Run would run** — same selection, same caret rule — so
there is no second scoping rule to learn.

They are two actions, not one, because the difference is whether your query runs.
pgAdmin splits them the same way (F7 / Shift+F7). **Explain** never executes:
given a statement that already says `EXPLAIN ANALYZE`, it refuses and says why,
rather than running an expensive query behind a control whose whole purpose is
to avoid running it. **Explain analyze** executes by definition, and asks for a
machine-readable plan on that single run, so nothing needs re-running afterwards.

Run stays literal: a typed `EXPLAIN ANALYZE` executes, because that is what the
user typed. What we never do is wrap a statement that already starts with
`EXPLAIN` in a second one — pgAdmin, DBeaver, DataGrip, Workbench and pgcli all
prepend blindly, and Workbench turns the resulting error into a dead "Explain
data not available for statement."

### 3.2 The drawing rule

**Rule: draw a picture whenever the engine gave us a machine-readable plan.
Otherwise show the plan as text.**

**The backend states the format on the result; the frontend never infers it.**
The server already knows: it builds explain statements through
`db.ExplainStatement` with an explicit `QueryOption.ExplainFormat`, and it
already parses a typed `EXPLAIN` to classify it (§2). Carrying that answer on
`QueryResult` is the whole mechanism. The driver reports the format it actually
received, and **unknown is a valid answer** — from MySQL 8.0.32 an omitted format
follows the session's `explain_format`, so the statement alone does not settle
it. Unknown means text.

The alternative — reading it off column metadata, as pgAdmin does with
`QUERY PLAN` plus a `json` type — is not available to us on the same terms.
PostgreSQL names the column `QUERY PLAN` and varies only its *type*, and whether
that type survives our driver layer is a question we would have to answer per
engine. The server knows without asking. The frontend still validates that the
payload parses before drawing, so a wrong or unparseable plan degrades to text
rather than to an error.

| Statement Bytebase ran | Result tab shows | `Query()` calls |
|---|---|---|
| **Explain** / **Explain analyze** — we build it, so we ask for the engine's machine-readable format | picture where the engine has one (§3.4) | 1 |
| Typed `EXPLAIN (FORMAT JSON)` | picture, drawn from what came back | 1 — nothing extra |
| Typed `EXPLAIN (FORMAT XML)` | text for now — see below | 1 |
| Typed `EXPLAIN` | text, plus a **Visualize** button | 1, +1 only on Visualize |
| Typed `EXPLAIN ANALYZE` | text; Visualize is disabled, because re-running would execute the query again | 1 — and it stays 1 |
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

**PostgreSQL XML is text until someone writes the parser.** `parsePostgresPlan`
reads JSON only, and `VISUALIZER_EXPLAIN_FORMATS` maps PostgreSQL to JSON, so a
typed `EXPLAIN (FORMAT XML)` has nothing to draw it today. The field set is
identical to JSON — the same `ExplainProperty` calls with a different serializer
— so a reader is a contained piece of work, and DBeaver's own PostgreSQL button
asks for XML, which suggests it is worth doing. It is not in this design's scope,
and until it lands the doc promises text.

**A typed `EXPLAIN ANALYZE` gets no Visualize button**, because the only way to
draw it would be to run the query a second time. The user who wants a picture of
a measured plan uses **Explain analyze** (§3.1), which asks for the structured
plan on its single run. That keeps the cost rule intact without this design
rewriting anyone's statement.

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

1. **Summary panel** — the most expensive operators, ranked; click one and the
   graph jumps to it. Snowflake and Databricks lead with this, because the ranked
   list names the expensive operator outright, with no graph to read first.
   **Rank by measured time where the plan is measured, and by estimated cost
   otherwise, labelling which.** PostgreSQL documents planner costs as arbitrary
   units, and they are wrong in exactly the cases worth investigating, so ranking
   a measured plan by estimate would point at the wrong operator precisely when
   it matters. Where neither is available, say so rather than inventing an order.
   Loops and parallel workers have to be defined before any percentage is shown.
2. **Graph** — opens zoomed on the costliest operator, not the root; long
   single-child chains fold into one box with a count. Nodes show their share of
   cost, the top one badged (Spanner); colour follows cost or rows (BigQuery).
   Zooming in can hide the join that gave that operator its work, so the
   alternative — fit the whole graph, highlight the bottleneck, offer *Jump to
   operator* — goes into the §1 usability test rather than being settled here.
3. **Table** — the same operators as sortable rows, like BigQuery.
4. **Warnings** — already computed in `plan-model.ts` and already rendered, on
   the node in `QueryPlanDiagram` and in `QueryPlanNodeDetails`. The change is
   relevance, and it applies **only to Bytebase's own scan heuristics**: a full
   scan of a 10-row lookup table is the correct plan, and warning on every scan
   teaches readers to ignore the colour. Diagnostics the engine itself reports —
   SQL Server's "Columns with no statistics", "Type conversion affects the plan",
   `NoJoinPredicate` — always stay in node details whatever the cost. Missing
   statistics make a node look cheap, so a cost threshold would hide the warning
   exactly where it is the explanation. Relevance decides what reaches the
   summary panel; it never removes a diagnostic.
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
| Query | becomes a **View SQL** disclosure in the Plan tab |

View SQL is not a tooltip. The editor may have moved on since the result was
produced, so the statement a plan belongs to has to be readable somewhere — and
selectable and copyable, which a hover tooltip is not, least of all for a long
statement.

Plan or Text is remembered per editor tab, like the result pane's height, so
comparing two runs does not mean re-selecting Plan each time (CUJ 2).

CUJ 2 only works on a plan captured **before** the change. A result tab holds the
plan it already fetched, so an earlier run keeps its baseline; opening Plan on
both tabs *after* adding an index gives two current plans and no comparison. Say
which is which rather than implying a before and after.

**Several statements.** Every product gives one plan per statement, never one for
the script. We already do, one result tab each, and we keep them in the editor —
only Spanner Studio also does. What is missing is a signal for which statement
matters: SSMS labels each plan with its share of the script's cost. Add that to
the tab label, marked as an estimate, and only when every statement's cost is
known — a share of an incomplete total is worse than no share.

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

| Engine | Plan tab on a result | Picture | Explain analyze |
|---|---|---|---|
| **PostgreSQL** | one `Query()`, on demand, cached | graph | yes — `EXPLAIN (ANALYZE, FORMAT JSON)` |
| **SQL Server** | one `Query()`, on demand, cached | graph | yes — `SET STATISTICS XML` |
| **Spanner** | one `Query()`, on demand, cached | graph | yes — profile mode |
| **MySQL** | one `Query()`, on demand, cached | text | text only — 8.0's `EXPLAIN ANALYZE` is TREE, and rejects JSON |
| **Oracle, 7 others** | one `Query()`, on demand, cached | text | no |

A plan opened from a result is always an estimate, on every engine. Measured
plans come from **Explain analyze** (§3.1), which is a deliberate action.

**It explains the statement that ran, limits included.** The driver wraps an
ordinary query with the result limit but skips that wrapper on the explain path
(`pg.go:801-804`), so explaining the original text would plan an unbounded query
when the user ran a bounded one — and PostgreSQL plans for `LIMIT`, so the two
differ, not just in cost but in shape. The plan request and its cache carry the
executed statement and its connection context, and View SQL shows that statement,
limit and all. The estimate badge is about statistics moving; it must not quietly
cover a different query.

**Say which one it is, in words.** Measured and estimated carry a badge, and an
estimated plan says *"A new plan, made just now. It may differ from the plan that
ran."* A timestamp alone is not enough: the plan sits in the Plan tab of the
result of the query that was slow, so everything about its placement says it
explains that run. It may not — PostgreSQL chooses a plan from current
statistics, and if the slow run was caused by stale statistics that have since
refreshed, the plan shown is the *good* one, and the reader concludes the query
was never the problem. That is the inverted failure this wording exists to stop.

**Nine of twelve engines have no parser**, so for those the plan is text. Showing
it as formatted text instead of grid rows is a small change with wide reach.
Whether it reaches most *users* is a separate question that engine counts cannot
answer — the §1 instrumentation is what settles which engines people actually
explain, and it should settle the sequencing too. MySQL and Oracle are the
cheapest graphs to add later: MySQL already emits structured JSON, and Oracle's
plan table already holds the parent-child links the driver discards.

**Attaching a measured plan to an ordinary run is deferred.** Two engines could:
Spanner has a run mode returning rows and plan and real timings in one execution,
and SQL Server's `SET STATISTICS XML` does the same. That is how BigQuery,
Snowflake and Databricks work, and it is tempting.

It does not fit here. To have the plan ready when someone later clicks Plan, we
would have to profile *every* run — and Spanner documents profiling overhead and
discourages it for production traffic, while `SET STATISTICS XML` needs
`SHOWPLAN` on every database the statement touches, so turning it on blindly
would fail queries the user is authorized to run. Profiling only when asked
requires a control that says so before the run, which is a second explain action
on the Run button for a case **Explain analyze** already covers. Out of scope.

Once the Plan tab matches the pop-up page, delete the page, its token hand-off
and the old link.

### 3.5 Permissions, masking and limits

**Explain is its own permission** — `bb.sql.explain`, not `bb.sql.select`
(`sql_service.go:1480`) — so a user can be allowed to run a query and not to plan
it. For them the Plan tab is **disabled with the reason shown**, not hidden:
hiding it leaves them unable to tell the capability exists or what to ask an
admin for.

One gap to close: a typed `EXPLAIN ANALYZE` is classified as its *inner*
statement (`pg/query_type.go:47`), so today it passes on `bb.sql.select` alone
and still returns a plan. It must require **both** — the permission to execute
the statement, and `bb.sql.explain` to receive the plan.

**Plans are never masked, and that is a privilege, not a safe default.** Explain
skips masking (`sql_service.go:726`, `:735`). A plan is not row data, but it is
not free of data either: SQL Server's parameter-sensitive plans carry
histogram-derived boundary values, and any engine's predicates and estimates say
something about the distribution behind them. So the justification is not "there
is nothing sensitive in here" — it is that reading a plan is a distinct,
deliberately granted capability, `bb.sql.explain`, held separately from the
permission to read rows. That boundary applies identically to the Plan tab, the
Text tab, Copy plan, and any plan attached to an ordinary run (§3.4): all of them
are the same capability, so none of them may be reachable without it.

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
- **Profile every run everywhere.** Possible only on Spanner and SQL Server, and
  not free even there (§3.4); on PostgreSQL it would run every statement twice.

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
