package result

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const testRolloutURL = "https://bytebase.example.com/" + testRollout

// sdlFixture is a database that ran a declarative release's statements.
func sdlFixture(id string, statements ...string) taskFixture {
	log := []*v1pb.TaskRunLogEntry{fileEntry("20261127.100000", ""), computeDiffEntry()}
	for _, statement := range statements {
		log = append(log, commandEntry(statement, 0))
	}
	return taskFixture{id: id, status: v1pb.Task_DONE, release: true, runs: []v1pb.TaskRun_Status{v1pb.TaskRun_DONE}, log: log}
}

// migrationFixture is a database that ran release files, each a version and
// its statements.
func migrationFixture(id string, files ...[]string) taskFixture {
	var log []*v1pb.TaskRunLogEntry
	for _, file := range files {
		log = append(log, fileEntry(file[0], "migrations/"+file[0]+".sql"))
		for _, statement := range file[1:] {
			log = append(log, commandEntry(statement, 1))
		}
	}
	return taskFixture{id: id, status: v1pb.Task_DONE, release: true, runs: []v1pb.TaskRun_Status{v1pb.TaskRun_DONE}, log: log}
}

// scriptsOf returns each script's databases and its SQL, one line each.
func scriptsOf(w *whatRan) [][2]string {
	var scripts [][2]string
	for _, s := range w.scripts {
		var targets []string
		for _, task := range s.tasks {
			_, database := task.instanceAndDatabase()
			targets = append(targets, database)
		}
		lines, _, _ := s.sqlLines()
		scripts = append(scripts, [2]string{strings.Join(targets, ","), strings.Join(lines, " | ")})
	}
	return scripts
}

const (
	addColumn   = "ALTER TABLE t ADD COLUMN a int"
	createIndex = "CREATE INDEX i ON t (a)"
	dropIndex   = "DROP INDEX old"
	createTable = "CREATE TABLE d (id int)"
)

func TestScriptsGroupDatabasesThatRanTheSameSQL(t *testing.T) {
	result := buildResult(t, Input{},
		sdlFixture("hr", createTable, addColumn, createIndex),
		sdlFixture("crm", createTable, addColumn, createIndex),
		sdlFixture("ops", dropIndex, createTable, addColumn, createIndex),
		sdlFixture("legacy", createTable),
	)

	require.Equal(t, [][2]string{
		{"hr,crm", "CREATE TABLE d (id int); | ALTER TABLE t ADD COLUMN a int; | CREATE INDEX i ON t (a);"},
		{"ops", "DROP INDEX old; | CREATE TABLE d (id int); | ALTER TABLE t ADD COLUMN a int; | CREATE INDEX i ON t (a);"},
		{"legacy", "CREATE TABLE d (id int);"},
	}, scriptsOf(result.whatRan()), "each script is all its databases ran, in order, with nothing marked")

	markdown := RenderMarkdown(result)
	require.Contains(t, markdown, "4 changed · 3 different scripts · [Open the rollout in Bytebase]("+testRolloutURL+")")
	require.Contains(t, markdown, "<details open><summary><strong>Script B · 1 database</strong></summary>\n\nops\n\n"+
		"```sql\nDROP INDEX old;\nCREATE TABLE d (id int);\nALTER TABLE t ADD COLUMN a int;\nCREATE INDEX i ON t (a);\n```\n\n</details>")
	require.NotContains(t, markdown, "-- 0 rows", "DDL shows no count")
}

func TestScriptsOfMigrationFilesCompareFiles(t *testing.T) {
	result := buildResult(t, Input{},
		migrationFixture("hr", []string{"0003", "UPDATE t SET a = 1"}, []string{"0004", addColumn}),
		migrationFixture("crm", []string{"0003", "UPDATE t SET a = 1"}, []string{"0004", addColumn}),
		migrationFixture("ops", []string{"0004", addColumn}),
	)

	require.Equal(t, [][2]string{
		{"hr,crm", "-- migrations/0003.sql | -- 1 row on each database | UPDATE t SET a = 1; | -- migrations/0004.sql | -- 1 row on each database | ALTER TABLE t ADD COLUMN a int;"},
		{"ops", "-- migrations/0004.sql | -- 1 row | ALTER TABLE t ADD COLUMN a int;"},
	}, scriptsOf(result.whatRan()))
	require.Contains(t, RenderText(result, nil), "DONE  instances/prod/databases/ops  script B · 1 file")
	require.Contains(t, RenderText(result, nil), "Script B · 1 database\n  instances/prod/databases/ops\n  -- migrations/0004.sql\n")
}

func TestRowsAcrossReadsForEachDatabase(t *testing.T) {
	require.Equal(t, "25 rows", rowsAcross([]*int64{rows(25)}))
	require.Equal(t, "25 rows on each database", rowsAcross([]*int64{rows(25), rows(25), rows(25)}))
	require.Equal(t, "36 rows across 3 databases", rowsAcross([]*int64{rows(25), rows(10), rows(1)}))
	require.Equal(t, "rows not reported", rowsAcross([]*int64{rows(25), nil}))
}

func TestScriptsSumRowsOnlyWhenEveryDatabaseReported(t *testing.T) {
	unreported := migrationFixture("ops")
	unreported.log = []*v1pb.TaskRunLogEntry{
		fileEntry("0003", "migrations/0003.sql"),
		{Type: v1pb.TaskRunLogEntry_COMMAND_EXECUTE, CommandExecute: &v1pb.TaskRunLogEntry_CommandExecute{Statement: "UPDATE t SET a = 1"}},
	}
	result := buildResult(t, Input{},
		migrationFixture("hr", []string{"0003", "UPDATE t SET a = 1"}),
		unreported,
	)
	require.Equal(t, [][2]string{
		{"hr,ops", "-- migrations/0003.sql | UPDATE t SET a = 1;"},
	}, scriptsOf(result.whatRan()), "a sum with a missing count would understate")
}

func TestScriptOfASheetTaskIsItsSheet(t *testing.T) {
	sheetTask := func(target, sheet, statement string) *Task {
		return &Task{
			Target: "instances/prod/databases/" + target, Status: "DONE", TaskRun: target + "-run", Attempt: 1, sheet: sheet,
			Commands: []*Command{{Index: 1, Statement: statement, AffectedRows: rows(0)}}, CommandCount: 1, AffectedRows: rows(0),
		}
	}
	result := &Result{Outcome: Done, URL: testRolloutURL, Stages: []*Stage{{Environment: "environments/prod", Tasks: []*Task{
		sheetTask("hr", "projects/example/sheets/1", "CREATE TABLE a (id int)"),
		sheetTask("crm", "projects/example/sheets/2", "CREATE TABLE b (id int)"),
		sheetTask("ops", "projects/example/sheets/1", "CREATE TABLE a (id int)"),
	}}}}

	require.Equal(t, [][2]string{
		{"hr,ops", "CREATE TABLE a (id int);"},
		{"crm", "CREATE TABLE b (id int);"},
	}, scriptsOf(result.whatRan()))
}

func TestScriptShowsAFileWithoutItsTextByItsCounts(t *testing.T) {
	result := sampleResult()
	for _, command := range result.Stages[1].Tasks[0].Files[0].Commands {
		command.Statement = ""
	}

	require.Contains(t, RenderMarkdown(result), "```sql\n-- migrations/0003_backfill_grade.sql: 3 statements, 1,204 rows; the text could not be read\n```")
}

func TestScriptsCountTheDatabasesTheyLeaveOut(t *testing.T) {
	unchanged := sdlFixture("ops")
	unreadable := sdlFixture("audit")
	unreadable.log = nil
	retried := sdlFixture("crm", addColumn)
	retried.runs = []v1pb.TaskRun_Status{v1pb.TaskRun_FAILED, v1pb.TaskRun_DONE}
	result := buildResult(t, Input{}, sdlFixture("hr", addColumn), unchanged, unreadable, retried)

	link := "[in Bytebase](" + testRolloutURL + ")"
	require.Contains(t, RenderMarkdown(result), "3 changed · 1 already up to date · [Open the rollout in Bytebase]("+testRolloutURL+")\n\n"+
		"1 database was retried; its earlier runs are "+link+". "+
		"1 database's statements could not be read; they are "+link+".\n\n"+
		"<details open><summary><strong>Script A · 2 databases</strong></summary>\n\nhr, crm\n\n")
	require.Contains(t, RenderText(result, nil), "3 changed · 1 already up to date\n"+
		"1 database was retried; its earlier runs are in Bytebase.\n"+
		"1 database's statements could not be read; they are in Bytebase.\n")
}

func TestScriptsRunByAsManyDatabasesKeepTheStageOrder(t *testing.T) {
	result := buildResult(t, Input{},
		sdlFixture("hr", "CREATE TABLE b (id int)"),
		sdlFixture("crm", "CREATE TABLE a (id int)"),
		sdlFixture("ops", "CREATE TABLE c (id int)"),
	)
	require.Equal(t, [][2]string{
		{"hr", "CREATE TABLE b (id int);"},
		{"crm", "CREATE TABLE a (id int);"},
		{"ops", "CREATE TABLE c (id int);"},
	}, scriptsOf(result.whatRan()))
}

func TestScriptsAreLimited(t *testing.T) {
	var fixtures []taskFixture
	for i := range shownScripts + 2 {
		fixtures = append(fixtures, sdlFixture(fmt.Sprintf("tenant_%02d", i), fmt.Sprintf("CREATE TABLE t%02d (id int)", i)))
	}
	result := buildResult(t, Input{}, fixtures...)

	markdown := RenderMarkdown(result)
	require.Equal(t, shownScripts, strings.Count(markdown, "<summary><strong>Script "))
	require.Contains(t, markdown, "and scripts F and G on 2 databases, [in Bytebase]("+testRolloutURL+").", "the letters the table shows")
	text := RenderText(result, nil)
	require.Equal(t, shownScripts, strings.Count(text, "\nScript "))
	require.Contains(t, text, "\nand scripts F and G on 2 databases, in Bytebase.\n")
}

func TestScriptStatementsAreLimited(t *testing.T) {
	var statements []string
	for i := range shownScriptStatements + 10 {
		statements = append(statements, fmt.Sprintf("CREATE TABLE t%02d (id int)", i))
	}
	result := buildResult(t, Input{}, sdlFixture("hr", statements...))

	markdown := RenderMarkdown(result)
	require.Contains(t, markdown, "<details><summary><strong>Script A · 1 database · 60 statements</strong></summary>", "a long script starts folded")
	require.Contains(t, markdown, "CREATE TABLE t49 (id int);\n```\n\nand 10 more statements, [in Bytebase]("+testRolloutURL+").\n\n</details>")
	require.NotContains(t, markdown, "CREATE TABLE t50")
	require.Contains(t, RenderText(result, nil), "  CREATE TABLE t49 (id int);\n  and 10 more statements, in Bytebase.\n")
}

func TestScriptNamesAndListsItsDatabases(t *testing.T) {
	var fixtures []taskFixture
	for i := range listedDatabases + 10 {
		fixtures = append(fixtures, sdlFixture(fmt.Sprintf("tenant_%02d", i), addColumn))
	}
	result := buildResult(t, Input{}, fixtures...)

	markdown := RenderMarkdown(result)
	require.Contains(t, markdown, "<summary><strong>Script A · 60 databases</strong></summary>\n\ntenant_00, tenant_01, tenant_02, tenant_03, tenant_04 and 55 more\n\n")
	require.Contains(t, markdown, "<details><summary>All 60 databases</summary>\n\ntenant_00, tenant_01,")
	require.Contains(t, markdown, "tenant_49 and 10 more databases\n\n</details>")
	text := RenderText(result, nil)
	require.Contains(t, text, "Script A · 60 databases\n  instances/prod/databases/tenant_00, ")
	require.Contains(t, text, "instances/prod/databases/tenant_49 and 10 more databases\n  ALTER TABLE", "the log lists the databases once")
}

func TestScriptNamesAnInstanceOnlyWhenNamesCollide(t *testing.T) {
	tasks := []*Task{
		{Target: "instances/prod-1/databases/hr"},
		{Target: "instances/prod-2/databases/hr"},
		{Target: "instances/prod-1/databases/crm"},
		{Target: "instances/prod-1"},
	}
	name := markdownNames(tasks)
	require.Equal(t, "hr (instance prod-1)", name(tasks[0]))
	require.Equal(t, "hr (instance prod-2)", name(tasks[1]))
	require.Equal(t, "crm", name(tasks[2]))
	require.Equal(t, "instance prod-1", name(tasks[3]))
}

func TestScriptShortensALongStatement(t *testing.T) {
	result := buildResult(t, Input{}, sdlFixture("hr", "CREATE FUNCTION f() RETURNS int AS $$\n"+strings.Repeat("SELECT 1;\n", 40)+"$$ LANGUAGE sql"))

	require.Contains(t, RenderMarkdown(result), "-- 12 more lines\n```\n\n"+scriptShortenedNote+"\n\n</details>")
}

func TestRenderMarkdownEndsTheScriptsAtOneThatDoesNotFit(t *testing.T) {
	line := strings.Repeat("x", shownLineWidth)
	var fixtures []taskFixture
	for i := range 4 {
		var statements []string
		for j := range shownScriptStatements {
			statements = append(statements, fmt.Sprintf("-- %d %d\n", i, j)+strings.Repeat(line+"\n", shownLines-1))
		}
		fixtures = append(fixtures, sdlFixture(fmt.Sprintf("tenant_%d", i), statements...))
	}
	markdown := RenderMarkdown(buildResult(t, Input{}, fixtures...))

	require.Less(t, len(markdown), 1<<20)
	shown := strings.Count(markdown, "<summary><strong>Script ")
	require.Positive(t, shown)
	require.Less(t, shown, 4)
	require.Contains(t, markdown, fmt.Sprintf("and %s on %s, [in Bytebase]", []string{"scripts B–D", "scripts C and D", "script D"}[shown-1], plural(4-shown, "database")), "the rest are named")
	require.Contains(t, markdown, "<details><summary>All 4 databases</summary>", "the table still fits after the scripts")
}

func TestMoreScriptsNamesThemByLetter(t *testing.T) {
	scripts := func(letters ...string) []*script {
		var rest []*script
		for _, letter := range letters {
			rest = append(rest, &script{letter: letter, tasks: []*Task{{}, {}}})
		}
		return rest
	}
	require.Equal(t, "and script F on 2 databases, in Bytebase.", moreScripts(scripts("F"), "in Bytebase"))
	require.Equal(t, "and scripts F and G on 4 databases, in Bytebase.", moreScripts(scripts("F", "G"), "in Bytebase"))
	require.Equal(t, "and scripts F–H on 6 databases, in Bytebase.", moreScripts(scripts("F", "G", "H"), "in Bytebase"))
}

func TestScriptsLeaveOutADatabasePastItsSQLCap(t *testing.T) {
	huge := "INSERT INTO t VALUES " + strings.Repeat("(1), ", maxDatabaseSQL/5)
	result := buildResult(t, Input{Statements: NewStatementPool()},
		sdlFixture("hr", addColumn),
		sdlFixture("crm", huge),
	)

	markdown := RenderMarkdown(result)
	require.Contains(t, markdown, "Statements run on 1 database are not shown; they are [in Bytebase]("+testRolloutURL+").")
	require.Contains(t, markdown, "<summary><strong>Script A · 1 database</strong></summary>\n\nhr\n\n")
	require.Less(t, len(markdown), 64<<10)
}

func TestScriptsAreOneLinePastTheResultsSQLCap(t *testing.T) {
	statement := func(i int) string {
		return fmt.Sprintf("-- %d\n", i) + strings.Repeat("x", maxDatabaseSQL-100)
	}
	var fixtures []taskFixture
	for i := range maxResultSQL/maxDatabaseSQL + 1 {
		fixtures = append(fixtures, sdlFixture(fmt.Sprintf("tenant_%02d", i), statement(i)))
	}
	result := buildResult(t, Input{Statements: NewStatementPool()}, fixtures...)

	markdown := RenderMarkdown(result)
	require.Contains(t, markdown, droppedScripts("[in Bytebase]("+testRolloutURL+")")+"\n\n<details><summary>All 17 databases</summary>")
	require.NotContains(t, markdown, "<summary><strong>Script")
	require.Contains(t, RenderText(result, nil), "\n"+droppedScripts("in Bytebase")+"\n")
}

// The caps decide the same in any order, since logs are read in parallel.
func TestStatementPoolCapsDoNotDependOnOrder(t *testing.T) {
	commands := func(i int) []*Command {
		return []*Command{{Statement: fmt.Sprintf("-- %d\n", i) + strings.Repeat("x", maxDatabaseSQL/2)}}
	}
	n := 2*maxResultSQL/maxDatabaseSQL + 1
	forward, backward := NewStatementPool(), NewStatementPool()
	for i := range n {
		forward.keep(commands(i))
		backward.keep(commands(n - 1 - i))
	}
	require.True(t, forward.Full())
	require.Equal(t, forward.Full(), backward.Full())

	shared := NewStatementPool()
	for range 100 {
		_, omitted := shared.keep(commands(0))
		require.False(t, omitted)
	}
	require.False(t, shared.Full(), "the same text is held once")
}

func TestScriptLetter(t *testing.T) {
	require.Equal(t, "A", scriptLetter(0))
	require.Equal(t, "Z", scriptLetter(25))
	require.Equal(t, "AA", scriptLetter(26))
	require.Equal(t, "AB", scriptLetter(27))
}

func TestScriptFoldsAtTheSameLengthInTheSummaryAndTheLog(t *testing.T) {
	scriptOf := func(n int) *Result {
		var statements []string
		for i := range n {
			statements = append(statements, fmt.Sprintf("CREATE TABLE t%02d (id int)", i))
		}
		return buildResult(t, Input{}, sdlFixture("hr", statements...))
	}
	folds := func(result *Result) bool {
		var folded bool
		RenderText(result, func(_ int, header string, detail []string) []string {
			folded = true
			return append([]string{header}, detail...)
		})
		return folded
	}

	open := scriptOf(openScriptLines)
	require.Contains(t, RenderMarkdown(open), "<details open><summary><strong>Script A")
	require.False(t, folds(open))

	long := scriptOf(openScriptLines + 1)
	require.NotContains(t, RenderMarkdown(long), "<details open><summary><strong>Script A")
	require.True(t, folds(long))
}

func TestDatabaseThatLoggedNoStatementHasNoScript(t *testing.T) {
	result := &Result{Outcome: Done, URL: testRolloutURL, Stages: []*Stage{{Environment: "environments/prod", Tasks: []*Task{
		{Target: "instances/prod/databases/hr", Status: "DONE", TaskRun: "hr-run", Attempt: 1, sheet: "projects/example/sheets/1"},
	}}}}

	require.Empty(t, result.whatRan().scripts)
	markdown := RenderMarkdown(result)
	require.NotContains(t, markdown, "Script A")
	require.Contains(t, markdown, "<details><summary>Database</summary>", "one database is not \"All 1 database\"")
	require.Contains(t, markdown, "| ✅ Changed | – |")
}
