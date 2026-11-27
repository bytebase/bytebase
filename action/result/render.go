package result

import (
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// shownCommands is how many row-changing commands a file lists in the log
	// and the summary. The output file lists all of them.
	shownCommands = 10
	// shownLines and shownLineWidth cap a failed statement and an error in the
	// log and the summary, by line count and by line length, so that neither a
	// long script nor a statement written on one line can crowd out the rest.
	shownLines     = 30
	shownLineWidth = 200
	// shortenedNote follows a statement or an error that was cut.
	shortenedNote = "(shortened; the full text is in the output file and in Bytebase)"
	// markdownBudget keeps a GitHub step summary under its 1 MiB limit, past
	// which GitHub drops the whole summary.
	markdownBudget = 900 << 10

	// The summary shows at most shownFailures failures and shownTableRows
	// databases. The log has no size limit, so it lists every one.
	shownFailures  = 10
	shownTableRows = 50
	// These bound the scripts, in the summary and in the log, so the two read
	// alike and a large rollout does not flood the log.
	shownScripts          = 5
	shownScriptStatements = 50
	namedDatabases        = 5
	listedDatabases       = 50
	// openScriptLines is the longest script shown unfolded.
	openScriptLines = 12
	// scriptShortenedNote follows a script with a statement that was cut. A
	// script's statement can be past the first KeptCommands of its file, so
	// only Bytebase is sure to hold its full text.
	scriptShortenedNote = "(shortened; the full text is in Bytebase)"
)

// Folder wraps one script's lines so that a CI can collapse them. The index
// is unique within one rendering.
type Folder func(index int, header string, detail []string) []string

// Headline is the one line that says how the rollout stands. A finished
// stage says "Deployed" only when a database changed.
func (r *Result) Headline() string {
	switch r.Outcome {
	case NotWaited:
		return "Rollout created. This job deployed nothing: no --target-stage was given."
	case StageNotFound:
		return fmt.Sprintf("This job deployed nothing: the rollout has no stage for %s.", r.TargetStage)
	case Interrupted:
		return fmt.Sprintf("Rollout to %s: INTERRUPTED. The action stopped before the stage finished, so this state is not final.", r.stageName())
	case Failed:
		return fmt.Sprintf("Deploy to %s failed on %d of %s", r.stageName(), r.countStatus("FAILED"), plural(r.taskCount(), "database"))
	case Canceled:
		return fmt.Sprintf("Deploy to %s canceled on %d of %s", r.stageName(), r.countStatus("CANCELED"), plural(r.taskCount(), "database"))
	default:
		if r.changedCount() > 0 {
			return "Deployed to " + r.stageName()
		}
		return "No changes to " + r.stageName()
	}
}

// mark is the sign the summary puts before the headline.
func (r *Result) mark() string {
	switch r.Outcome {
	case Done:
		return "✅ "
	case Failed:
		return "❌ "
	case Canceled:
		return "⛔ "
	default:
		return ""
	}
}

func (r *Result) countStatus(status string) int {
	var n int
	if stage := r.detail(); stage != nil {
		for _, task := range stage.Tasks {
			if task.Status == status {
				n++
			}
		}
	}
	return n
}

func (r *Result) changedCount() int {
	var n int
	if stage := r.detail(); stage != nil {
		for _, task := range stage.Tasks {
			if task.changed() {
				n++
			}
		}
	}
	return n
}

// changed reports whether the task finished and applied something.
func (t *Task) changed() bool {
	return t.Status == "DONE" && !t.Unchanged
}

func (r *Result) taskCount() int {
	if stage := r.detail(); stage != nil {
		return len(stage.Tasks)
	}
	return 0
}

// Failures returns one line per failed or canceled task, failed first.
func (r *Result) Failures() []string {
	var lines []string
	for _, task := range r.orderedTasks() {
		switch task.Status {
		case "FAILED":
			file, _, errText := task.failure()
			line := task.Target + " failed"
			if file != nil {
				line += " at " + file.name()
			}
			if errText != "" {
				line += ": " + oneLine(errText, 500)
			}
			lines = append(lines, line)
		case "CANCELED":
			lines = append(lines, task.Target+" was canceled")
		default:
		}
	}
	return lines
}

// Warnings returns what a passing job should still draw attention to: a
// target stage the rollout does not have.
func (r *Result) Warnings() []string {
	if r.Outcome != StageNotFound {
		return nil
	}
	line := fmt.Sprintf("the rollout has no stage for %s, so this job deployed nothing", r.TargetStage)
	if len(r.Stages) == 0 {
		return []string{line + "; it has no stages"}
	}
	environments := make([]string, 0, len(r.Stages))
	for _, stage := range r.Stages {
		environments = append(environments, stage.Environment)
	}
	return []string{line + "; its stages are " + strings.Join(environments, ", ")}
}

// RenderText renders the result for a job log. fold may be nil. The answer
// comes first and the rollout's URL last, since most readers never need it.
func RenderText(r *Result, fold Folder) string {
	w := r.whatRan()
	lines := []string{r.Headline()}
	if counts := r.countsLine(w); counts != "" {
		lines = append(lines, counts)
	}
	lines = append(lines, r.otherStages()...)
	lines = append(lines, w.notes("in Bytebase")...)

	tasks := r.orderedTasks()
	if len(tasks) > 0 {
		lines = append(lines, "")
	}
	letters := w.letters()
	var statusWidth, targetWidth int
	for _, task := range tasks {
		statusWidth = max(statusWidth, len(task.Status))
		targetWidth = max(targetWidth, utf8.RuneCountInString(task.Target))
	}
	indent := strings.Repeat(" ", statusWidth+2)
	for _, task := range tasks {
		note := task.note()
		if letter := letters[task]; letter != "" {
			note = "script " + letter + " · " + note
		}
		lines = append(lines, strings.TrimRight(fmt.Sprintf("%-*s  %-*s  %s", statusWidth, task.Status, targetWidth, task.Target, note), " "))
		// What a finished database ran is in its script.
		if task.Status == "DONE" {
			continue
		}
		for _, line := range task.detailLines() {
			lines = append(lines, indent+line)
		}
		// Retrying or skipping happens in Bytebase, so the task's URL sits
		// with the failure it follows from.
		if task.stopped() && task.URL != "" {
			lines = append(lines, indent+"Retry or skip: "+task.URL)
		}
	}
	if len(tasks) > 0 {
		lines = append(lines, "")
	}
	if scripts := w.textLines(fold, "in Bytebase"); len(scripts) > 0 {
		lines = append(lines, scripts...)
		lines = append(lines, "")
	}
	lines = append(lines, "Rollout: "+r.URL)
	return strings.Join(lines, "\n") + "\n"
}

// RenderMarkdown renders the result for a GitHub step summary: the headline,
// the counts with the rollout's link, notes, failures, the scripts and a
// folded table of databases. Every line that says what was left out links
// the rollout too.
func RenderMarkdown(r *Result) string {
	w := r.whatRan()
	inBytebase := "[in Bytebase](" + r.URL + ")"
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "### %s%s\n\n", r.mark(), r.Headline())
	open := "[Open the rollout in Bytebase](" + r.URL + ")"
	if counts := r.countsLine(w); counts != "" {
		_, _ = fmt.Fprintf(&sb, "%s · %s\n\n", counts, open)
	} else {
		_, _ = sb.WriteString(open + "\n\n")
	}
	for _, line := range r.otherStages() {
		_, _ = sb.WriteString(line + "\n\n")
	}
	if notes := w.notes(inBytebase); len(notes) > 0 {
		_, _ = sb.WriteString(strings.Join(notes, " ") + "\n\n")
	}
	tasks := r.orderedTasks()
	if len(tasks) == 0 {
		return sb.String()
	}

	var shown, omitted int
	for _, task := range tasks {
		if !task.stopped() {
			continue
		}
		block := task.failureMarkdown()
		if shown == shownFailures || sb.Len()+len(block) > markdownBudget/2 {
			omitted++
			continue
		}
		_, _ = sb.WriteString(block)
		shown++
	}
	if omitted > 0 {
		_, _ = fmt.Fprintf(&sb, "and %s, %s.\n\n", more(omitted, "failed database"), inBytebase)
	}

	table := databaseTable(tasks, w.letters(), inBytebase)
	_, _ = sb.WriteString(w.markdown(markdownBudget-sb.Len()-len(table), inBytebase, markdownNames(tasks)))
	_, _ = sb.WriteString(table)
	return sb.String()
}

func (r *Result) detail() *Stage {
	for _, stage := range r.Stages {
		if len(stage.Tasks) > 0 {
			return stage
		}
	}
	return nil
}

func (r *Result) stageName() string {
	if stage := r.detail(); stage != nil {
		return strings.TrimPrefix(stage.Environment, "environments/")
	}
	return strings.TrimPrefix(r.TargetStage, "environments/")
}

// orderedTasks returns the detailed stage's tasks with failed and canceled
// ones first.
func (r *Result) orderedTasks() []*Task {
	stage := r.detail()
	if stage == nil {
		return nil
	}
	rank := func(status string) int {
		switch status {
		case "FAILED":
			return 0
		case "CANCELED":
			return 1
		default:
			return 2
		}
	}
	tasks := slices.Clone(stage.Tasks)
	slices.SortStableFunc(tasks, func(a, b *Task) int {
		return rank(a.Status) - rank(b.Status)
	})
	return tasks
}

// countsLine counts the detailed stage's databases by what happened to them,
// and says when they ran different scripts. It is empty when no stage is
// detailed.
func (r *Result) countsLine(w *whatRan) string {
	detail := r.detail()
	if detail == nil {
		return ""
	}
	counts := map[string]int{}
	for _, task := range detail.Tasks {
		switch {
		case task.changed():
			counts["changed"]++
		case task.Status == "DONE":
			counts["already up to date"]++
		default:
			counts[statusWord(task.Status)]++
		}
	}
	line := joinCounts(counts, []string{"changed", "already up to date", "failed", "canceled", "skipped", "running", "pending", "not started"})
	if len(w.scripts) > 1 {
		line += " · " + plural(len(w.scripts), "different script")
	}
	return line
}

// otherStages counts the other stages' tasks by status, in one line.
func (r *Result) otherStages() []string {
	var lines []string
	detail := r.detail()
	var others []string
	for _, stage := range r.Stages {
		if stage == detail {
			continue
		}
		counts := map[string]int{}
		for status, n := range stage.TaskCounts {
			counts[statusWord(status)] += n
		}
		text := joinCounts(counts, []string{"done", "failed", "canceled", "skipped", "running", "pending", "not started"})
		if text == "" {
			text = "no tasks"
		}
		others = append(others, fmt.Sprintf("%s (%s)", strings.TrimPrefix(stage.Environment, "environments/"), text))
	}
	if len(others) > 0 {
		label := "Other stages: "
		if detail == nil {
			label = "Stages: "
		}
		lines = append(lines, label+strings.Join(others, ", "))
	}
	return lines
}

func statusWord(status string) string {
	return strings.ToLower(strings.ReplaceAll(status, "_", " "))
}

func joinCounts(counts map[string]int, order []string) string {
	var parts []string
	seen := map[string]bool{}
	for _, key := range order {
		seen[key] = true
		if counts[key] > 0 {
			parts = append(parts, formatCount(int64(counts[key]))+" "+key)
		}
	}
	var rest []string
	for key := range counts {
		if !seen[key] {
			rest = append(rest, key)
		}
	}
	slices.Sort(rest)
	for _, key := range rest {
		parts = append(parts, formatCount(int64(counts[key]))+" "+key)
	}
	return strings.Join(parts, " · ")
}

// note is what follows the database on a task's line.
func (t *Task) note() string {
	var parts []string
	if t.Attempt > 1 {
		parts = append(parts, fmt.Sprintf("attempt %d", t.Attempt))
	}
	switch t.Status {
	case "SKIPPED":
		note := ""
		if t.Skipper != "" {
			note = "by " + strings.TrimPrefix(t.Skipper, "users/")
		}
		if t.SkippedReason != "" {
			if note != "" {
				note += ": "
			}
			note += oneLine(t.SkippedReason, 200)
		}
		if note != "" {
			parts = append(parts, note)
		}
	case "DONE":
		switch {
		case t.LogUnavailable:
			parts = append(parts, "log not available")
		case t.Unchanged:
			parts = append(parts, "already up to date")
		default:
			if len(t.Files) > 0 {
				parts = append(parts, plural(len(t.Files), "file"))
			}
			if n := t.commandCount(); n > 0 {
				parts = append(parts, plural(n, "statement"))
			}
			parts = append(parts, rowsText(t.AffectedRows))
			if d := t.duration(); d != "" {
				parts = append(parts, d)
			}
		}
	case "FAILED", "CANCELED":
		if t.LogUnavailable {
			parts = append(parts, "log not available")
		}
		if t.FilesNotRun > 0 {
			parts = append(parts, plural(t.FilesNotRun, "file")+" not run")
		}
	default:
	}
	return strings.Join(parts, " · ")
}

// detailLines are the lines under a task: its files, the commands that
// changed rows, and for a failure the statement and the error.
func (t *Task) detailLines() []string {
	var lines []string
	var errorShown bool
	for _, file := range t.Files {
		lines = append(lines, strings.TrimRight(file.name()+"  "+file.note(), " "))
		if file.Status == "DONE" {
			lines = append(lines, indented(commandLines(file.Commands, file.CommandCount, file.AffectedRows))...)
			continue
		}
		failed := failureLines(file.Commands, file.Error)
		errorShown = errorShown || len(failed) > 0
		lines = append(lines, indented(failed)...)
	}
	if failed := failureLines(t.Commands, ""); len(failed) > 0 {
		errorShown = true
		lines = append(lines, failed...)
	} else {
		var rows *int64
		if len(t.Files) == 0 {
			rows = t.AffectedRows
		}
		lines = append(lines, commandLines(t.Commands, t.CommandCount, rows)...)
	}
	if !errorShown && t.Detail != "" && t.stopped() {
		detail, cut := shorten(t.Detail, textCut)
		lines = append(lines, detail...)
		if cut {
			lines = append(lines, shortenedNote)
		}
	}
	return lines
}

// failure returns the file, statement and error a failed task stopped on.
// The file is nil when the failure was outside any release file.
func (t *Task) failure() (*File, string, string) {
	for _, file := range t.Files {
		if file.Status != "FAILED" {
			continue
		}
		for _, command := range file.Commands {
			if command.Error != "" {
				return file, command.Statement, command.Error
			}
		}
		if file.Error != "" {
			return file, "", file.Error
		}
		return file, "", t.Detail
	}
	for _, command := range t.Commands {
		if command.Error != "" {
			return nil, command.Statement, command.Error
		}
	}
	return nil, "", t.Detail
}

// commandCount is how many commands ran, which can be more than the result
// holds.
func (t *Task) commandCount() int {
	n := max(t.CommandCount, len(t.Commands))
	for _, file := range t.Files {
		n += max(file.CommandCount, len(file.Commands))
	}
	return n
}

func (t *Task) duration() string {
	if t.StartTime == nil || t.UpdateTime == nil || t.UpdateTime.Before(*t.StartTime) {
		return ""
	}
	d := t.UpdateTime.Sub(*t.StartTime)
	if d < time.Minute {
		return strconv.FormatFloat(d.Seconds(), 'f', 1, 64) + "s"
	}
	return d.Round(time.Second).String()
}

// stopped reports whether the task needs someone to retry or skip it.
func (t *Task) stopped() bool {
	return t.Status == "FAILED" || t.Status == "CANCELED"
}

// instanceAndDatabase splits the task's target. The database is empty for a
// task that creates one.
func (t *Task) instanceAndDatabase() (string, string) {
	before, database, _ := strings.Cut(t.Target, "/databases/")
	instance := before
	if i := strings.LastIndex(before, "instances/"); i >= 0 {
		instance = before[i+len("instances/"):]
	}
	return instance, database
}

// databaseTable is the folded table of the stage's databases, failed first.
// The Instance column shows only when the stage spans more than one
// instance, and Rows only when a database changed rows.
func databaseTable(tasks []*Task, letters map[*Task]string, inBytebase string) string {
	instances := map[string]bool{}
	var anyRows bool
	for _, task := range tasks {
		instance, _ := task.instanceAndDatabase()
		instances[instance] = true
		anyRows = anyRows || (task.AffectedRows != nil && *task.AffectedRows > 0)
	}
	multiInstance := len(instances) > 1
	head := []string{"Database"}
	if multiInstance {
		head = append(head, "Instance")
	}
	head = append(head, "Status", "Script")
	if anyRows {
		head = append(head, "Rows")
	}
	head = append(head, "Duration")
	cell := func(s string) string {
		return strings.ReplaceAll(s, "|", "\\|")
	}
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "| %s |\n|%s\n", strings.Join(head, " | "), strings.Repeat("---|", len(head)))
	for _, task := range tasks[:min(len(tasks), shownTableRows)] {
		instance, database := task.instanceAndDatabase()
		if database == "" {
			database = instance
		}
		row := []string{fmt.Sprintf("[%s](%s)", cell(database), task.URL)}
		if multiInstance {
			row = append(row, cell(instance))
		}
		script := letters[task]
		if script == "" {
			script = "–"
		}
		row = append(row, task.statusCell(), script)
		if anyRows {
			rows := ""
			if task.AffectedRows != nil {
				rows = formatCount(*task.AffectedRows)
			}
			row = append(row, rows)
		}
		row = append(row, task.duration())
		_, _ = fmt.Fprintf(&sb, "| %s |\n", strings.Join(row, " | "))
	}
	if rest := len(tasks) - shownTableRows; rest > 0 {
		_, _ = fmt.Fprintf(&sb, "\nand %s, %s.\n", more(rest, "database"), inBytebase)
	}
	title := "All " + plural(len(tasks), "database")
	if len(tasks) == 1 {
		title = "Database"
	}
	return fmt.Sprintf("<details><summary>%s</summary>\n\n%s\n</details>\n\n", title, sb.String())
}

func (t *Task) statusCell() string {
	var status string
	switch {
	case t.changed():
		status = "✅ Changed"
	case t.Status == "DONE":
		status = "Up to date"
	case t.Status == "FAILED":
		status = "❌ Failed"
	case t.Status == "CANCELED":
		status = "⛔ Canceled"
	case t.Status == "SKIPPED":
		status = "⏭️ Skipped"
	default:
		status = statusWord(t.Status)
		status = strings.ToUpper(status[:1]) + status[1:]
	}
	if t.Attempt > 1 {
		status += fmt.Sprintf(", attempt %d", t.Attempt)
	}
	return status
}

// failureMarkdown names the database and labels its instance and file, so
// the reader never has to work out which name is which.
func (t *Task) failureMarkdown() string {
	var sb strings.Builder
	file, statement, errText := t.failure()
	instance, database := t.instanceAndDatabase()
	mark, verb := "❌", "failed"
	if t.Status == "CANCELED" {
		mark, verb = "⛔", "was canceled"
	}
	var facts []string
	if database != "" {
		_, _ = fmt.Fprintf(&sb, "#### %s Database %s %s\n\n", mark, html.EscapeString(database), verb)
		facts = append(facts, "Instance: `"+instance+"`")
	} else {
		_, _ = fmt.Fprintf(&sb, "#### %s Instance %s %s\n\n", mark, html.EscapeString(instance), verb)
	}
	if file != nil {
		fact := "File: `" + file.name() + "`"
		if file.RolledBack {
			fact += " (rolled back)"
		}
		facts = append(facts, fact)
	}
	if len(facts) > 0 {
		_, _ = sb.WriteString(strings.Join(facts, "<br>\n") + "\n\n")
	}
	if applied := t.appliedBeforeFailure(); applied != "" {
		_, _ = sb.WriteString(applied + "\n\n")
	}
	statementLines, statementCut := shorten(statement, sqlCut)
	if len(statementLines) > 0 {
		_, _ = sb.WriteString(fenced("sql", strings.Join(statementLines, "\n")))
	}
	errorLines, errorCut := shorten(errText, textCut)
	if len(errorLines) > 0 {
		_, _ = sb.WriteString(fenced("text", strings.Join(errorLines, "\n")))
	}
	if statementCut || errorCut {
		_, _ = sb.WriteString(shortenedNote + "\n\n")
	}
	if t.URL != "" {
		_, _ = fmt.Fprintf(&sb, "[Retry or skip this task in Bytebase](%s)\n\n", t.URL)
	}
	return sb.String()
}

// appliedBeforeFailure says which files the task finished before it
// stopped.
func (t *Task) appliedBeforeFailure() string {
	var done []string
	var rows []*int64
	for _, file := range t.Files {
		if file.Status != "DONE" {
			continue
		}
		done = append(done, "`"+file.name()+"`")
		rows = append(rows, file.AffectedRows)
	}
	if len(done) == 0 {
		return ""
	}
	applied := strings.Join(done, ", ")
	if total := sumAll(rows); total != nil {
		applied += " (" + rowsText(total) + ")"
	}
	return "Before it stopped, it applied " + applied + "."
}

func (f *File) name() string {
	switch {
	case f.Path != "":
		return f.Path
	case f.schema != "":
		return f.schema
	default:
		return "version " + f.Version
	}
}

func (f *File) note() string {
	var note string
	if f.Status == "DONE" {
		if f.AffectedRows != nil {
			note = rowsText(f.AffectedRows)
		}
	} else {
		note = statusWord(f.Status)
	}
	if f.RolledBack {
		if note != "" {
			note += ", "
		}
		note += "rolled back"
	}
	return note
}

// commandLines lists the first commands that changed rows, counts first so
// that they line up, then the rows changed by the commands it did not list.
// commandCount and total cover every command that ran, which can be more than
// commands holds.
func commandLines(commands []*Command, commandCount int, total *int64) []string {
	type numbered struct {
		*Command
		label string
	}
	var shown []numbered
	for _, command := range commands {
		if len(shown) == shownCommands {
			break
		}
		if command.AffectedRows != nil && *command.AffectedRows > 0 {
			shown = append(shown, numbered{Command: command, label: commandLabel(command)})
		}
	}
	var width int
	for _, command := range shown {
		width = max(width, len(rowsText(command.AffectedRows)))
	}
	var lines []string
	var listed int64
	for _, command := range shown {
		line := fmt.Sprintf("%*s  %s", width, rowsText(command.AffectedRows), command.label)
		if command.CommitUnknown {
			line += "  (commit outcome unknown)"
		} else if command.RolledBack {
			line += "  (rolled back)"
		} else {
			listed += *command.AffectedRows
		}
		if command.EarlierAttempt {
			line += "  (earlier attempt)"
		}
		lines = append(lines, line)
	}
	if total != nil {
		if rest := *total - listed; rest > 0 {
			unlisted := max(commandCount, len(commands)) - len(shown)
			lines = append(lines, fmt.Sprintf("and %s in %s", rowsText(&rest), more(unlisted, "statement")))
		}
	}
	return lines
}

// failureLines are the failed statement and its error, or nil when the
// commands carry no error. Both are shortened; the error is always kept.
func failureLines(commands []*Command, scopeErr string) []string {
	for _, command := range commands {
		if command.Error == "" {
			continue
		}
		lines, statementCut := shorten(command.Statement, sqlCut)
		if len(lines) == 0 {
			lines = []string{commandLabel(command)}
		}
		errorLines, errorCut := shorten(command.Error, textCut)
		lines = append(lines, errorLines...)
		if statementCut || errorCut {
			lines = append(lines, shortenedNote)
		}
		return lines
	}
	lines, cut := shorten(scopeErr, textCut)
	if cut {
		lines = append(lines, shortenedNote)
	}
	return lines
}

// commandLabel is the command's statement on one line, or its position when
// the statement's text could not be read.
func commandLabel(command *Command) string {
	if strings.TrimSpace(command.Statement) == "" {
		return fmt.Sprintf("statement %d", command.Index)
	}
	return oneLine(command.Statement, 100)
}

// Markers that stand for the lines shorten leaves out: a comment inside SQL,
// and an ellipsis inside other text.
const (
	sqlCut  = "-- "
	textCut = "... "
)

// shorten returns the text's first lines, each cut to shownLineWidth, and
// whether anything was left out. The lines left out are counted after
// marker.
func shorten(text, marker string) ([]string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	lines := strings.Split(text, "\n")
	var cut bool
	if len(lines) > shownLines {
		rest := len(lines) - shownLines
		lines = append(lines[:shownLines:shownLines], marker+more(rest, "line"))
		cut = true
	}
	for i, line := range lines {
		if short, ok := cutRunes(line, shownLineWidth); ok {
			lines[i] = short + "..."
			cut = true
		}
	}
	return lines, cut
}

// cutRunes returns the first limit characters of s and whether s was longer.
func cutRunes(s string, limit int) (string, bool) {
	var n int
	for i := range s {
		if n == limit {
			return s[:i], true
		}
		n++
	}
	return s, false
}

// indented indents each line by two spaces.
func indented(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, "  "+line)
	}
	return out
}

// fenced wraps text in a code fence longer than any backtick run inside it.
func fenced(language, text string) string {
	fence := "```"
	for strings.Contains(text, fence) {
		fence += "`"
	}
	return fence + language + "\n" + text + "\n" + fence + "\n\n"
}

// oneLine collapses whitespace and cuts the text to limit characters.
func oneLine(s string, limit int) string {
	s = strings.Join(strings.Fields(s), " ")
	if short, ok := cutRunes(s, limit); ok {
		return short + "..."
	}
	return s
}

func rowsText(rows *int64) string {
	if rows == nil {
		return "rows not reported"
	}
	if *rows == 1 {
		return "1 row"
	}
	return formatCount(*rows) + " rows"
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return formatCount(int64(n)) + " " + noun + "s"
}

// more reads "3 more statements".
func more(n int, noun string) string {
	if n != 1 {
		noun += "s"
	}
	return formatCount(int64(n)) + " more " + noun
}

func formatCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	start := 0
	if n < 0 {
		start = 1
	}
	for i := len(s) - 3; i > start; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
