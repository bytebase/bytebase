# AI review cases

A fixed set of reviews that says how good the AI review is. Run it before a
rollout and after every change to the prompt, once per launch engine and
provider. `TestReviewCases` in `model_ai_cases_test.go` runs every case with
the real `Reviewer`, the real catalog tools, and a live model, then logs one
row per case and a total.

## A case

A case is one YAML file in the directory of its engine, such as
`postgres/index-large-table.yaml`. Its fields:

- `description`: what the case tests, and the fact that decides it.
- `engine`: the `storepb.Engine` name, such as `POSTGRES` or `MYSQL`. The
  directory is its name in lower case.
- `version` and `environment`: what the prompt states about the target.
- `workspace_policy` and `project_policy`: the policy text, both optional.
- `statement`: the SQL under review. Finding lines count from its first line.
- `schema`: the synced schema behind the read and search tools, written as a
  `DatabaseSchemaMetadata` with the proto field names (`row_count`,
  `foreign_keys`, `on_delete`). An unknown field fails the case. The
  database's character set, collation, search path, and owner go into the
  prompt's target section, as they do for a real review.
- `expect.find`: the findings the review must report. Each one names a `line`
  and `keywords`. A finding matches when it is on that line and its title or
  evidence holds any keyword where a word starts, regardless of case: `view`
  matches `views` but not `review`. A keyword can be a phrase, such as
  `no index`.
- `expect.not_flag`: what the review must not report. An entry with a `line`
  forbids any finding on that line, an entry with `keywords` forbids any
  finding that holds one, and an entry with both forbids a finding on that
  line that holds one. A safe change lists every line of its statement that
  holds SQL.

A `find` entry counts only one finding, so two entries need two findings. The
loader rejects an expectation on a blank line or a `--` comment line.

## Scoring

A case passes when the review reports every `find` and nothing a `not_flag`
forbids. The row also counts the other findings, which neither answer a `find`
nor break a `not_flag`; the log prints every finding so that a reader can
judge them.

- `miss`: a `find` was not reported. The test fails.
- `noisy`: every `find` was reported, but so was something a `not_flag`
  forbids. The test passes, since a false flag costs noise, not a missed
  problem; the row and the total show it.
- `error`: the review itself failed. The test fails, and the total counts the
  case's `find` entries as not found.

A model's answers vary from run to run, so compare totals over several runs
before trusting a small difference.

## Adding a case

1. Pick the decision the case tests and make it depend on a fact only the
   schema, the target, or the policy holds. Keep that fact out of the
   statement and out of the names, or the case tests nothing the model could
   not guess.
2. Copy the closest case, change the statement and the fixture, and put each
   statement the expectation names on a line of its own.
3. Name the lines and keywords. Prefer the name of the object that decides the
   case, such as a view, over a word every finding uses.
4. Run `go test ./backend/plugin/aireview/ -run '^TestReviewCasesLoad$'`,
   which checks every case file without a model, then run the case live.

A contrast pair makes the decision visible: two cases that differ in one fact
and expect opposite results, such as `index-large-table` and
`index-small-table`, which differ only in the row count.

## Running

The test is skipped unless `AIREVIEW_CASES_PROVIDER` and
`AIREVIEW_CASES_API_KEY` are set. `AIREVIEW_CASES_MODEL` and
`AIREVIEW_CASES_ENDPOINT` replace the defaults below.

| Provider | Default model | Default endpoint |
| --- | --- | --- |
| `gemini` | `gemini-3.5-flash` | `https://generativelanguage.googleapis.com/v1beta` |
| `openai` | `gpt-5.6-terra` | `https://api.openai.com/v1/responses` |
| `claude` | `claude-sonnet-5` | `https://api.anthropic.com/v1/messages` |

```bash
AIREVIEW_CASES_PROVIDER=gemini AIREVIEW_CASES_API_KEY=... \
  go test ./backend/plugin/aireview/ -run '^TestReviewCases$' -v -count=1 -timeout 30m

AIREVIEW_CASES_PROVIDER=openai AIREVIEW_CASES_API_KEY=... \
  go test ./backend/plugin/aireview/ -run '^TestReviewCases$' -v -count=1 -timeout 30m

AIREVIEW_CASES_PROVIDER=claude AIREVIEW_CASES_API_KEY=... \
  go test ./backend/plugin/aireview/ -run '^TestReviewCases$' -v -count=1 -timeout 30m
```

The cases run in parallel; `-parallel 2` eases a tight rate limit. To run one
engine or one case, extend the pattern, as in
`-run '^TestReviewCases$/postgres/'` or
`-run '^TestReviewCases$/mysql/delete-cascades$'`.
