package result

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
	want := `Deploy to prod failed on 1 of 3 databases
1 changed · 1 failed · 1 skipped
Other stages: test (2 done)

FAILED   instances/prod/databases/crm     attempt 2 · 1 file not run
         migrations/0004_add_risk_score.sql  failed, rolled back
           ALTER TABLE account ADD COLUMN risk_score int
           ERROR: column "risk_score" of relation "account" already exists (SQLSTATE 42701)
         Retry or skip: https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/2
DONE     instances/prod/databases/hr      script A · 1 file · 3 statements · 1,204 rows · 3.2s
SKIPPED  instances/prod/databases/legacy  by dba@example.com: decommissioned

Script A · 1 database
  instances/prod/databases/hr
  -- migrations/0003_backfill_grade.sql
  -- 1,200 rows
  UPDATE employee
    SET grade = 'L1'
    WHERE grade IS NULL;
  ANALYZE employee;
  -- 4 rows
  DELETE FROM employee_draft WHERE stale;

Rollout: https://bytebase.example.com/projects/example/plans/101/rollout
`
	require.Equal(t, want, RenderText(sampleResult(), nil))
}

func TestRenderTextFoldsOnlyLongScripts(t *testing.T) {
	fold := func(index int, header string, detail []string) []string {
		return []string{fmt.Sprintf("<fold %d %d lines> %s", index, len(detail), header)}
	}
	text := RenderText(sampleResult(), fold)
	require.Contains(t, text, "DELETE FROM employee_draft", "a short script stays open")
	require.Contains(t, text, "ALTER TABLE account ADD COLUMN risk_score int", "a failure is never folded away")

	result := sampleResult()
	file := result.Stages[1].Tasks[0].Files[0]
	for i := range openScriptLines {
		file.Commands = append(file.Commands, &Command{Statement: fmt.Sprintf("ANALYZE t%d", i), AffectedRows: rows(0)})
	}
	text = RenderText(result, fold)
	require.Contains(t, text, "<fold 0 21 lines> Script A · 1 database · 15 statements\n", "a folded title counts its statements, as in the summary")
	require.NotContains(t, text, "DELETE FROM employee_draft")
}

func TestRenderMarkdown(t *testing.T) {
	want := "### ❌ Deploy to prod failed on 1 of 3 databases\n\n" +
		"1 changed · 1 failed · 1 skipped · [Open the rollout in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout)\n\n" +
		"Other stages: test (2 done)\n\n" +
		"#### ❌ Database crm failed\n\n" +
		"Instance: `prod`<br>\n" +
		"File: `migrations/0004_add_risk_score.sql` (rolled back)\n\n" +
		"```sql\nALTER TABLE account ADD COLUMN risk_score int\n```\n\n" +
		"```text\nERROR: column \"risk_score\" of relation \"account\" already exists (SQLSTATE 42701)\n```\n\n" +
		"[Retry or skip this task in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/2)\n\n" +
		"<details open><summary><strong>Script A · 1 database</strong></summary>\n\n" +
		"hr\n\n" +
		"```sql\n" +
		"-- migrations/0003_backfill_grade.sql\n" +
		"-- 1,200 rows\n" +
		"UPDATE employee\n  SET grade = 'L1'\n  WHERE grade IS NULL;\n" +
		"ANALYZE employee;\n" +
		"-- 4 rows\n" +
		"DELETE FROM employee_draft WHERE stale;\n" +
		"```\n\n" +
		"</details>\n\n" +
		"<details><summary>All 3 databases</summary>\n\n" +
		"| Database | Status | Script | Rows | Duration |\n" +
		"|---|---|---|---|---|\n" +
		"| [crm](https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/2) | ❌ Failed, attempt 2 | – | 0 |  |\n" +
		"| [hr](https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/1) | ✅ Changed | A | 1,204 | 3.2s |\n" +
		"| [legacy](https://bytebase.example.com/projects/example/plans/101/rollout/stages/prod/tasks/3) | ⏭️ Skipped | – |  |  |\n" +
		"\n</details>\n\n"
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
	require.Contains(t, markdown, "20,000 changed · 1 failed · ")
	require.Contains(t, markdown, "#### ❌ Database crm failed", "the failure survives the cut")
	require.Equal(t, shownTableRows-1, strings.Count(markdown, "| ✅ Changed |"), "the failed database takes the first row")
	require.Contains(t, markdown, "and 19,951 more databases, [in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout).")
	require.Contains(t, markdown, "<summary><strong>Script A · 20,000 databases</strong></summary>\n\ntenant_00000, tenant_00001, tenant_00002, tenant_00003, tenant_00004 and 19,995 more", "identical scripts make one")
	require.Contains(t, markdown, "tenant_00049 and 19,950 more databases\n\n</details>")
}

func TestHeadline(t *testing.T) {
	tests := []struct {
		name    string
		outcome Outcome
		edit    func(tasks []*Task)
		want    string
	}{
		{"failed", Failed, nil, "Deploy to prod failed on 1 of 3 databases"},
		{"done", Done, func(tasks []*Task) { tasks[1].Status = "DONE" }, "Deployed to prod"},
		{"done without a change", Done, func(tasks []*Task) {
			tasks[0].Unchanged = true
			tasks[1].Status = "SKIPPED"
		}, "No changes to prod"},
		{"canceled", Canceled, func(tasks []*Task) { tasks[1].Status = "CANCELED" }, "Deploy to prod canceled on 1 of 3 databases"},
		{"interrupted", Interrupted, nil, "Rollout to prod: INTERRUPTED. The action stopped before the stage finished, so this state is not final."},
		{"not waited", NotWaited, nil, "Rollout created. This job deployed nothing: no --target-stage was given."},
		{"stage not found", StageNotFound, nil, "This job deployed nothing: the rollout has no stage for environments/prod."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sampleResult()
			result.Outcome = tc.outcome
			if tc.edit != nil {
				tc.edit(result.Stages[1].Tasks)
			}
			require.Equal(t, tc.want, result.Headline())
		})
	}
}

func TestRenderMarkdownMarksTheHeadline(t *testing.T) {
	result := sampleResult()
	require.True(t, strings.HasPrefix(RenderMarkdown(result), "### ❌ Deploy to prod failed"))

	result.Outcome = Canceled
	result.Stages[1].Tasks[1].Status = "CANCELED"
	markdown := RenderMarkdown(result)
	require.True(t, strings.HasPrefix(markdown, "### ⛔ Deploy to prod canceled on 1 of 3 databases"))
	require.Contains(t, markdown, "#### ⛔ Database crm was canceled")
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
		"[Open the rollout in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout)\n\n" +
		"Stages: test (2 not started)\n\n"
	require.Equal(t, wantMarkdown, RenderMarkdown(result), "with no finished stage to show, the link is the content")
}

func TestRenderMarkdownLinksTheRolloutWhileItIsNotFinal(t *testing.T) {
	result := sampleResult()
	result.Outcome = Interrupted

	require.Contains(t, RenderMarkdown(result), "[Open the rollout in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout)")
}

func TestInstanceAndDatabase(t *testing.T) {
	tests := []struct {
		target       string
		wantInstance string
		wantDatabase string
	}{
		{"instances/prod/databases/hr", "prod", "hr"},
		{"projects/example/instances/prod/databases/hr", "prod", "hr"},
		{"instances/prod", "prod", ""},
	}
	for _, tc := range tests {
		task := &Task{Target: tc.target}
		instance, database := task.instanceAndDatabase()
		require.Equal(t, tc.wantInstance, instance)
		require.Equal(t, tc.wantDatabase, database)
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

func TestWarningsNameAMissingTargetStage(t *testing.T) {
	result := &Result{
		Outcome:     StageNotFound,
		TargetStage: "environments/prod",
		Stages:      []*Stage{{Environment: "environments/test"}, {Environment: "environments/staging"}},
	}
	require.Equal(t, []string{
		"the rollout has no stage for environments/prod, so this job deployed nothing; its stages are environments/test, environments/staging",
	}, result.Warnings())
	require.Empty(t, result.Failures(), "a missing stage does not fail the job")

	result.Stages = nil
	require.Equal(t, []string{
		"the rollout has no stage for environments/prod, so this job deployed nothing; it has no stages",
	}, result.Warnings())

	require.Empty(t, sampleResult().Warnings())
}

func TestTaskNoteDistinguishesMissingRowsFromZero(t *testing.T) {
	reported := &Task{Status: "DONE", Attempt: 1, AffectedRows: rows(0), Files: []*File{{}}}
	missing := &Task{Status: "DONE", Attempt: 1, Files: []*File{{}}}
	unchanged := &Task{Status: "DONE", Attempt: 1, Unchanged: true}

	require.Equal(t, "1 file · 0 rows", reported.note())
	require.Equal(t, "1 file · rows not reported", missing.note())
	require.Equal(t, "already up to date", unchanged.note())
}

func TestCommandLinesCountsWhatItLeavesOut(t *testing.T) {
	var commands []*Command
	for i := range 13 {
		commands = append(commands, &Command{Statement: fmt.Sprintf("UPDATE t SET a = %d", i), AffectedRows: rows(1000)})
	}
	lines := commandLines(commands, len(commands), rows(13000))

	require.Len(t, lines, shownCommands+1)
	require.Equal(t, "1,000 rows  UPDATE t SET a = 0", lines[0])
	require.Equal(t, "and 3,000 rows in 3 more statements", lines[shownCommands])

	// The file ran more commands than the result kept.
	lines = commandLines(commands, 50000, rows(50000000))
	require.Equal(t, "and 49,990,000 rows in 49,990 more statements", lines[shownCommands])

	// Nothing is added when the listed commands account for every row.
	lines = commandLines(commands[:2], 5, rows(2000))
	require.Len(t, lines, 2)
}

func TestShorten(t *testing.T) {
	lines, cut := shorten("  SELECT 1;\n", sqlCut)
	require.Equal(t, []string{"SELECT 1;"}, lines)
	require.False(t, cut)

	lines, cut = shorten("", sqlCut)
	require.Empty(t, lines)
	require.False(t, cut)

	lines, cut = shorten(strings.Repeat("SELECT 1\n", 40), sqlCut)
	require.Len(t, lines, shownLines+1)
	require.Equal(t, "-- 10 more lines", lines[shownLines], "a comment inside SQL")
	require.True(t, cut)

	lines, _ = shorten(strings.Repeat("ERROR\n", 40), textCut)
	require.Equal(t, "... 10 more lines", lines[shownLines])

	// A statement written on one line is cut by width, on a character boundary.
	lines, cut = shorten("INSERT INTO t VALUES "+strings.Repeat("('你好'), ", 100000), sqlCut)
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
	require.Contains(t, markdown, "#### ❌ Database crm failed")
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
	require.Contains(t, markdown, "#### ❌ Database tenant_000 failed")
	require.Equal(t, shownFailures, strings.Count(markdown, "#### ❌ Database "))
	require.Contains(t, markdown, "and 190 more failed databases, [in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout).")
}

func TestCommandsWithoutStatementAreNamedByPosition(t *testing.T) {
	commands := []*Command{
		{Index: 1, AffectedRows: rows(0)},
		{Index: 2, AffectedRows: rows(7)},
		{Index: 3, Error: "duplicate key"},
	}

	require.Equal(t, []string{"7 rows  statement 2"}, commandLines(commands, len(commands), rows(7)))
	require.Equal(t, []string{"statement 3", "duplicate key"}, failureLines(commands, ""))
}

func TestCommandLinesMarkAnEarlierAttempt(t *testing.T) {
	commands := []*Command{
		{Statement: "UPDATE t SET a = 1", AffectedRows: rows(1200), EarlierAttempt: true},
		{Statement: "UPDATE t SET a = 1", AffectedRows: rows(0)},
	}

	require.Equal(t, []string{"1,200 rows  UPDATE t SET a = 1  (earlier attempt)"}, commandLines(commands, len(commands), rows(1200)))
}

func TestCommandLinesMarkAnUnknownCommit(t *testing.T) {
	commands := []*Command{{Index: 1, AffectedRows: rows(1), CommitUnknown: true}}
	require.Equal(t, []string{"1 row  statement 1  (commit outcome unknown)"}, commandLines(commands, 1, nil))
}

func TestFailedDeclarativeFileIsNamedByItsReleasePath(t *testing.T) {
	const version = "20261127.100000"
	failed := taskFixture{id: "hr", status: v1pb.Task_FAILED, release: true, runs: []v1pb.TaskRun_Status{v1pb.TaskRun_FAILED}, log: []*v1pb.TaskRunLogEntry{
		fileEntry(version, ""), computeDiffEntry(), failedCommandEntry("ALTER TABLE t ADD COLUMN a int NOT NULL", "ERROR: column contains null values"),
	}}
	releases := map[string]*v1pb.Release{testReleaseName: {Name: testReleaseName, Files: []*v1pb.Release_File{{Version: version, Path: "schema/*.sql"}}}}

	result := buildResult(t, Input{Releases: releases}, failed)
	require.Contains(t, RenderMarkdown(result), "Instance: `prod`<br>\nFile: `schema/*.sql`")
	require.Contains(t, result.Failures()[0], "failed at schema/*.sql: ")
	require.Contains(t, RenderText(result, nil), "\n        schema/*.sql  failed")

	withoutRelease := buildResult(t, Input{}, failed)
	require.Contains(t, RenderMarkdown(withoutRelease), "File: `version "+version+"`")
}
