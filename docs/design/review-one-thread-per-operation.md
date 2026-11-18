# One thread for one operation

Status: step 1 built · 2026-11-18 · [BYT-10533](https://linear.app/bytebase/issue/BYT-10533),
[BYT-10523](https://linear.app/bytebase/issue/BYT-10523)

Written on `main` at `345007290e`. "Today" in this document means that state. It covers the non-goal
of [One review on the plan page](plan-one-review.md).

## Problem

The review posts two threads for one operation. A standard rule flags the operation, and the AI
review flags it again in its own thread. The author reads both and resolves both for one fix. Only
a sentence in the AI review's prompt keeps the two reviewers apart today, and no step compares
their findings. For `WALK_THROUGH` the prompt hands the check to the model on purpose, so a missing
table always gets two threads.

## Goals

1. One operation has one thread. When a rule flags an operation, everything the AI review finds on
   that operation is in the rule's thread.
2. The rule's thread says why. A rule only says an operation is not allowed. The AI review adds
   the facts of the databases and what the operation breaks.
3. A rule finding stays exact. No model answer drops it, changes its rule, its codes, or its
   databases, or lowers its priority.

## Non-goals

- Two rule findings on one operation stay two threads. Each rule is exact and has its own setting.
- The text of each database is not kept. A thread carries one bounded text for all of them.

## How a review runs

### 1. Review

The standard rules run first, then the AI review reviews each database of the sample. This order is
unchanged.

Step 2 changes what the model receives. Today the prompt states each rule with a state. In step 2
the model receives the rule findings of its database, each with an id, and adds text to an id. That
text is never a finding of its own.

### 2. Merge

After the review calls, every finding of a sheet goes to one merge call: the findings of the AI
review, the threads its earlier runs left, and the rule findings. Today the call holds no rule
finding.

The model answers three things:

- Which findings of the AI review report the same problem.
- Which problems are on the operation of a rule finding. Such a problem joins the rule's thread and
  posts no thread of its own.
- Which rule threads already state what this run found. Such a thread keeps its comment.

Four rules keep the merge safe:

- A rule finding is a fixed member. The model adds problems to it and nothing else.
- A list holds one rule finding at most. Two rule findings never join.
- A statement with one finding and no earlier thread skips the call.
- A failed call drops nothing. Every finding posts on its own for that run.

### 3. Thread

A rule's thread keeps the rule, the codes, the databases, and the fingerprint of the rule finding.
Two parts change.

Text:

- The rule's own text comes first.
- Each problem adds its evidence and its fix as plain text. It has no title of its own, so the
  thread reads as one finding.
- A problem keeps the rule it cites when that rule is a sentence of the AI review policy. A
  sentence of the review's own prompt, or a standard rule, is left out.
- A rule result that lists several statements shows each problem under the line of its statement.

Priority:

- The thread shows the highest priority among the rule and its problems.
- A problem that restates the rule's condition never raises it. The rule sets the priority of its
  own condition.
- A policy sentence that names a rule's check counts as any other problem, so the policy's priority
  shows when it is the higher one.

Step 2 bounds the text across databases: one sentence for what every database shares, then three
lines at most for the databases that differ, the worst case first. Every database is in one of the
lines.

### 4. Rerun

A rule finding keeps its thread while its statement is unchanged. The text of the AI review differs
on every run, so the merge call decides what the thread shows:

- The model says the comment already states this run's problems: the comment stays.
- Anything else: the text of the AI review is written again from this run.
- This run finds no problem on the operation: the thread shows the rule's text alone, and the
  priority is the rule's again.
- The priority rises: the comment is written again, and a resolved thread is reopened.
- The AI review is off: the thread shows the rule's text alone.

A comment stays only when it was written for every database of the rule finding, and this run's
problems are on every one of them too. The merge call sees no database name, so it cannot tell
that a problem moved to other databases. A comment that names a part of the databases is written
again.

## Storage

No field is added. The text of the AI review is part of the thread's comment.

Step 3 removes `run_type` from a stored finding. One thread then holds text of both reviewers, so
one value for a thread is wrong. A rule finding is the one that carries `rule_type`, which is how
the plan page tells it today.

## Alternatives considered

- Keep the reviewers apart by the prompt alone. Rejected. It is a request to the model, and it has
  failed in both directions: a repeat in BYT-10533 and a missed finding in BYT-10507.
- Keep a different outcome in its own thread, such as a view the operation breaks. Rejected. The
  outcomes come from one operation, and the author fixes or accepts that operation.
- Store the text of the AI review in fields of its own. Rejected. The model reads a comment as it
  is, and a field for each part adds storage that only a rerun would read.
- Post the text of the AI review as replies. Rejected. Replies are what people say in time order,
  and a problem that is fixed would have to be removed from the middle of them.

## Build plan

1. The merge step. Rule findings join the merge call, a problem on a rule finding joins its
   thread, and a rerun keeps or rewrites the comment. The review prompt is unchanged. This closes
   BYT-10533.
2. The model receives the rule findings with ids and adds text to them. The text across databases
   is bounded. This is BYT-10523.
3. `run_type` is removed from a stored finding.
