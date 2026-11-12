package result

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	testRollout = "projects/example/plans/101/rollout"
	testStage   = testRollout + "/stages/prod"
)

func fileEntry(version, path string) *v1pb.TaskRunLogEntry {
	return &v1pb.TaskRunLogEntry{
		Type:               v1pb.TaskRunLogEntry_RELEASE_FILE_EXECUTE,
		ReleaseFileExecute: &v1pb.TaskRunLogEntry_ReleaseFileExecute{Version: version, FilePath: path},
	}
}

func commandEntry(statement string, rows int64) *v1pb.TaskRunLogEntry {
	return &v1pb.TaskRunLogEntry{
		Type: v1pb.TaskRunLogEntry_COMMAND_EXECUTE,
		CommandExecute: &v1pb.TaskRunLogEntry_CommandExecute{
			Statement: statement,
			Response:  &v1pb.TaskRunLogEntry_CommandExecute_CommandResponse{AffectedRows: rows, AllAffectedRows: []int64{rows}},
		},
	}
}

func failedCommandEntry(statement, err string) *v1pb.TaskRunLogEntry {
	return &v1pb.TaskRunLogEntry{
		Type: v1pb.TaskRunLogEntry_COMMAND_EXECUTE,
		CommandExecute: &v1pb.TaskRunLogEntry_CommandExecute{
			Statement: statement,
			Response:  &v1pb.TaskRunLogEntry_CommandExecute_CommandResponse{Error: err},
		},
	}
}

func transactionEntry(kind v1pb.TaskRunLogEntry_TransactionControl_Type, err string) *v1pb.TaskRunLogEntry {
	return &v1pb.TaskRunLogEntry{
		Type:               v1pb.TaskRunLogEntry_TRANSACTION_CONTROL,
		TransactionControl: &v1pb.TaskRunLogEntry_TransactionControl{Type: kind, Error: err},
	}
}

func retryEntry() *v1pb.TaskRunLogEntry {
	return &v1pb.TaskRunLogEntry{
		Type:      v1pb.TaskRunLogEntry_RETRY_INFO,
		RetryInfo: &v1pb.TaskRunLogEntry_RetryInfo{Error: "lock timeout", RetryCount: 1},
	}
}

var (
	begin    = transactionEntry(v1pb.TaskRunLogEntry_TransactionControl_BEGIN, "")
	commit   = transactionEntry(v1pb.TaskRunLogEntry_TransactionControl_COMMIT, "")
	rollback = transactionEntry(v1pb.TaskRunLogEntry_TransactionControl_ROLLBACK, "")
)

type taskFixture struct {
	id      string
	status  v1pb.Task_Status
	release bool
	pending int32
	// runs are the task's runs, oldest first.
	runs []v1pb.TaskRun_Status
	// log is the latest run's log. A nil log is one that could not be read.
	log []*v1pb.TaskRunLogEntry
}

// buildStage builds the result for one prod stage holding the fixtures and
// returns its tasks.
func buildStage(t *testing.T, fixtures ...taskFixture) []*Task {
	t.Helper()
	created := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	stage := &v1pb.Stage{Name: testStage, Environment: "environments/prod"}
	in := Input{
		URL:         "https://bytebase.example.com",
		TargetStage: "environments/prod",
		Rollout:     &v1pb.Rollout{Name: testRollout, Stages: []*v1pb.Stage{stage}},
		Logs:        map[string]*v1pb.TaskRunLog{},
	}
	for _, f := range fixtures {
		task := &v1pb.Task{
			Name:   testStage + "/tasks/" + f.id,
			Status: f.status,
			Target: "instances/prod/databases/" + f.id,
		}
		if f.release {
			task.Payload = &v1pb.Task_DatabaseUpdate_{DatabaseUpdate: &v1pb.Task_DatabaseUpdate{
				Source:       &v1pb.Task_DatabaseUpdate_Release{Release: "projects/example/releases/r1"},
				ReleaseState: &v1pb.Task_ReleaseState{PendingCount: f.pending},
			}}
		}
		stage.Tasks = append(stage.Tasks, task)
		for i, status := range f.runs {
			run := &v1pb.TaskRun{
				Name:       task.Name + "/taskRuns/" + string(rune('a'+i)),
				Status:     status,
				CreateTime: timestamppb.New(created.Add(time.Duration(i) * time.Minute)),
			}
			in.TaskRuns = append(in.TaskRuns, run)
			if i == len(f.runs)-1 && f.log != nil {
				in.Logs[run.Name] = &v1pb.TaskRunLog{Entries: f.log}
			}
		}
	}
	result := Build(in)
	require.Len(t, result.Stages, 1)
	return result.Stages[0].Tasks
}

func TestDetailStage(t *testing.T) {
	stage := func(environment string, statuses ...v1pb.Task_Status) *v1pb.Stage {
		s := &v1pb.Stage{Name: testRollout + "/stages/" + environment, Environment: "environments/" + environment}
		for _, status := range statuses {
			s.Tasks = append(s.Tasks, &v1pb.Task{Status: status})
		}
		return s
	}
	tests := []struct {
		name        string
		targetStage string
		stages      []*v1pb.Stage
		wantOutcome Outcome
		wantStage   string
	}{
		{
			name:        "no target stage is not waited",
			stages:      []*v1pb.Stage{stage("test", v1pb.Task_NOT_STARTED)},
			wantOutcome: NotWaited,
		},
		{
			name:        "target stage absent from the rollout",
			targetStage: "environments/prod",
			stages:      []*v1pb.Stage{stage("test", v1pb.Task_DONE)},
			wantOutcome: StageNotFound,
		},
		{
			name:        "rollout with no stages",
			targetStage: "environments/prod",
			wantOutcome: StageNotFound,
		},
		{
			name:        "done and skipped tasks are done",
			targetStage: "environments/prod",
			stages:      []*v1pb.Stage{stage("test", v1pb.Task_DONE), stage("prod", v1pb.Task_DONE, v1pb.Task_SKIPPED)},
			wantOutcome: Done,
			wantStage:   "environments/prod",
		},
		{
			name:        "failure in an earlier stage is the detailed stage",
			targetStage: "environments/prod",
			stages:      []*v1pb.Stage{stage("test", v1pb.Task_DONE, v1pb.Task_FAILED), stage("prod", v1pb.Task_NOT_STARTED)},
			wantOutcome: Failed,
			wantStage:   "environments/test",
		},
		{
			name:        "failed wins over canceled in one stage",
			targetStage: "environments/prod",
			stages:      []*v1pb.Stage{stage("prod", v1pb.Task_CANCELED, v1pb.Task_FAILED)},
			wantOutcome: Failed,
			wantStage:   "environments/prod",
		},
		{
			name:        "canceled",
			targetStage: "environments/prod",
			stages:      []*v1pb.Stage{stage("prod", v1pb.Task_DONE, v1pb.Task_CANCELED)},
			wantOutcome: Canceled,
			wantStage:   "environments/prod",
		},
		{
			name:        "running task is interrupted",
			targetStage: "environments/prod",
			stages:      []*v1pb.Stage{stage("prod", v1pb.Task_DONE, v1pb.Task_RUNNING)},
			wantOutcome: Interrupted,
			wantStage:   "environments/prod",
		},
		{
			name:        "a stage after the target is ignored",
			targetStage: "environments/test",
			stages:      []*v1pb.Stage{stage("test", v1pb.Task_DONE), stage("prod", v1pb.Task_FAILED)},
			wantOutcome: Done,
			wantStage:   "environments/test",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome, got := DetailStage(&v1pb.Rollout{Name: testRollout, Stages: tc.stages}, tc.targetStage)
			require.Equal(t, tc.wantOutcome, outcome)
			require.Equal(t, tc.wantStage, got.GetEnvironment())
		})
	}
}

func TestBuildStageCounts(t *testing.T) {
	rollout := &v1pb.Rollout{
		Name: testRollout,
		Stages: []*v1pb.Stage{
			{Name: testRollout + "/stages/test", Environment: "environments/test", Tasks: []*v1pb.Task{{Status: v1pb.Task_DONE}, {Status: v1pb.Task_DONE}}},
			{Name: testStage, Environment: "environments/prod", Tasks: []*v1pb.Task{{Name: testStage + "/tasks/1", Status: v1pb.Task_DONE}, {Name: testStage + "/tasks/2", Status: v1pb.Task_SKIPPED}}},
		},
	}
	result := Build(Input{URL: "https://bytebase.example.com", TargetStage: "environments/prod", Rollout: rollout})

	require.Equal(t, Done, result.Outcome)
	require.Equal(t, "https://bytebase.example.com/"+testRollout, result.URL)
	require.Equal(t, map[string]int{"DONE": 2}, result.Stages[0].TaskCounts)
	require.Empty(t, result.Stages[0].Tasks, "only the stage the run ended on is detailed")
	require.Equal(t, map[string]int{"DONE": 1, "SKIPPED": 1}, result.Stages[1].TaskCounts)
	require.Len(t, result.Stages[1].Tasks, 2)
	require.Equal(t, "https://bytebase.example.com/"+testStage+"/tasks/1", result.Stages[1].Tasks[0].URL)
}

func TestBuildTaskRows(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "hr", status: v1pb.Task_DONE, release: true, pending: 2,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_DONE},
		log: []*v1pb.TaskRunLogEntry{
			fileEntry("0001", "migrations/0001_init.sql"),
			begin,
			commandEntry("CREATE TABLE t (id int)", 0),
			commit,
			fileEntry("0002", "migrations/0002_backfill.sql"),
			begin,
			commandEntry("UPDATE t SET id = 1", 1200),
			commandEntry("DELETE FROM t WHERE id = 2", 4),
			commit,
		},
	})

	task := tasks[0]
	require.Equal(t, int64(1204), *task.AffectedRows)
	require.False(t, task.Unchanged)
	require.Zero(t, task.FilesNotRun)
	require.Len(t, task.Files, 2)
	require.Equal(t, "DONE", task.Files[0].Status)
	require.Equal(t, int64(0), *task.Files[0].AffectedRows, "a count of zero is reported, not dropped")
	require.Equal(t, "migrations/0002_backfill.sql", task.Files[1].Path)
	require.Equal(t, int64(1204), *task.Files[1].AffectedRows)
	require.Equal(t, int64(1200), *task.Files[1].Commands[0].AffectedRows)
	require.Empty(t, task.Files[1].Commands[0].AllAffectedRows, "a single count is not repeated")
}

func TestBuildTaskRowsNotReported(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "hr", status: v1pb.Task_DONE, release: true, pending: 1,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_DONE},
		// A driver that logs no command leaves the file with no entries.
		log: []*v1pb.TaskRunLogEntry{fileEntry("0001", "migrations/0001_init.sql")},
	})

	require.Nil(t, tasks[0].AffectedRows)
	require.Nil(t, tasks[0].Files[0].AffectedRows)
	require.False(t, tasks[0].Unchanged)
}

func TestBuildTaskFailedFileRolledBack(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "crm", status: v1pb.Task_FAILED, release: true, pending: 3,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_FAILED},
		log: []*v1pb.TaskRunLogEntry{
			fileEntry("0001", "migrations/0001_seed.sql"),
			begin,
			commandEntry("INSERT INTO t VALUES (1)", 1),
			commit,
			fileEntry("0002", "migrations/0002_add_column.sql"),
			begin,
			commandEntry("UPDATE account SET tier = 'basic'", 40),
			failedCommandEntry("ALTER TABLE account ADD COLUMN risk_score int", "column already exists"),
			rollback,
		},
	})

	task := tasks[0]
	require.Equal(t, int64(1), *task.AffectedRows, "rolled-back rows are left out of the total")
	require.Equal(t, 1, task.FilesNotRun)
	require.Equal(t, "DONE", task.Files[0].Status)
	require.False(t, task.Files[0].RolledBack)

	failed := task.Files[1]
	require.Equal(t, "FAILED", failed.Status)
	require.True(t, failed.RolledBack)
	require.Equal(t, int64(0), *failed.AffectedRows)
	require.Equal(t, int64(40), *failed.Commands[0].AffectedRows, "the command keeps the count the database returned")
	require.True(t, failed.Commands[0].RolledBack)
	require.Nil(t, failed.Commands[1].AffectedRows)
	require.Equal(t, "column already exists", failed.Commands[1].Error)
}

func TestBuildTaskFailureWithoutLoggedRollback(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "crm", status: v1pb.Task_FAILED, release: true, pending: 1,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_FAILED},
		log: []*v1pb.TaskRunLogEntry{
			fileEntry("0001", "migrations/0001_seed.sql"),
			commandEntry("INSERT INTO t VALUES (1)", 1),
			failedCommandEntry("INSERT INTO t VALUES (1)", "duplicate key"),
		},
	})

	file := tasks[0].Files[0]
	require.Equal(t, "FAILED", file.Status)
	require.False(t, file.RolledBack)
	require.False(t, file.Commands[0].RolledBack)
	require.Equal(t, int64(1), *file.AffectedRows)
}

func TestBuildTaskFailedRollbackClaimsNothing(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "crm", status: v1pb.Task_FAILED, release: true, pending: 1,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_FAILED},
		log: []*v1pb.TaskRunLogEntry{
			fileEntry("0001", "migrations/0001_seed.sql"),
			begin,
			commandEntry("INSERT INTO t VALUES (1)", 1),
			transactionEntry(v1pb.TaskRunLogEntry_TransactionControl_COMMIT, "connection reset"),
			transactionEntry(v1pb.TaskRunLogEntry_TransactionControl_ROLLBACK, "connection reset"),
		},
	})

	file := tasks[0].Files[0]
	require.Equal(t, "FAILED", file.Status)
	require.Equal(t, "commit: connection reset", file.Error)
	require.False(t, file.RolledBack)
}

func TestBuildTaskCommandOutsideTransactionIsNotRolledBack(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "crm", status: v1pb.Task_FAILED, release: true, pending: 1,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_FAILED},
		log: []*v1pb.TaskRunLogEntry{
			fileEntry("0001", "migrations/0001_mixed.sql"),
			begin,
			commandEntry("UPDATE t SET a = 1", 5),
			commit,
			commandEntry("UPDATE t SET b = 1", 7),
			begin,
			commandEntry("UPDATE t SET c = 1", 9),
			rollback,
		},
	})

	file := tasks[0].Files[0]
	require.False(t, file.Commands[0].RolledBack)
	require.False(t, file.Commands[1].RolledBack)
	require.True(t, file.Commands[2].RolledBack)
	require.Equal(t, int64(12), *file.AffectedRows)
}

func TestBuildTaskSupersededAttemptIsNotCounted(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "hr", status: v1pb.Task_DONE, release: true, pending: 2,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_DONE},
		log: []*v1pb.TaskRunLogEntry{
			fileEntry("0001", "migrations/0001_seed.sql"),
			begin,
			commandEntry("INSERT INTO t VALUES (1)", 1),
			commit,
			fileEntry("0002", "migrations/0002_backfill.sql"),
			begin,
			commandEntry("UPDATE t SET a = 1", 30),
			failedCommandEntry("ALTER TABLE t ADD COLUMN b int", "lock timeout"),
			rollback,
			retryEntry(),
			begin,
			commandEntry("UPDATE t SET a = 1", 30),
			commandEntry("ALTER TABLE t ADD COLUMN b int", 0),
			commit,
		},
	})

	task := tasks[0]
	require.Equal(t, int64(31), *task.AffectedRows)
	require.Len(t, task.Files, 2, "a retry in one file does not drop the files before it")
	retried := task.Files[1]
	require.Equal(t, "DONE", retried.Status)
	require.False(t, retried.RolledBack)
	require.Len(t, retried.Commands, 2)
}

func TestBuildTaskLatestRunIsTheResult(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "hr", status: v1pb.Task_DONE, release: true, pending: 1,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_FAILED, v1pb.TaskRun_DONE},
		log: []*v1pb.TaskRunLogEntry{
			fileEntry("0001", "migrations/0001_seed.sql"),
			commandEntry("INSERT INTO t VALUES (1)", 1),
		},
	})

	task := tasks[0]
	require.Equal(t, 2, task.Attempt)
	require.Equal(t, task.Name+"/taskRuns/b", task.TaskRun)
	require.Equal(t, []*Run{{Name: task.Name + "/taskRuns/a", Status: "FAILED"}}, task.PreviousRuns)
	require.Equal(t, int64(1), *task.AffectedRows)
}

func TestBuildTaskSkipped(t *testing.T) {
	tasks := buildStage(t, taskFixture{id: "legacy", status: v1pb.Task_SKIPPED, release: true, pending: 1})

	task := tasks[0]
	require.Equal(t, "SKIPPED", task.Status)
	require.Zero(t, task.Attempt)
	require.Nil(t, task.AffectedRows)
	require.False(t, task.LogUnavailable)
}

func TestBuildTaskUnchanged(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "hr", status: v1pb.Task_DONE, release: true,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_DONE},
		log:  []*v1pb.TaskRunLogEntry{},
	})

	require.True(t, tasks[0].Unchanged)
	require.Nil(t, tasks[0].AffectedRows)
}

func TestBuildSheetTaskHasNoFiles(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "hr", status: v1pb.Task_DONE,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_DONE},
		log: []*v1pb.TaskRunLogEntry{
			begin,
			commandEntry("UPDATE t SET a = 1", 3),
			commit,
		},
	})

	task := tasks[0]
	require.Empty(t, task.Files)
	require.Len(t, task.Commands, 1)
	require.Equal(t, int64(3), *task.AffectedRows)
	require.False(t, task.Unchanged, "a sheet task logs no file, so no file does not mean no change")
}

func TestBuildTaskLogUnavailable(t *testing.T) {
	tasks := buildStage(t,
		taskFixture{id: "hr", status: v1pb.Task_DONE, release: true, runs: []v1pb.TaskRun_Status{v1pb.TaskRun_DONE}},
		taskFixture{id: "crm", status: v1pb.Task_PENDING, release: true, runs: []v1pb.TaskRun_Status{v1pb.TaskRun_PENDING}},
	)

	require.True(t, tasks[0].LogUnavailable)
	require.False(t, tasks[0].Unchanged, "an unread log says nothing about what ran")
	require.False(t, tasks[1].LogUnavailable, "a run that has not started has no log to miss")
}

func TestBuildStoppedRunMarksItsLastFile(t *testing.T) {
	tasks := buildStage(t, taskFixture{
		id: "hr", status: v1pb.Task_CANCELED, release: true, pending: 3,
		runs: []v1pb.TaskRun_Status{v1pb.TaskRun_CANCELED},
		log: []*v1pb.TaskRunLogEntry{
			fileEntry("0001", "migrations/0001_seed.sql"),
			commandEntry("INSERT INTO t VALUES (1)", 1),
			fileEntry("0002", "migrations/0002_backfill.sql"),
			{Type: v1pb.TaskRunLogEntry_COMMAND_EXECUTE, CommandExecute: &v1pb.TaskRunLogEntry_CommandExecute{Statement: "UPDATE t SET a = 1"}},
		},
	})

	task := tasks[0]
	require.Equal(t, "DONE", task.Files[0].Status)
	require.Equal(t, "CANCELED", task.Files[1].Status)
	require.Nil(t, task.Files[1].Commands[0].AffectedRows, "a command with no response reports no count")
	require.Equal(t, 1, task.FilesNotRun)
}

func rangeEntry(start, end int32, rows int64) *v1pb.TaskRunLogEntry {
	return &v1pb.TaskRunLogEntry{
		Type: v1pb.TaskRunLogEntry_COMMAND_EXECUTE,
		CommandExecute: &v1pb.TaskRunLogEntry_CommandExecute{
			Range:    &v1pb.Range{Start: start, End: end},
			Response: &v1pb.TaskRunLogEntry_CommandExecute_CommandResponse{AffectedRows: rows},
		},
	}
}

func TestBuildCommandsCutStatementsFromTheSource(t *testing.T) {
	source := []byte("UPDATE t SET a = 1;\nDELETE FROM t WHERE 你好 = 1;")
	entries := []*v1pb.TaskRunLogEntry{
		rangeEntry(0, 19, 3),
		rangeEntry(19, int32(len(source)), 1),
		rangeEntry(40, 4000, 1),
		commandEntry("SELECT 1", 0),
	}

	commands, _ := buildCommands(entries, source)
	require.Equal(t, "UPDATE t SET a = 1;", commands[0].Statement)
	require.Equal(t, "DELETE FROM t WHERE 你好 = 1;", commands[1].Statement, "the range counts bytes, not characters")
	require.Empty(t, commands[2].Statement, "a range past the source is not guessed at")
	require.Equal(t, "SELECT 1", commands[3].Statement, "a logged statement needs no source")

	commands, _ = buildCommands(entries, nil)
	require.Empty(t, commands[0].Statement)
	require.Equal(t, int64(3), *commands[0].AffectedRows, "the count does not depend on the statement text")
}

func TestNeededSources(t *testing.T) {
	stage := &v1pb.Stage{Name: testStage}
	var runs []*v1pb.TaskRun
	logs := map[string]*v1pb.TaskRunLog{}
	add := func(id string, update *v1pb.Task_DatabaseUpdate, entries ...*v1pb.TaskRunLogEntry) {
		task := &v1pb.Task{Name: testStage + "/tasks/" + id, Payload: &v1pb.Task_DatabaseUpdate_{DatabaseUpdate: update}}
		stage.Tasks = append(stage.Tasks, task)
		run := &v1pb.TaskRun{Name: task.Name + "/taskRuns/a", Status: v1pb.TaskRun_DONE}
		runs = append(runs, run)
		logs[run.Name] = &v1pb.TaskRunLog{Entries: entries}
	}
	release := &v1pb.Task_DatabaseUpdate{Source: &v1pb.Task_DatabaseUpdate_Release{Release: "projects/example/releases/r1"}}
	sheet := &v1pb.Task_DatabaseUpdate{Source: &v1pb.Task_DatabaseUpdate_Sheet{Sheet: "projects/example/sheets/s1"}}
	add("hr", release, fileEntry("0001", "a.sql"), rangeEntry(0, 5, 1), fileEntry("0002", "b.sql"), commandEntry("SELECT 1", 0))
	add("crm", release, fileEntry("0001", "a.sql"), rangeEntry(0, 5, 1))
	add("ops", sheet, rangeEntry(0, 5, 1))

	require.Equal(t, []SourceRef{
		{Release: "projects/example/releases/r1", Version: "0001"},
		{Sheet: "projects/example/sheets/s1"},
	}, NeededSources(stage, runs, logs), "one per source, and none for a file logged with its statements")
	require.Equal(t, "projects/example/releases/r1/files/0001", SourceRef{Release: "projects/example/releases/r1", Version: "0001"}.Key())
	require.Equal(t, "projects/example/sheets/s1", SourceRef{Sheet: "projects/example/sheets/s1"}.Key())
}
