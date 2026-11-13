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

	// AffectedRows is nil when no command reported a count or a commit's
	// outcome is unknown.
	AffectedRows *int64 `json:"affectedRows,omitempty"`
	// Unchanged is a release task that finished with nothing to apply: every
	// version was already applied, or the schema already matched.
	Unchanged bool `json:"unchanged,omitempty"`
	// FilesNotRun counts the release's files a stopped run never reached.
	FilesNotRun int     `json:"filesNotRun,omitempty"`
	Files       []*File `json:"files,omitempty"`
	// Commands holds what ran outside any release file, as a sheet task's
	// does, and CommandCount how many of them ran. See File.
	Commands     []*Command `json:"commands,omitempty"`
	CommandCount int        `json:"commandCount,omitempty"`
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
	// AffectedRows leaves out rolled-back commands and is nil when a commit's
	// outcome is unknown.
	AffectedRows *int64 `json:"affectedRows,omitempty"`
	// Error is a failure outside any command, such as a commit that failed.
	Error string `json:"error,omitempty"`
	// CommandCount is how many commands ran. Commands holds the first
	// KeptCommands of them and any that failed, so a file of many statements
	// cannot make the result unbounded; the counts cover all of them.
	CommandCount int        `json:"commandCount,omitempty"`
	Commands     []*Command `json:"commands,omitempty"`
}

type Command struct {
	// Index is the command's one-based position in its file or sheet attempt.
	Index     int    `json:"index"`
	Statement string `json:"statement"`
	// AffectedRows is nil when the command failed or logged no response.
	AffectedRows *int64 `json:"affectedRows,omitempty"`
	// AllAffectedRows is set when the command held several statements and the
	// driver counted each.
	AllAffectedRows []int64 `json:"allAffectedRows,omitempty"`
	RolledBack      bool    `json:"rolledBack,omitempty"`
	// CommitUnknown means the log does not establish whether the transaction
	// committed. The command keeps its count, but the file and task have no total.
	CommitUnknown bool `json:"commitUnknown,omitempty"`
	// EarlierAttempt marks a command from an attempt the executor retried.
	// It is kept when it committed or its commit outcome is unknown.
	EarlierAttempt bool   `json:"earlierAttempt,omitempty"`
	Error          string `json:"error,omitempty"`

	// byteRange is where the log says the statement is in its source, kept
	// until the source is read.
	byteRange *v1pb.Range
}

// KeptCommands is how many commands of one file the result keeps, besides any
// that failed.
const KeptCommands = 100

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
	Logs map[string]*LogDigest
	// Sources hold what the commands ran, keyed by SourceKey for a release
	// file and by sheet name for a sheet task. The log records a command of
	// either as a byte range of its source, so a command whose source is
	// missing is reported without its statement.
	Sources map[string][]byte
	// Releases are keyed by name. They give the order of a release's files,
	// which says how many a stopped run never reached.
	Releases map[string]*v1pb.Release
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

// NeededSources lists each source that a kept command of a latest run was cut
// from, once.
func NeededSources(stage *v1pb.Stage, taskRuns []*v1pb.TaskRun, logs map[string]*LogDigest) []SourceRef {
	var refs []SourceRef
	seen := map[SourceRef]bool{}
	runsByTask := GroupTaskRuns(taskRuns)
	for _, task := range stage.GetTasks() {
		runs := runsByTask[task.GetName()]
		if len(runs) == 0 {
			continue
		}
		for _, scope := range logs[runs[len(runs)-1].GetName()].getScopes() {
			ref := SourceRef{Sheet: task.GetDatabaseUpdate().GetSheet()}
			if scope.file != nil {
				ref = SourceRef{Release: task.GetDatabaseUpdate().GetRelease(), Version: scope.file.GetVersion()}
			}
			if ref == (SourceRef{}) || seen[ref] || !scope.cutFromSource() {
				continue
			}
			seen[ref] = true
			refs = append(refs, ref)
		}
	}
	return refs
}

func Build(in Input) *Result {
	// Each source is stripped once and held as a string, so that the
	// statements cut from it for every database share its memory.
	sources := make(map[string]sourceText, len(in.Sources))
	for key, source := range in.Sources {
		stored := string(source)
		sources[key] = sourceText{stored: stored, executed: executedSource(stored)}
	}

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
				s.Tasks = append(s.Tasks, buildTask(in, sources, task, runsByTask[task.GetName()]))
			}
		}
		result.Stages = append(result.Stages, s)
	}
	return result
}

func buildTask(in Input, sources map[string]sourceText, task *v1pb.Task, runs []*v1pb.TaskRun) *Task {
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
	scopes := log.getScopes()
	// applied turns true once a file may have changed the database.
	var applied bool
	var rowsUnknown bool
	var rows []*int64
	for i, scope := range scopes {
		source := sources[task.GetDatabaseUpdate().GetSheet()]
		if scope.file != nil {
			source = sources[SourceKey(task.GetDatabaseUpdate().GetRelease(), scope.file.GetVersion())]
		}
		scope.readStatements(source)
		rows = append(rows, scope.rows)
		rowsUnknown = rowsUnknown || scope.rowsUnknown
		if scope.file == nil {
			t.Commands = append(t.Commands, scope.commands...)
			t.CommandCount += scope.commandCount
			continue
		}
		// A declarative file computes a diff first, and runs nothing when the
		// schema already matches.
		if scope.commandCount > 0 || !scope.computedDiff {
			applied = true
		}
		file := &File{
			Version:      scope.file.GetVersion(),
			Path:         scope.file.GetFilePath(),
			Status:       v1pb.TaskRun_DONE.String(),
			RolledBack:   scope.rolledBack,
			AffectedRows: scope.rows,
			Error:        scope.err,
			CommandCount: scope.commandCount,
			Commands:     scope.commands,
		}
		if scope.failed || scope.err != "" {
			file.Status = v1pb.TaskRun_FAILED.String()
		} else if i == len(scopes)-1 && !runDone {
			// The executor stops at the first file that does not finish, so the
			// last file a stopped run reached shares the run's status.
			file.Status = latest.GetStatus().String()
		}
		t.Files = append(t.Files, file)
	}
	if !rowsUnknown {
		t.AffectedRows = addRows(rows)
	}

	release := task.GetDatabaseUpdate().GetRelease()
	if release != "" && runDone && !applied {
		t.Unchanged = true
	}
	if release != "" && !runDone {
		t.FilesNotRun = filesNotRun(task.GetDatabaseUpdate().GetReleaseState(), in.Releases[release], t.Files, len(runs))
	}
	return t
}

// filesNotRun counts the files a stopped run never reached: the pending ones
// after the last file it ran, in the release's order. The task's release
// state is a snapshot from when the rollout was created, and a later run
// skips what an earlier one applied, so its pending count is only right for a
// first run. Without the release the count is known for a first run alone.
func filesNotRun(state *v1pb.Task_ReleaseState, release *v1pb.Release, files []*File, runs int) int {
	position := -1
	if len(files) > 0 {
		last := files[len(files)-1].Version
		position = slices.IndexFunc(release.GetFiles(), func(file *v1pb.Release_File) bool {
			return file.GetVersion() == last
		})
	}
	if position < 0 {
		if runs > 1 {
			return 0
		}
		return max(int(state.GetPendingCount())-len(files), 0)
	}
	var count int
	for _, pending := range state.GetPendingRanges() {
		count += max(int(pending.GetEnd())-max(int(pending.GetStart()), position+1), 0)
	}
	return count
}

// LogDigest is a task run's log reduced to what the result keeps. A log is
// reduced as soon as it is read and then dropped, so that the action holds
// one log's worth of entries per reader, not every database's.
type LogDigest struct {
	scopes []*scopeDigest
}

// DigestLog reduces a task run's log.
func DigestLog(log *v1pb.TaskRunLog) *LogDigest {
	digest := &LogDigest{}
	for _, scope := range splitScopes(log.GetEntries()) {
		commands, scopeErr := buildCommands(scope.entries)
		d := &scopeDigest{
			file:         scope.file,
			commandCount: len(commands),
			rows:         sumRows(commands),
			err:          scopeErr,
			computedDiff: computedDiff(scope.entries),
		}
		for _, command := range commands {
			d.failed = d.failed || command.Error != ""
			d.rolledBack = d.rolledBack || command.RolledBack
			d.rowsUnknown = d.rowsUnknown || command.CommitUnknown
			d.maxRangeEnd = max(d.maxRangeEnd, command.byteRange.GetEnd())
		}
		d.commands = commands
		if len(commands) > KeptCommands {
			// Cloned, so that the commands left out can be freed.
			d.commands = slices.Clone(commands[:KeptCommands])
			for _, command := range commands[KeptCommands:] {
				if command.Error != "" {
					d.commands = append(d.commands, command)
				}
			}
		}
		digest.scopes = append(digest.scopes, d)
	}
	return digest
}

func (d *LogDigest) getScopes() []*scopeDigest {
	if d == nil {
		return nil
	}
	return d.scopes
}

// scopeDigest is one release file's part of a log, or the part outside any
// file. Everything but commands covers every command that ran.
type scopeDigest struct {
	file         *v1pb.TaskRunLogEntry_ReleaseFileExecute
	commands     []*Command
	commandCount int
	rows         *int64
	rowsUnknown  bool
	failed       bool
	rolledBack   bool
	err          string
	computedDiff bool
	// maxRangeEnd is the furthest any command reaches into its source.
	maxRangeEnd int32
}

func (d *scopeDigest) cutFromSource() bool {
	return slices.ContainsFunc(d.commands, func(command *Command) bool {
		return command.Statement == "" && command.byteRange != nil
	})
}

// readStatements cuts each kept command's statement from its source, unless
// the log carried the statement or the range falls outside the source.
func (d *scopeDigest) readStatements(source sourceText) {
	text := source.forRanges(d.maxRangeEnd)
	for _, command := range d.commands {
		r := command.byteRange
		if command.Statement != "" || r == nil || r.GetStart() < 0 || r.GetEnd() < r.GetStart() || int(r.GetEnd()) > len(text) {
			continue
		}
		command.Statement = strings.TrimSpace(text[r.GetStart():r.GetEnd()])
	}
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
// result. Such an attempt's commands are kept if they committed or their commit
// outcome is unknown. It also returns the final attempt's first failure logged
// outside a command.
func buildCommands(entries []*v1pb.TaskRunLogEntry) ([]*Command, string) {
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
				if committed[command] || command.CommitUnknown {
					command.EarlierAttempt = true
					earlier = append(earlier, command)
				}
			}
			commands, scopeErr, open, inTransaction = nil, "", false, nil
		case v1pb.TaskRunLogEntry_COMMAND_EXECUTE:
			execute := entry.GetCommandExecute()
			command := &Command{Index: len(commands) + 1, Statement: execute.GetStatement(), byteRange: execute.GetRange()}
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
				if control.GetError() == "" {
					for _, command := range inTransaction {
						committed[command] = command.AffectedRows != nil
					}
					open, inTransaction = false, nil
				} else {
					rejected := control.GetError() == "commit unexpectedly resulted in rollback" || rejectedCommit.MatchString(control.GetError())
					for _, command := range inTransaction {
						command.RolledBack = rejected
						command.CommitUnknown = !rejected
					}
				}
			case v1pb.TaskRunLogEntry_TransactionControl_ROLLBACK:
				if control.GetError() == "" {
					for _, command := range inTransaction {
						// A rollback after a lost COMMIT response may be a no-op.
						if !command.CommitUnknown {
							command.RolledBack = true
						}
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
	// PostgreSQL constraint and transaction-rollback errors reject COMMIT;
	// 40003 (statement_completion_unknown) deliberately does not match.
	rejectedCommit = regexp.MustCompile(`\(SQLSTATE (23[A-Z0-9]{3}|4000[012]|40P01)\)$`)
)

// sourceText is a file as stored and as a driver that honors transaction
// directives executes it.
type sourceText struct {
	stored   string
	executed string
}

// forRanges leaves the text unavailable when the ranges fit both the stored
// and stripped sources. The API does not identify which one the driver used.
func (s sourceText) forRanges(maxRangeEnd int32) string {
	if s.stored == s.executed || int(maxRangeEnd) > len(s.executed) {
		return s.stored
	}
	return ""
}

func computedDiff(entries []*v1pb.TaskRunLogEntry) bool {
	return slices.ContainsFunc(entries, func(entry *v1pb.TaskRunLogEntry) bool {
		return entry.GetType() == v1pb.TaskRunLogEntry_COMPUTE_DIFF
	})
}

// executedSource returns the file as a driver that honors transaction
// directives executes it: without the directive lines at its top. This must
// match base.ParseTransactionConfig, which the action cannot import without
// linking the SQL parsers; a test holds the two together.
func executedSource(source string) string {
	lines := strings.Split(source, "\n")
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
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// sumRows adds the counts of commands that were not rolled back. It returns
// nil when no command reported a count or a commit's outcome is unknown.
func sumRows(commands []*Command) *int64 {
	var total int64
	var reported bool
	for _, command := range commands {
		if command.CommitUnknown {
			return nil
		}
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

// addRows adds the totals that were reported. It returns nil when none was.
func addRows(totals []*int64) *int64 {
	var total int64
	var reported bool
	for _, rows := range totals {
		if rows != nil {
			reported = true
			total += *rows
		}
	}
	if !reported {
		return nil
	}
	return &total
}
