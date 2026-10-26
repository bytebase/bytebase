# Query plan visualizer

**Status:** partly implemented, see §6 · **Scope:** SQL Editor · Code references are to `main` at `7664688752` unless a PR is named.

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

This ordering is a judgment call, not a measurement.

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

### 3.1 Three triggers

| Trigger | Where the control is | Journey | Executes? |
|---|---|---|---|
| Choose **Plan** in the result's view switch | the result pane, beside Table and Vertical | 1, 2 | no |
| Type `EXPLAIN …`, press **Run** (`Ctrl/Cmd+Enter`) | the editor | 1, 2 | as typed |
| Choose **Explain**, press `Ctrl/Cmd+E`, or right-click → Explain | Run button's dropdown; keyboard; editor context menu | 3 | **no** |

**Explain** goes in the dropdown Run already has, not on the toolbar. Spanner
Studio puts "Results only / Explanation only" in exactly that place, and none of
the four reference products ships a toolbar Explain button. It explains
**exactly the statements Run would run** — same selection, same caret rule — so
there is no second scoping rule to learn. With one refusal: a submission that
sets session state, such as `SET search_path = tenant; SELECT * FROM orders`,
cannot be explained statement by statement. `EXPLAIN SET …` is not valid, and
skipping the `SET` would plan the next statement against the wrong search path,
so the request is refused rather than explaining part of the script — the same
rule §3.3 applies to the Plan view.

**Explain never executes.** Given a statement that already says `EXPLAIN
ANALYZE`, it refuses and says why, rather than running an expensive query behind
a control whose whole purpose is to avoid running it. It asks for the engine's
machine-readable plan, so the picture comes back in one call.

**Explain plans the statement as written, with no limit added.** The result
limit caps how many rows the editor reads back. It is not part of the query, and
appending `LIMIT N` would plan a different statement, because `LIMIT` changes
plan shape. The drivers already leave the limit off on the explain path. A plan
opened from a result is different: it explains the statement that produced the
result, including the limit Run added (§3.4).

**There is no Explain analyze action.** A measured plan comes from typing
`EXPLAIN ANALYZE` and pressing Run, which executes exactly once and is drawn
when it names a format we read (§3.2). Desktop clients do pair the two — pgAdmin
(F7 / Shift+F7), SSMS's estimated and actual plans, SQL Developer's Explain Plan
and Autotrace — but how an engine measures a plan does not make one control: a
statement prefix on PostgreSQL, CockroachDB and TiDB; text only on MySQL 8.0.18
and later; a session setting that returns the rows as well on SQL Server; an API
mode on Spanner; a hint and `DBMS_XPLAN` on Oracle; nothing on ClickHouse or
Redshift; and every run already measured on BigQuery, Snowflake and Databricks.
Built cheaply it would exist for PostgreSQL alone, it would be the one explain
control that executes — writes included — and it would save only remembering the
syntax. Revisit if users ask.

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
`base.ExplainStatement` with an explicit `QueryOption.ExplainFormat`, and it
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
| **Explain** — we build it, so we ask for the engine's machine-readable format | picture where the engine has one (§3.4) | 1 |
| Typed `EXPLAIN (FORMAT JSON)` | picture, drawn from what came back | 1 — nothing extra |
| Typed `EXPLAIN (FORMAT XML)` | text for now — see below | 1 |
| Typed `EXPLAIN` | text, plus a **Visualize** button | 1, +1 only on Visualize |
| Typed `EXPLAIN (ANALYZE, FORMAT JSON)` | picture — it already ran once and came back structured | 1 — nothing extra |
| Typed `EXPLAIN ANALYZE`, no format | text; Visualize is disabled, because re-running would execute the query again | 1 — and it stays 1 |
| Typed `EXPLAIN (FORMAT YAML)` | text — YAML arrives looking identical to plain text, and no viewer reads it | 1 |

**Cost rule: a picture never costs more than one extra `Query()` call, and never
re-executes a statement that already ran.** Today every Visualize click costs an
extra call on PostgreSQL and SQL Server, because the link re-runs `EXPLAIN` just
to change format; drawing from what came back removes it.

**Visualize is a server-side re-explain, not a rewiring.** Changing the gate is
what makes the button appear on a typed `EXPLAIN`; it is not what makes it work.
The existing link replays the same params with a different format
(`getExplainToken`), which leaves `explain` false and returns the same text —
and flipping `explain` true instead would have the driver prefix a second
`EXPLAIN` onto a statement that already starts with one. So Visualize sends that
statement as an explain request naming the format it wants, and the backend
substitutes the format rather than prefixing, as it does for every explain
request whose statement already explains itself.

Substitution today keeps the planned statement and nothing else: the caller's
remaining options (`VERBOSE`, `COSTS OFF`) go with the format they came with,
because the parsers locate the planned statement rather than re-render an option
list. An `EXPLAIN ANALYZE` is refused instead of being replanned, per §3.1, so
what substitution drops is never the difference between planning and executing.

**We do not silently turn a plain `EXPLAIN` into `FORMAT JSON`.** It would change
what the user asked to see, and no SQL editor we surveyed rewrites a typed
`EXPLAIN`. Supabase tried the nearest thing — parsing the text plan — and deleted
it seven months later for non-use.

**Not every result offers Plan.** DDL and anything else the engine cannot
explain gets no Plan option in its view switch — a Plan that cannot plan is
worse than none.

A query killed by the workspace timeout is the case worth having: no measured
plan can exist, but an estimate is usually the answer to why it died. It has a
prerequisite. Today a timeout returns no result at all — `executeWithTimeout`
yields nil (`sql_service.go:999`), `Query` turns that into an RPC error
(`:451-456`), and the client records an empty result set — so there is no
executed statement to explain, and explaining the user's text instead would plan
a differently-limited query, which is what §3.4 rejects. The timeout response
has to carry the executed statement first. Until it does, no Plan here
either.

**PostgreSQL XML is text until someone writes the parser.** `parsePostgresPlan`
reads JSON only, and `VISUALIZER_EXPLAIN_FORMATS` maps PostgreSQL to JSON, so a
typed `EXPLAIN (FORMAT XML)` has nothing to draw it today. The field set is
identical to JSON — the same `ExplainProperty` calls with a different serializer
— so a reader is a contained piece of work, and DBeaver's own PostgreSQL button
asks for XML, which suggests it is worth doing. It is not in this design's scope,
and until it lands the doc promises text.

**An *unformatted* typed `EXPLAIN ANALYZE` gets no Visualize button**, because the
only way to draw it would be to run the query a second time. One that names a
format we can read is drawn like any other structured plan — it has already run,
exactly once, and came back machine-readable, so the drawing rule applies
unchanged. The user who wants a picture of
a measured plan types `EXPLAIN (ANALYZE, FORMAT JSON)`, which returns the
structured plan on its single run. That keeps the cost rule intact without this
design rewriting anyone's statement.

### 3.3 Reading the plan

**A view of the result, in the result pane: not a side drawer, not a tab
row.** No product surveyed uses a drawer. BigQuery, Spanner Studio, SSMS,
DBeaver, DataGrip, pgAdmin and Navicat all put the plan beside the results, as
a tab; BigQuery and Spanner Studio add a full-screen control on that pane,
which is what our Maximize already is. Snowflake and Databricks escalate to a
full page instead, which we rejected in §4. Their tabs exist because the plan
came back with the job. On our engines the plan is fetched on demand, so a
permanent tab promises something that is not there until clicked, and the
result toolbar is already full (§4). So: **Plan** is one more option in the
result pane's view switch, beside Table and Vertical — the plan is another way
of looking at this result, which is what it is — and **Text** is one click
away; every plan viewer we checked keeps the raw text next to the picture, so
Text is not optional. The explain action is that one icon: where the engine
cannot explain, the option is absent; where the user lacks the permission, it
is disabled with the reason (§3.5).

**Text appears only when Plan has a picture.** On the nine engines we cannot draw
(§3.4), the plan *is* text, so a second text view would say the same thing
twice. There, Plan shows the plan as text, and nothing else.

1. **Summary panel** — the most expensive operators, ranked; click one and the
   graph jumps to it. Snowflake and Databricks lead with this, because the ranked
   list names the expensive operator outright, with no graph to read first.
   **Rank by measured time where the plan is measured, and by estimated cost
   otherwise, labelling which.** PostgreSQL documents planner costs as arbitrary
   units, and they are wrong in exactly the cases worth investigating, so ranking
   a measured plan by estimate would point at the wrong operator precisely when
   it matters. Where neither is available, say so rather than inventing an order.

   Either way the metric is **exclusive**: a parent's `Actual Total Time`
   includes its children and is reported per loop, so ranking the raw number
   just finds the root. Subtract the children and multiply out the loops, per
   engine — which is what the model already does for estimates by ranking
   `selfCost` (`planCostliestNodes`), and the measured side has to match it
   before any percentage or opening zoom is shown.
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

**What changes from the pop-up viewer.** The pop-up page offered five tabs —
Diagram, Grid, Summary, Raw plan, Query. Five tabs for one plan is two too many:
Summary is the answer most readers want and should not be hidden behind a tab,
and Query duplicates what the result pane already shows.

| Pop-up | Becomes |
|---|---|
| Diagram | **Diagram**, the tab the viewer opens on, centered on the costliest operator instead of the root, which starts selected |
| Grid | **Table** |
| Summary (tab, only when the plan has costs) | the three costliest operators, ranked, at the top of the **side panel** beside the diagram and the table; the cost timeline and per-operation table are dropped |
| Raw plan | the **Text** tab |
| Query | dropped: the result pane's status bar shows the statement that ran, limit and all, with a copy control |

The statement a plan belongs to still has to be readable, selectable and
copyable, because the editor may have moved on since the result was produced.
The status bar under the Plan view already does that, so the viewer does not
repeat it. What the diagram's bars and edges encode is a hover away on the
totals line rather than a line of its own, since the result pane is short.

The view switch's choice — Table, Vertical, Plan or Text — is remembered per
editor tab, like the result pane's height, so comparing two runs does not mean
re-selecting Plan each time (CUJ 2).

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

**A plan assumes its statement stands alone.** In `SET search_path = tenant;
SELECT * FROM orders`, the second statement's plan depends on the first, and we
explain statements one at a time on a connection that never saw the `SET`. The
driver would also prefix a split script blindly, producing an invalid
`EXPLAIN SET …`. Temporary tables and SQL Server's `USE` break the same way. So
when a script establishes session state, the statements after it get no Plan
option. A result cannot show whether its statement did —
`SELECT set_config('search_path', …)` returns a row like any read — so Plan is
offered only where every earlier statement of the run was a plan that did not
execute.
Reproducing session context is a larger feature than this design, and a plan for
the wrong table is worse than no plan.

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

**It has to be the same table.** The editor knows the connected database, and
only that one. A plan can name another — SQL Server writes the database on every
object it reads, and a query can cross databases — so a node whose table lives
elsewhere says nothing rather than describing the local table that happens to
share its name. The same holds for a name two schemas both use where the plan
names no schema. The wrong table's indexes are worse than none.

**Keep the plan portable.** Experts have tools they trust, so a **Copy plan**
action stays, as it is in the viewer today (`PlanCopyButton`) and in every viewer
we surveyed. Dropping it while retiring the pop-up page would be a quiet
regression.

### 3.4 Engine differences

| Engine | Plan on a result | Picture | Measured plan, typed |
|---|---|---|---|
| **PostgreSQL** | one `Query()`, on demand, cached | graph | `EXPLAIN (ANALYZE, FORMAT JSON)`, drawn |
| **SQL Server** | one `Query()`, on demand, cached | graph | `SET STATISTICS XML ON`, with the rows |
| **Spanner** | one `Query()`, on demand, cached | graph | none in the editor — profile mode is an API, not SQL |
| **MySQL** | one `Query()`, on demand, cached | text | `EXPLAIN ANALYZE` from 8.0.18, TREE text only |
| **Oracle, 7 others** | one `Query()`, on demand, cached | text | per engine, text at most |

A plan opened from a result is always an estimate, on every engine. Measured
plans come from a typed `EXPLAIN ANALYZE` or its engine's equivalent, which is a
deliberate act (§3.1).

**What a measured plan is read for.** PostgreSQL reports a node's time and rows
as averages over its executions, so both are multiplied out to totals before
anything is ranked or compared. SQL Server's actual plan is read for rows alone:
`ActualElapsedms` is cumulative on a row-mode operator but per operator on a
batch-mode one, so a single plan can mean both and a ranking built on it would
be wrong for half of it. Those plans rank by the estimated cost they always
carry, with estimated and actual rows side by side on each node.

**It explains the statement that ran, limits included.** The driver wraps an
ordinary query with the result limit but skips that wrapper on the explain path
(`pg.go:801-809`), so explaining the original text would plan an unbounded query
when the user ran a bounded one — and PostgreSQL plans for `LIMIT`, so the two
differ, not just in cost but in shape. The plan request and its cache carry the
executed statement and its connection context, and the status bar shows that statement,
limit and all. The estimate badge is about statistics moving; it must not quietly
cover a different query.

That requires every driver that rewrites a statement to record what it actually
ran. SQL Server and Spanner did not until
[#21476](https://github.com/bytebase/bytebase/pull/21476) and
[#21481](https://github.com/bytebase/bytebase/pull/21481); the timed-out case
still waits on the timeout response carrying its statement.

**Say which one it is.** Measured and estimated carry a one-word badge at the
end of the viewer's tab row, and the estimated badge's tooltip says *"A new
plan, made just now. It may differ from the plan that ran."* The words are a
hover away rather than a line of their own because vertical space in the result
pane is scarce, and the pane already says "estimated" in the diagram header and
the summary. A timestamp alone is not enough: the plan sits in the Plan view of the
result of the query that was slow, so everything about its placement says it
explains that run. It may not — PostgreSQL chooses a plan from current
statistics, and if the slow run was caused by stale statistics that have since
refreshed, the plan shown is the *good* one, and the reader concludes the query
was never the problem. That is the inverted failure this wording exists to stop.

**Nine of twelve engines have no parser**, so for those the plan is text. Showing
it as formatted text instead of grid rows is a small change with wide reach.
MySQL and Oracle are the
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
requires a control that says so before the run — the Explain analyze action §3.1
leaves out. Out of scope.

The pop-up page, its token hand-off and the old link are deleted
([#21652](https://github.com/bytebase/bytebase/pull/21652)).

### 3.5 Permissions, masking and limits

**Explain is its own permission** — `bb.sql.explain`, not `bb.sql.select`
(`sql_service.go:1480`) — so a user can be allowed to run a query and not to plan
it. For them the Plan option and Explain in Run's dropdown are **disabled with
the reason shown**, not hidden: hiding them leaves them unable to tell the
capability exists or what to ask an admin for.

**Anything that both executes and returns a plan needs both permissions.** A
typed `EXPLAIN ANALYZE` was classified as its *inner* statement, so it passed on
`bb.sql.select` alone and still returned a plan; SQL Server's `SET STATISTICS
XML ON` and `SET STATISTICS PROFILE ON` did the same for every statement after
them. Both now mark the span as an executed plan, so the statement needs its
normal ACL, write-target checks included, **and** `bb.sql.explain`
([#21642](https://github.com/bytebase/bytebase/pull/21642),
[#21648](https://github.com/bytebase/bytebase/pull/21648)). SQL Server's settings are read from the parsed `SET` statement,
not matched as text, so a comment or an option list does not hide them. No
explain request executes — there is no Explain analyze action (§3.1) — so an
explain request needs `bb.sql.explain` alone, and `validateExplainStatements`
refuses any planned form that is not read-only.

**Plans are never masked, and that is a privilege, not a safe default.** Explain
skips masking (`sql_service.go:726`, `:735`). A plan is not row data, but it is
not free of data either: SQL Server's parameter-sensitive plans carry
histogram-derived boundary values, and any engine's predicates and estimates say
something about the distribution behind them. So the justification is not "there
is nothing sensitive in here" — it is that reading a plan is a distinct,
deliberately granted capability, `bb.sql.explain`, held separately from the
permission to read rows. That boundary applies identically to the Plan view, the
Text view, Copy plan, and any plan attached to an ordinary run (§3.4): all of them
are the same capability, so none of them may be reachable without it.

Two ways that boundary leaked, both in `checkDatabaseAccess`
(`sql_service.go:1411-1447`), both closed:

- **It was checked per connection, not per referenced database**, on both
  routes. An explain request had no query spans at all; a typed `EXPLAIN` had
  one, but MySQL extracted it as a pseudo-table with no source columns, so
  nothing per-resource fired there either. Both ended at the same single check
  against the database the editor is connected to, so `EXPLAIN SELECT * FROM
  db_b.secret` on `db_a` yielded `db_b`'s plan to someone with explain on `db_a`
  only. Now the inner statement's span names every database it references —
  typed and generated alike — and `bb.sql.explain` is required on each tracked
  one, through the same per-target check a write uses (`checkExplainAccess`
  over `authorizeWriteTargets`). One instance can hold databases owned by
  different projects, and the connected database's policy says nothing about
  another project's, so a database in another project fails closed, as it does
  for a write. A name is resolved as typed, and case-insensitively where the
  engine folds case; one Bytebase does not track as a database — Oracle's
  `SYS`, Snowflake's `SNOWFLAKE` — is left to the connected database's check. A
  typed `EXPLAIN ANALYZE` carries an `ExplainAnalyze` mark on its span and needs
  the inner statement's permission as well. SQL Server's `SET STATISTICS XML
  ON` also executes and returns a plan, and still passes on `bb.sql.select`
  alone: the explain recognizer knows only SHOWPLAN. That is the one permission
  item still open here.
- **A data access grant exempted it**
  ([#21619](https://github.com/bytebase/bytebase/pull/21619)). The check
  returned early for a granted target before the permission was evaluated, so a
  temporary select or unmask grant carried explain along with it. Access grants
  convey data access; they do not convey the right to read plans, and
  `bb.sql.explain` now comes from IAM on every route.

Reading the statement does not always reveal what it touches. `EXEC dbo.p` is
classified as DML with no source columns (`tsql/query_type.go:42`,
`query_span_extractor.go:501-505`), while SHOWPLAN and `STATISTICS XML` expose
plans for the statements *inside* the procedure — so a procedure reading `db_b`
is authorized against `db_a` alone. Resolving routine dependencies is a larger
feature than this design, and a plan is still worth having: a statement whose
span names no table is checked on the connected database, as every explain was
before. Every query span now carries the tables its statement touches, whatever
the statement type, so this is the only case left to that single check.

**Slow and large plans.** The on-demand call can take as long as any query: show
a skeleton, allow cancel, time out rather than hang. Past a node cap the Plan view
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
- **A Plan tab beside every result.** The warehouses have one because the plan
  came back with the job. On our engines it is fetched on demand, the journey's
  frequency is unmeasured, and the result toolbar already holds search, the
  view switch, the row count, Copy and Export. One icon in the view switch
  costs one icon if nobody uses it; a tab row costs every result.
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

---

## 6. Status

Checked against `main` on 2026-10-26, after [#21652](https://github.com/bytebase/bytebase/pull/21652), plus the §3.1 and §3.3 work on the branch below.

**Landed**

- §3.2 — the result states its plan format (`QueryResult.query_plan`), a typed
  `EXPLAIN (FORMAT JSON)` draws inline, and Visualize substitutes the format on
  the server: [#21485](https://github.com/bytebase/bytebase/pull/21485),
  [#21492](https://github.com/bytebase/bytebase/pull/21492),
  [#21493](https://github.com/bytebase/bytebase/pull/21493).
- §3.4 — SQL Server and Spanner record the statement they ran:
  [#21476](https://github.com/bytebase/bytebase/pull/21476),
  [#21481](https://github.com/bytebase/bytebase/pull/21481).
- §3.5 — the explain gate refuses a smuggled write
  ([#21447](https://github.com/bytebase/bytebase/pull/21447)); an access
  grant no longer carries `bb.sql.explain`
  ([#21619](https://github.com/bytebase/bytebase/pull/21619)); explain is
  required on every database the statement references, with a database in
  another project failing closed, and a typed `EXPLAIN ANALYZE` needs the inner
  statement's permission as well
  ([#21642](https://github.com/bytebase/bytebase/pull/21642)); the Plan option
  is disabled with the reason, a load can be cancelled, and a plan past the
  node cap is shown as text
  ([#21646](https://github.com/bytebase/bytebase/pull/21646)); SQL Server's
  `SET STATISTICS XML` and `PROFILE` need `bb.sql.explain` as well as the
  statement's own permission, and its SHOWPLAN settings are read from the
  parsed statement ([#21648](https://github.com/bytebase/bytebase/pull/21648)).
  §3.5 is closed.
- §3.2/§3.3 — Plan in the result pane's view switch on an ordinary result,
  planning the executed statement, with the estimate wording from §3.4, the
  choice remembered per editor tab
  ([#21646](https://github.com/bytebase/bytebase/pull/21646)).

- §3.3/§3.4 — the viewer opens on the diagram, centered on the costliest
  operator; three tabs (Diagram, Table, Text) instead of five; the costliest
  operators ranked in the side panel; the legend on hover; the pop-up page
  deleted ([#21652](https://github.com/bytebase/bytebase/pull/21652)). The
  ranking and the opening view use estimated cost and say so.

- §3.1 — Explain at the top of Run's dropdown, on the statement as written,
  disabled with the reason without `bb.sql.explain`; it, `Ctrl/Cmd+E` and
  right-click → Explain, in the editor and the admin terminal, ask for the
  drawable format, so the picture takes one call. No limit is added to an
  explain, and there is no Explain analyze action; measured plans come from a
  typed `EXPLAIN ANALYZE`. §3.1 is closed.

- §3.3 — the rest of the reading changes. A measured plan is ranked by the time
  each operator spent — its own total less its children's — and says that is
  what it ranked; the shading, the node cards and the table follow the same
  number, and a plan reporting neither cost nor time says why it ranks nothing.
  Planned and actual rows sit together on a node, with the misestimate spelled
  out as a multiple once it is worth a line. One of Bytebase's own scan
  heuristics reaches the summary panel only where the operator is a tenth of
  the plan, while a diagnostic the engine itself reports always does, grouped
  by warning and never removed from node details. A run that planned every
  statement puts each statement's share of the script's estimated cost on its
  result tab. A node that reads a table Bytebase has synced shows that table's
  size and its indexes, which no standalone viewer can do; a table in another
  database, or a bare name two schemas both use, is left unanswered rather than
  answered with the wrong table's indexes. §3.3 is closed.

**Not yet**

Nothing: every section of this design is implemented. The one question §3.3
left open is not code but evidence — whether the diagram should open zoomed on
the costliest operator, as it does, or fit the whole graph and offer *Jump to
operator*. That belongs to the §1 usability test.
