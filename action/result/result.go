// Package result builds what a rollout job reports from the rollout, its task
// runs and their logs. It makes no calls.
package result

import (
	"regexp"
	"slices"
	"strings"
	"time"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Outcome is how the rollout stood when the action stopped.
type Outcome string

const (
	// Done means every task through the target stage is DONE or SKIPPED.
	Done Outcome = "DONE"
	// Failed means a task is FAILED.
	Failed Outcome = "FAILED"
	// Canceled means a task is CANCELED.
	Canceled Outcome = "CANCELED"
	// NotWaited means no target stage was given, so nothing was awaited.
	NotWaited Outcome = "NOT_WAITED"
	// StageNotFound means the rollout has no stage for the target stage.
	StageNotFound Outcome = "STAGE_NOT_FOUND"
	// Interrupted means the action stopped before the stage finished.
	Interrupted Outcome = "INTERRUPTED"
)

// Result is the rollout as the job reports it. Enum values are the API's.
type Result struct {
	Outcome     Outcome  `json:"outcome"`
	URL         string   `json:"url"`
	TargetStage string   `json:"targetStage,omitempty"`
	Stages      []*Stage `json:"stages"`
}

type Stage struct {
	Environment string         `json:"environment"`
	TaskCounts  map[string]int `json:"taskCounts"`
	// Tasks is set on the one stage the result details.
	Tasks []*Task `json:"tasks,omitempty"`
}

// Task is one database. The run fields describe its latest task run.
type Task struct {
	Name          string `json:"name"`
	Target        string `json:"target"`
	Status        string `json:"status"`
	URL           string `json:"url"`
	SkippedReason string `json:"skippedReason,omitempty"`
	Skipper       string `json:"skipper,omitempty"`

	// Attempt is the number of task runs the task has had.
	Attempt      int        `json:"attempt,omitempty"`
	TaskRun      string     `json:"taskRun,omitempty"`
	Detail       string     `json:"detail,omitempty"`
	StartTime    *time.Time `json:"startTime,omitempty"`
	UpdateTime   *time.Time `json:"updateTime,omitempty"`
	PreviousRuns []*Run     `json:"previousRuns,omitempty"`

	// AffectedRows is nil when no command reported a count.
	AffectedRows *int64 `json:"affectedRows,omitempty"`
	// Unchanged is a release task that finished with no file left to apply.
	Unchanged bool `json:"unchanged,omitempty"`
	// FilesNotRun counts the release's files a stopped run never reached.
	FilesNotRun int     `json:"filesNotRun,omitempty"`
	Files       []*File `json:"files,omitempty"`
	// Commands holds what ran outside any release file, as a sheet task's does.
	Commands []*Command `json:"commands,omitempty"`
	// LogUnavailable means the run's log could not be read, so the fields
	// above carry no execution detail.
	LogUnavailable bool `json:"logUnavailable,omitempty"`
}

// Run is an earlier task run of a task.
type Run struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type File struct {
	Version string `json:"version"`
	Path    string `json:"path,omitempty"`
	Status  string `json:"status"`
	// RolledBack means a transaction in the file was rolled back.
	RolledBack bool `json:"rolledBack,omitempty"`
	// AffectedRows leaves out rolled-back commands.
	AffectedRows *int64 `json:"affectedRows,omitempty"`
	// Error is a failure outside any command, such as a commit that failed.
	Error    string     `json:"error,omitempty"`
	Commands []*Command `json:"commands,omitempty"`
}

type Command struct {
	Statement string `json:"statement"`
	// AffectedRows is nil when the command failed or logged no response.
	AffectedRows *int64 `json:"affectedRows,omitempty"`
	// AllAffectedRows is set when the command held several statements and the
	// driver counted each.
	AllAffectedRows []int64 `json:"allAffectedRows,omitempty"`
	RolledBack      bool    `json:"rolledBack,omitempty"`
	// EarlierAttempt marks a command that committed in an attempt the
	// executor then retried. It ran again in the final attempt.
	EarlierAttempt bool   `json:"earlierAttempt,omitempty"`
	Error          string `json:"error,omitempty"`
}

// DetailStage returns the outcome and the stage the result details: the first
// stage through the target that failed, was canceled or is unfinished, or the
// target stage itself. The stage is nil when nothing was awaited.
func DetailStage(rollout *v1pb.Rollout, targetStage string) (Outcome, *v1pb.Stage) {
	if targetStage == "" {
		return NotWaited, nil
	}
	target := -1
	for i, stage := range rollout.GetStages() {
		if stage.GetEnvironment() == targetStage {
			target = i
			break
		}
	}
	if target < 0 {
		return StageNotFound, nil
	}
	for _, stage := range rollout.GetStages()[:target+1] {
		var failed, canceled, unfinished bool
		for _, task := range stage.GetTasks() {
			switch task.GetStatus() {
			case v1pb.Task_FAILED:
				failed = true
			case v1pb.Task_CANCELED:
				canceled = true
			case v1pb.Task_DONE, v1pb.Task_SKIPPED:
			default:
				unfinished = true
			}
		}
		switch {
		case failed:
			return Failed, stage
		case canceled:
			return Canceled, stage
		case unfinished:
			return Interrupted, stage
		}
	}
	return Done, rollout.GetStages()[target]
}

// GroupTaskRuns returns each task's runs, oldest first, keyed by task name.
func GroupTaskRuns(taskRuns []*v1pb.TaskRun) map[string][]*v1pb.TaskRun {
	groups := map[string][]*v1pb.TaskRun{}
	for _, run := range taskRuns {
		task, _, ok := strings.Cut(run.GetName(), "/taskRuns/")
		if !ok {
			continue
		}
		groups[task] = append(groups[task], run)
	}
	for _, runs := range groups {
		slices.SortStableFunc(runs, func(a, b *v1pb.TaskRun) int {
			return a.GetCreateTime().AsTime().Compare(b.GetCreateTime().AsTime())
		})
	}
	return groups
}

// HasLog reports whether a task run has started and so has a log to read.
func HasLog(run *v1pb.TaskRun) bool {
	return run.GetStatus() != v1pb.TaskRun_PENDING && run.GetStatus() != v1pb.TaskRun_AVAILABLE
}

// Input is everything Build reads.
type Input struct {
	// URL is the Bytebase URL, without a trailing slash.
	URL         string
	TargetStage string
	Rollout     *v1pb.Rollout
	// TaskRuns are the task runs of the detailed stage.
	TaskRuns []*v1pb.TaskRun
	// Logs are keyed by task run name. A started run with no entry is
	// reported as having no log.
	Logs map[string]*v1pb.TaskRunLog
	// Sources hold what the commands ran, keyed by SourceKey for a release
	// file and by sheet name for a sheet task. The log records a command of
	// either as a byte range of its source, so a command whose source is
	// missing is reported without its statement.
	Sources map[string][]byte
}

// SourceKey is the Sources key of one file of a release.
func SourceKey(release, version string) string {
	return release + "/files/" + version
}

// SourceRef names a source: a release file, or a sheet task's sheet.
type SourceRef struct {
	Release string
	Version string
	Sheet   string
}

// Key is the Sources key of the source.
func (r SourceRef) Key() string {
	if r.Sheet != "" {
		return r.Sheet
	}
	return SourceKey(r.Release, r.Version)
}

// NeededSources lists each source that a latest run's log cut a command
// from, once.
func NeededSources(stage *v1pb.Stage, taskRuns []*v1pb.TaskRun, logs map[string]*v1pb.TaskRunLog) []SourceRef {
	var refs []SourceRef
	seen := map[SourceRef]bool{}
	runsByTask := GroupTaskRuns(taskRuns)
	for _, task := range stage.GetTasks() {
		runs := runsByTask[task.GetName()]
		if len(runs) == 0 {
			continue
		}
		for _, scope := range splitScopes(logs[runs[len(runs)-1].GetName()].GetEntries()) {
			ref := SourceRef{Sheet: task.GetDatabaseUpdate().GetSheet()}
			if scope.file != nil {
				ref = SourceRef{Release: task.GetDatabaseUpdate().GetRelease(), Version: scope.file.GetVersion()}
			}
			if ref == (SourceRef{}) || seen[ref] || !cutFromSource(scope.entries) {
				continue
			}
			seen[ref] = true
			refs = append(refs, ref)
		}
	}
	return refs
}

func cutFromSource(entries []*v1pb.TaskRunLogEntry) bool {
	for _, entry := range entries {
		if execute := entry.GetCommandExecute(); execute != nil && execute.GetStatement() == "" && execute.GetRange() != nil {
			return true
		}
	}
	return false
}

func Build(in Input) *Result {
	sources := make(map[string][]byte, len(in.Sources))
	for key, source := range in.Sources {
		sources[key] = executedSource(source)
	}
	in.Sources = sources

	outcome, detail := DetailStage(in.Rollout, in.TargetStage)
	result := &Result{
		Outcome:     outcome,
		URL:         in.URL + "/" + in.Rollout.GetName(),
		TargetStage: in.TargetStage,
	}
	runsByTask := GroupTaskRuns(in.TaskRuns)
	for _, stage := range in.Rollout.GetStages() {
		s := &Stage{
			Environment: stage.GetEnvironment(),
			TaskCounts:  map[string]int{},
		}
		for _, task := range stage.GetTasks() {
			s.TaskCounts[task.GetStatus().String()]++
		}
		if detail != nil && stage.GetName() == detail.GetName() {
			for _, task := range stage.GetTasks() {
				s.Tasks = append(s.Tasks, buildTask(in, task, runsByTask[task.GetName()]))
			}
		}
		result.Stages = append(result.Stages, s)
	}
	return result
}

func buildTask(in Input, task *v1pb.Task, runs []*v1pb.TaskRun) *Task {
	t := &Task{
		Name:          task.GetName(),
		Target:        task.GetTarget(),
		Status:        task.GetStatus().String(),
		URL:           in.URL + "/" + task.GetName(),
		SkippedReason: task.GetSkippedReason(),
		Skipper:       task.GetSkipper(),
		Attempt:       len(runs),
	}
	if len(runs) == 0 {
		return t
	}
	for _, run := range runs[:len(runs)-1] {
		t.PreviousRuns = append(t.PreviousRuns, &Run{
			Name:   run.GetName(),
			Status: run.GetStatus().String(),
			Detail: run.GetDetail(),
		})
	}
	latest := runs[len(runs)-1]
	t.TaskRun = latest.GetName()
	t.Detail = latest.GetDetail()
	if latest.GetStartTime() != nil {
		start := latest.GetStartTime().AsTime()
		t.StartTime = &start
	}
	if latest.GetUpdateTime() != nil {
		update := latest.GetUpdateTime().AsTime()
		t.UpdateTime = &update
	}

	if !HasLog(latest) {
		return t
	}
	log, ok := in.Logs[latest.GetName()]
	if !ok {
		t.LogUnavailable = true
		return t
	}
	runDone := latest.GetStatus() == v1pb.TaskRun_DONE
	scopes := splitScopes(log.GetEntries())
	for i, scope := range scopes {
		source := in.Sources[task.GetDatabaseUpdate().GetSheet()]
		if scope.file != nil {
			source = in.Sources[SourceKey(task.GetDatabaseUpdate().GetRelease(), scope.file.GetVersion())]
		}
		commands, scopeErr := buildCommands(scope.entries, source)
		if scope.file == nil {
			t.Commands = append(t.Commands, commands...)
			continue
		}
		file := &File{
			Version:      scope.file.GetVersion(),
			Path:         scope.file.GetFilePath(),
			Status:       v1pb.TaskRun_DONE.String(),
			Error:        scopeErr,
			Commands:     commands,
			AffectedRows: sumRows(commands),
		}
		for _, command := range commands {
			if command.RolledBack {
				file.RolledBack = true
			}
			if command.Error != "" {
				file.Status = v1pb.TaskRun_FAILED.String()
			}
		}
		if scopeErr != "" {
			file.Status = v1pb.TaskRun_FAILED.String()
		} else if file.Status == v1pb.TaskRun_DONE.String() && i == len(scopes)-1 && !runDone {
			// The executor stops at the first file that does not finish, so the
			// last file a stopped run reached shares the run's status.
			file.Status = latest.GetStatus().String()
		}
		t.Files = append(t.Files, file)
	}

	var counted []*Command
	counted = append(counted, t.Commands...)
	for _, file := range t.Files {
		counted = append(counted, file.Commands...)
	}
	t.AffectedRows = sumRows(counted)

	release := task.GetDatabaseUpdate().GetRelease() != ""
	if release && runDone && len(t.Files) == 0 {
		t.Unchanged = true
	}
	if release && !runDone {
		if notRun := int(task.GetDatabaseUpdate().GetReleaseState().GetPendingCount()) - len(t.Files); notRun > 0 {
			t.FilesNotRun = notRun
		}
	}
	return t
}

type scope struct {
	// file is nil for entries logged outside any release file.
	file    *v1pb.TaskRunLogEntry_ReleaseFileExecute
	entries []*v1pb.TaskRunLogEntry
}

// splitScopes cuts a run's log at each release file. Entries before the first
// file, or all of them for a sheet task, form a scope with no file.
func splitScopes(entries []*v1pb.TaskRunLogEntry) []scope {
	var scopes []scope
	current := scope{}
	for _, entry := range entries {
		if entry.GetType() == v1pb.TaskRunLogEntry_RELEASE_FILE_EXECUTE {
			if current.file != nil || len(current.entries) > 0 {
				scopes = append(scopes, current)
			}
			current = scope{file: entry.GetReleaseFileExecute()}
			continue
		}
		current.entries = append(current.entries, entry)
	}
	if current.file != nil || len(current.entries) > 0 {
		scopes = append(scopes, current)
	}
	return scopes
}

// buildCommands reads one scope. A RETRY_INFO entry closes an attempt the
// executor ran again: its failure is the reason for the retry, not the
// result. Such an attempt's commands are dropped unless they are known to
// have committed, as they are when the script runs outside a transaction and
// the retry starts it again from the top. It also returns the final attempt's
// first failure logged outside a command.
func buildCommands(entries []*v1pb.TaskRunLogEntry, source []byte) ([]*Command, string) {
	var earlier, commands []*Command
	var scopeErr string
	// inTransaction holds the commands since the last BEGIN that no COMMIT has
	// settled. A command run outside a transaction commits by itself.
	var open bool
	var inTransaction []*Command
	committed := map[*Command]bool{}
	for _, entry := range entries {
		switch entry.GetType() {
		case v1pb.TaskRunLogEntry_RETRY_INFO:
			for _, command := range commands {
				if committed[command] {
					command.EarlierAttempt = true
					earlier = append(earlier, command)
				}
			}
			commands, scopeErr, open, inTransaction = nil, "", false, nil
		case v1pb.TaskRunLogEntry_COMMAND_EXECUTE:
			execute := entry.GetCommandExecute()
			command := &Command{Statement: statementOf(execute, source)}
			if response := execute.GetResponse(); response != nil {
				if response.GetError() != "" {
					command.Error = response.GetError()
				} else {
					rows := response.GetAffectedRows()
					command.AffectedRows = &rows
					if len(response.GetAllAffectedRows()) > 1 {
						command.AllAffectedRows = response.GetAllAffectedRows()
					}
				}
			}
			commands = append(commands, command)
			if open {
				inTransaction = append(inTransaction, command)
			} else if command.AffectedRows != nil {
				committed[command] = true
			}
		case v1pb.TaskRunLogEntry_TRANSACTION_CONTROL:
			control := entry.GetTransactionControl()
			if control.GetError() != "" && scopeErr == "" {
				scopeErr = strings.ToLower(control.GetType().String()) + ": " + control.GetError()
			}
			switch control.GetType() {
			case v1pb.TaskRunLogEntry_TransactionControl_BEGIN:
				open, inTransaction = control.GetError() == "", nil
			case v1pb.TaskRunLogEntry_TransactionControl_COMMIT:
				// A failed commit is followed by a rollback entry, which
				// settles these commands.
				if control.GetError() == "" {
					for _, command := range inTransaction {
						committed[command] = command.AffectedRows != nil
					}
					open, inTransaction = false, nil
				}
			case v1pb.TaskRunLogEntry_TransactionControl_ROLLBACK:
				if control.GetError() == "" {
					for _, command := range inTransaction {
						command.RolledBack = true
					}
				}
				open, inTransaction = false, nil
			default:
			}
		default:
		}
	}
	return append(earlier, commands...), scopeErr
}

var (
	txnModeDirective      = regexp.MustCompile(`(?i)^\s*--\s*txn-mode\s*=\s*(on|off)\s*$`)
	txnIsolationDirective = regexp.MustCompile(`(?i)^\s*--\s*txn-isolation\s*=\s*(READ\s+UNCOMMITTED|READ\s+COMMITTED|REPEATABLE\s+READ|SERIALIZABLE)\s*$`)
)

// executedSource returns the text that logged byte ranges count from. A
// driver removes the transaction directives at the top of a file before it
// splits and logs the commands, so the ranges point into what is left. This
// must match base.ParseTransactionConfig, which the action cannot import
// without linking the SQL parsers; a test holds the two together.
func executedSource(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	var kept []string
	var removed, pastHeader bool
	for _, line := range lines {
		if !pastHeader {
			trimmed := strings.TrimSpace(line)
			switch {
			case trimmed == "":
			case !strings.HasPrefix(trimmed, "--"):
				pastHeader = true
			case txnModeDirective.MatchString(line), txnIsolationDirective.MatchString(line):
				removed = true
				continue
			default:
			}
		}
		kept = append(kept, line)
	}
	if !removed {
		return source
	}
	return []byte(strings.TrimSpace(strings.Join(kept, "\n")))
}

// statementOf returns the command's text: logged with it, or cut from its
// source by the logged byte range.
func statementOf(execute *v1pb.TaskRunLogEntry_CommandExecute, source []byte) string {
	if execute.GetStatement() != "" {
		return execute.GetStatement()
	}
	r := execute.GetRange()
	if r == nil || r.GetStart() < 0 || r.GetEnd() < r.GetStart() || int(r.GetEnd()) > len(source) {
		return ""
	}
	return strings.TrimSpace(string(source[r.GetStart():r.GetEnd()]))
}

// sumRows adds the counts of commands that were not rolled back. It returns
// nil when no command reported a count, which is not the same as zero rows.
func sumRows(commands []*Command) *int64 {
	var total int64
	var reported bool
	for _, command := range commands {
		if command.AffectedRows == nil {
			continue
		}
		reported = true
		if !command.RolledBack {
			total += *command.AffectedRows
		}
	}
	if !reported {
		return nil
	}
	return &total
}
