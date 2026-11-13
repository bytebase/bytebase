package result

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func rows(n int64) *int64 {
	return &n
}

// sampleResult is a prod stage with one changed, one failed and one skipped
// database, after a test stage that finished.
func sampleResult() *Result {
	start := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	end := start.Add(3200 * time.Millisecond)
	base := "https://bytebase.example.com/projects/example/plans/101/rollout"
	return &Result{
		Outcome:     Failed,
		URL:         base,
		TargetStage: "environments/prod",
		Stages: []*Stage{
			{Environment: "environments/test", TaskCounts: map[string]int{"DONE": 2}},
			{
				Environment: "environments/prod",
				TaskCounts:  map[string]int{"DONE": 1, "FAILED": 1, "SKIPPED": 1},
				Tasks: []*Task{
					{
						Target: "instances/prod/databases/hr", Status: "DONE", URL: base + "/stages/prod/tasks/1",
						Attempt: 1, TaskRun: "run-1", StartTime: &start, UpdateTime: &end, AffectedRows: rows(1204),
						Files: []*File{{
							Version: "0003", Path: "migrations/0003_backfill_grade.sql", Status: "DONE", AffectedRows: rows(1204),
							Commands: []*Command{
								{Statement: "UPDATE employee\n  SET grade = 'L1'\n  WHERE grade IS NULL", AffectedRows: rows(1200)},
								{Statement: "ANALYZE employee", AffectedRows: rows(0)},
								{Statement: "DELETE FROM employee_draft WHERE stale", AffectedRows: rows(4)},
							},
						}},
					},
					{
						Target: "instances/prod/databases/crm", Status: "FAILED", URL: base + "/stages/prod/tasks/2",
						Attempt: 2, TaskRun: "run-3", Detail: "failed to execute release file", FilesNotRun: 1, AffectedRows: rows(0),
						Files: []*File{{
							Version: "0004", Path: "migrations/0004_add_risk_score.sql", Status: "FAILED", RolledBack: true, AffectedRows: rows(0),
							Commands: []*Command{
								{Statement: "UPDATE account SET tier = 'basic' WHERE tier IS NULL", AffectedRows: rows(40), RolledBack: true},
								{Statement: "ALTER TABLE account ADD COLUMN risk_score int", Error: `ERROR: column "risk_score" of relation "account" already exists (SQLSTATE 42701)`},
							},
						}},
					},
					{
						Target: "instances/prod/databases/legacy", Status: "SKIPPED", URL: base + "/stages/prod/tasks/3",
						SkippedReason: "decommissioned", Skipper: "users/dba@example.com",
					},
				},
			},
		},
	}
}

func TestRenderText(t *testing.T) {
	want := `Rollout to prod: FAILED
1 changed · 1 failed · 1 skipped
Other stages: test (2 done)

FAILED   instances/prod/databases/crm     attempt 2 · 1 file not run
         migrations/0004_add_risk_score.sql  failed, rolled back
           ALTER TABLE account ADD COLUMN risk_score int
           ERROR: column "risk_score" of relation "account" already exists (SQLSTATE 42701)
         Retry or skip: https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/2
DONE     instances/prod/databases/hr      1 file · 3 statements · 1,204 rows · 3.2s
         migrations/0003_backfill_grade.sql  1,204 rows
           1,200 rows  UPDATE employee SET grade = 'L1' WHERE grade IS NULL
               4 rows  DELETE FROM employee_draft WHERE stale
SKIPPED  instances/prod/databases/legacy  by dba@example.com: decommissioned

Rollout: https://bytebase.example.com/projects/example/plans/101/rollout
`
	require.Equal(t, want, RenderText(sampleResult(), nil))
}

func TestRenderTextFoldsOnlyDoneTasks(t *testing.T) {
	fold := func(index int, header string, detail []string) []string {
		return []string{fmt.Sprintf("<fold %d %d lines> %s", index, len(detail), strings.Fields(header)[1])}
	}
	text := RenderText(sampleResult(), fold)

	require.Contains(t, text, "<fold 1 3 lines> instances/prod/databases/hr")
	require.Contains(t, text, "ALTER TABLE account ADD COLUMN risk_score int", "a failure is never folded away")
	require.NotContains(t, text, "DELETE FROM employee_draft")
}

func TestRenderMarkdown(t *testing.T) {
	want := "### Rollout to prod: FAILED\n\n" +
		"1 changed · 1 failed · 1 skipped\n\n" +
		"Other stages: test (2 done)\n\n" +
		"| Database | Instance | Status | Files | Statements | Rows | Duration |\n" +
		"|---|---|---|---|---|---|---|\n" +
		"| [crm](https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/2) | prod | ❌ FAILED, attempt 2 | 1 (1 not run) | 2 | 0 |  |\n" +
		"| [hr](https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/1) | prod | ✅ DONE | 1 | 3 | 1,204 | 3.2s |\n" +
		"| [legacy](https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/3) | prod | ⏭️ SKIPPED |  |  |  |  |\n" +
		"\n" +
		"#### crm on prod failed\n\n" +
		"`migrations/0004_add_risk_score.sql` failed, rolled back\n\n" +
		"```sql\nALTER TABLE account ADD COLUMN risk_score int\n```\n\n" +
		"```text\nERROR: column \"risk_score\" of relation \"account\" already exists (SQLSTATE 42701)\n```\n\n" +
		"[Retry or skip this task in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/2)\n\n" +
		"<details><summary>Statements run on hr on prod</summary>\n\n" +
		"```text\n" +
		"migrations/0003_backfill_grade.sql  1,204 rows\n" +
		"  1,200 rows  UPDATE employee SET grade = 'L1' WHERE grade IS NULL\n" +
		"      4 rows  DELETE FROM employee_draft WHERE stale\n" +
		"```\n\n" +
		"</details>\n\n"
	require.NotContains(t, want, "](https://bytebase.example.com/projects/example/plans/101/rollout)", "a finished stage needs no link to the rollout")
	require.Equal(t, want, RenderMarkdown(sampleResult()))
}

func TestRenderMarkdownStaysUnderBudget(t *testing.T) {
	result := sampleResult()
	stage := result.Stages[1]
	failed, done := stage.Tasks[1], stage.Tasks[0]
	stage.Tasks = nil
	for i := range 20000 {
		task := *done
		task.Target = fmt.Sprintf("instances/prod/databases/tenant_%05d", i)
		stage.Tasks = append(stage.Tasks, &task)
	}
	stage.Tasks = append(stage.Tasks, failed)

	markdown := RenderMarkdown(result)

	require.Less(t, len(markdown), 1<<20)
	require.Contains(t, markdown, "#### crm on prod failed", "the failure survives the cut")
	require.Contains(t, markdown, " more databases are in the output file and [in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout).")
}

func TestHeadline(t *testing.T) {
	tests := []struct {
		outcome Outcome
		want    string
	}{
		{Done, "Rollout to prod: DONE"},
		{Canceled, "Rollout to prod: CANCELED"},
		{Interrupted, "Rollout to prod: INTERRUPTED. The action stopped before the stage finished, so this state is not final."},
		{NotWaited, "Rollout created. This job deployed nothing: no --target-stage was given."},
		{StageNotFound, "This job deployed nothing: the rollout has no stage for environments/prod."},
	}
	for _, tc := range tests {
		t.Run(string(tc.outcome), func(t *testing.T) {
			result := sampleResult()
			result.Outcome = tc.outcome
			require.Equal(t, tc.want, result.Headline())
		})
	}
}

func TestRenderTextWithoutDetail(t *testing.T) {
	result := &Result{
		Outcome:     StageNotFound,
		URL:         "https://bytebase.example.com/projects/example/plans/101/rollout",
		TargetStage: "environments/prod",
		Stages:      []*Stage{{Environment: "environments/test", TaskCounts: map[string]int{"NOT_STARTED": 2}}},
	}
	want := `This job deployed nothing: the rollout has no stage for environments/prod.
Stages: test (2 not started)
Rollout: https://bytebase.example.com/projects/example/plans/101/rollout
`
	require.Equal(t, want, RenderText(result, nil))

	wantMarkdown := "### This job deployed nothing: the rollout has no stage for environments/prod.\n\n" +
		"[Follow the rollout in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout)\n\n" +
		"Stages: test (2 not started)\n\n"
	require.Equal(t, wantMarkdown, RenderMarkdown(result), "with no finished stage to show, the link is the content")
}

func TestRenderMarkdownLinksTheRolloutWhileItIsNotFinal(t *testing.T) {
	result := sampleResult()
	result.Outcome = Interrupted

	require.Contains(t, RenderMarkdown(result), "[Follow the rollout in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout)")
}

func TestInstanceAndDatabase(t *testing.T) {
	tests := []struct {
		target       string
		wantInstance string
		wantDatabase string
		wantName     string
	}{
		{"instances/prod/databases/hr", "prod", "hr", "hr on prod"},
		{"projects/example/instances/prod/databases/hr", "prod", "hr", "hr on prod"},
		{"instances/prod", "prod", "", "prod"},
	}
	for _, tc := range tests {
		task := &Task{Target: tc.target}
		instance, database := task.instanceAndDatabase()
		require.Equal(t, tc.wantInstance, instance)
		require.Equal(t, tc.wantDatabase, database)
		require.Equal(t, tc.wantName, task.displayName())
	}
}

func TestFailures(t *testing.T) {
	result := sampleResult()
	canceled := &Task{Target: "instances/prod/databases/ops", Status: "CANCELED"}
	noLog := &Task{Target: "instances/prod/databases/audit", Status: "FAILED", Detail: "failed to connect\nto the instance", LogUnavailable: true}
	result.Stages[1].Tasks = append(result.Stages[1].Tasks, canceled, noLog)

	require.Equal(t, []string{
		`instances/prod/databases/crm failed at migrations/0004_add_risk_score.sql: ERROR: column "risk_score" of relation "account" already exists (SQLSTATE 42701)`,
		"instances/prod/databases/audit failed: failed to connect to the instance",
		"instances/prod/databases/ops was canceled",
	}, result.Failures())
}

func TestTaskNoteDistinguishesMissingRowsFromZero(t *testing.T) {
	reported := &Task{Status: "DONE", Attempt: 1, AffectedRows: rows(0), Files: []*File{{}}}
	missing := &Task{Status: "DONE", Attempt: 1, Files: []*File{{}}}
	unchanged := &Task{Status: "DONE", Attempt: 1, Unchanged: true}

	require.Equal(t, "1 file · 0 rows", reported.note())
	require.Equal(t, "1 file · rows not reported", missing.note())
	require.Equal(t, "unchanged, nothing to apply", unchanged.note())
}

func TestCommandLinesCountsWhatItLeavesOut(t *testing.T) {
	var commands []*Command
	for i := range 13 {
		commands = append(commands, &Command{Statement: fmt.Sprintf("UPDATE t SET a = %d", i), AffectedRows: rows(1000)})
	}
	lines := commandLines(commands)

	require.Len(t, lines, shownCommands+1)
	require.Equal(t, "1,000 rows  UPDATE t SET a = 0", lines[0])
	require.Equal(t, "and 3 more statements, 3,000 rows", lines[shownCommands])
}

func TestShorten(t *testing.T) {
	lines, cut := shorten("  SELECT 1;\n")
	require.Equal(t, []string{"SELECT 1;"}, lines)
	require.False(t, cut)

	lines, cut = shorten("")
	require.Empty(t, lines)
	require.False(t, cut)

	lines, cut = shorten(strings.Repeat("SELECT 1\n", 40))
	require.Len(t, lines, shownLines+1)
	require.Equal(t, "... 10 more lines", lines[shownLines])
	require.True(t, cut)

	// A statement written on one line is cut by width, on a character boundary.
	lines, cut = shorten("INSERT INTO t VALUES " + strings.Repeat("('你好'), ", 100000))
	require.Len(t, lines, 1)
	require.True(t, strings.HasSuffix(lines[0], "..."))
	require.Equal(t, shownLineWidth+3, utf8.RuneCountInString(lines[0]))
	require.True(t, utf8.ValidString(lines[0]))
	require.True(t, cut)
}

func TestFailureLinesKeepTheErrorOfAHugeStatement(t *testing.T) {
	huge := "INSERT INTO t VALUES " + strings.Repeat("(1, 'x'), ", 200000)
	commands := []*Command{{Statement: huge, Error: "ERROR: duplicate key value violates unique constraint \"t_pkey\""}}

	lines := failureLines(commands, "")

	require.Equal(t, []string{
		huge[:shownLineWidth] + "...",
		"ERROR: duplicate key value violates unique constraint \"t_pkey\"",
		shortenedNote,
	}, lines)
}

func TestRenderKeepsAFailureWithAHugeStatement(t *testing.T) {
	result := sampleResult()
	failed := result.Stages[1].Tasks[1]
	failed.Files[0].Commands[1].Statement = "INSERT INTO account VALUES " + strings.Repeat("(1, 'x'), ", 200000)

	markdown := RenderMarkdown(result)
	require.Less(t, len(markdown), 16<<10)
	require.Contains(t, markdown, "#### crm on prod failed")
	require.Contains(t, markdown, `ERROR: column "risk_score" of relation "account" already exists (SQLSTATE 42701)`)
	require.Contains(t, markdown, "[Retry or skip this task in Bytebase]")
	require.Contains(t, markdown, shortenedNote)

	text := RenderText(result, nil)
	require.Less(t, len(text), 16<<10)
	require.Contains(t, text, `ERROR: column "risk_score" of relation "account" already exists (SQLSTATE 42701)`)
	require.Contains(t, text, shortenedNote)
}

func TestRenderMarkdownCountsTheFailuresItLeavesOut(t *testing.T) {
	result := sampleResult()
	stage := result.Stages[1]
	template := stage.Tasks[1]
	stage.Tasks = nil
	longError := strings.TrimSpace(strings.Repeat(strings.Repeat("e", shownLineWidth)+"\n", shownLines))
	for i := range 200 {
		task := *template
		task.Target = fmt.Sprintf("instances/prod/databases/tenant_%03d", i)
		task.Files = []*File{{
			Version: "0004", Path: "migrations/0004_add_risk_score.sql", Status: "FAILED",
			Commands: []*Command{{Statement: "ALTER TABLE account ADD COLUMN risk_score int", Error: longError}},
		}}
		stage.Tasks = append(stage.Tasks, &task)
	}

	markdown := RenderMarkdown(result)

	require.Less(t, len(markdown), 1<<20)
	require.Contains(t, markdown, "#### tenant_000 on prod failed")
	shown := strings.Count(markdown, " on prod failed\n")
	require.Positive(t, shown)
	require.Less(t, shown, 200)
	require.Contains(t, markdown, fmt.Sprintf("%d more failures are in the output file and [in Bytebase](", 200-shown))
}

func TestCommandsWithoutStatementAreNamedByPosition(t *testing.T) {
	commands := []*Command{
		{AffectedRows: rows(0)},
		{AffectedRows: rows(7)},
		{Error: "duplicate key"},
	}

	require.Equal(t, []string{"7 rows  statement 2"}, commandLines(commands))
	require.Equal(t, []string{"statement 3", "duplicate key"}, failureLines(commands, ""))
}

func TestCommandLinesMarkAnEarlierAttempt(t *testing.T) {
	commands := []*Command{
		{Statement: "UPDATE t SET a = 1", AffectedRows: rows(1200), EarlierAttempt: true},
		{Statement: "UPDATE t SET a = 1", AffectedRows: rows(0)},
	}

	require.Equal(t, []string{"1,200 rows  UPDATE t SET a = 1  (earlier attempt)"}, commandLines(commands))
}
