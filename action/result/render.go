package result

import (
	"fmt"
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
)

// Folder wraps one task's lines so that a CI can collapse them. The index is
// unique within one rendering.
type Folder func(index int, header string, detail []string) []string

// Headline is the one line that says how the rollout stands.
func (r *Result) Headline() string {
	switch r.Outcome {
	case NotWaited:
		return "Rollout created. This job deployed nothing: no --target-stage was given."
	case StageNotFound:
		return fmt.Sprintf("This job deployed nothing: the rollout has no stage for %s.", r.TargetStage)
	case Interrupted:
		return fmt.Sprintf("Rollout to %s: INTERRUPTED. The action stopped before the stage finished, so this state is not final.", r.stageName())
	default:
		return fmt.Sprintf("Rollout to %s: %s", r.stageName(), r.Outcome)
	}
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

// RenderText renders the result for a job log. fold may be nil. The answer
// comes first and the rollout's URL last, since most readers never need it.
func RenderText(r *Result, fold Folder) string {
	lines := []string{r.Headline()}
	lines = append(lines, r.countLines()...)

	tasks := r.orderedTasks()
	if len(tasks) > 0 {
		lines = append(lines, "")
	}
	var statusWidth, targetWidth int
	for _, task := range tasks {
		statusWidth = max(statusWidth, len(task.Status))
		targetWidth = max(targetWidth, utf8.RuneCountInString(task.Target))
	}
	indent := strings.Repeat(" ", statusWidth+2)
	for i, task := range tasks {
		header := strings.TrimRight(fmt.Sprintf("%-*s  %-*s  %s", statusWidth, task.Status, targetWidth, task.Target, task.note()), " ")
		var detail []string
		for _, line := range task.detailLines() {
			detail = append(detail, indent+line)
		}
		// Retrying or skipping happens in Bytebase, so the task's URL sits
		// with the failure it follows from.
		if task.stopped() && task.URL != "" {
			detail = append(detail, indent+"Retry or skip: "+task.URL)
		}
		// A failure stays unfolded so the reader sees it without a click.
		if fold != nil && len(detail) > 0 && task.Status == "DONE" {
			lines = append(lines, fold(i, header, detail)...)
			continue
		}
		lines = append(lines, header)
		lines = append(lines, detail...)
	}
	if len(tasks) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, "Rollout: "+r.URL)
	return strings.Join(lines, "\n") + "\n"
}

// RenderMarkdown renders the result for a GitHub step summary. A link appears
// only where the page cannot answer: beside a failure, which is retried or
// skipped in Bytebase, and where the page shows no finished stage.
func RenderMarkdown(r *Result) string {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "### %s\n\n", r.Headline())
	tasks := r.orderedTasks()
	if len(tasks) == 0 || r.Outcome == Interrupted {
		_, _ = fmt.Fprintf(&sb, "[Follow the rollout in Bytebase](%s)\n\n", r.URL)
	}
	for _, line := range r.countLines() {
		_, _ = sb.WriteString(line + "\n\n")
	}
	if len(tasks) == 0 {
		return sb.String()
	}

	// Failures are written out first and placed after the table, so that a
	// large rollout spends the budget on them before anything else.
	var failures strings.Builder
	var omitted int
	for _, task := range tasks {
		if !task.stopped() {
			continue
		}
		block := task.failureMarkdown()
		if failures.Len()+len(block) > markdownBudget/2 {
			omitted++
			continue
		}
		_, _ = failures.WriteString(block)
	}
	if omitted > 0 {
		_, _ = fmt.Fprintf(&failures, "%s are in the output file and [in Bytebase](%s).\n\n", more(omitted, "failure"), r.URL)
	}
	budget := markdownBudget - sb.Len() - failures.Len()

	_, _ = sb.WriteString("| Database | Instance | Status | Files | Statements | Rows | Duration |\n|---|---|---|---|---|---|---|\n")
	var shown int
	for _, task := range tasks {
		row := task.markdownRow()
		if len(row) > budget {
			break
		}
		budget -= len(row)
		_, _ = sb.WriteString(row)
		shown++
	}
	if rest := len(tasks) - shown; rest > 0 {
		_, _ = fmt.Fprintf(&sb, "\n%s are in the output file and [in Bytebase](%s).\n", more(rest, "database"), r.URL)
	}
	_, _ = sb.WriteString("\n")
	_, _ = sb.WriteString(failures.String())

	for _, task := range tasks[:shown] {
		if task.Status != "DONE" {
			continue
		}
		detail := task.detailLines()
		if len(detail) == 0 {
			continue
		}
		block := fmt.Sprintf("<details><summary>Statements run on %s</summary>\n\n%s</details>\n\n", task.displayName(), fenced("text", strings.Join(detail, "\n")))
		if len(block) > budget {
			break
		}
		budget -= len(block)
		_, _ = sb.WriteString(block)
	}
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

// countLines summarizes the detailed stage by what happened to each database,
// and the other stages, in one line, by task status.
func (r *Result) countLines() []string {
	var lines []string
	detail := r.detail()
	if detail != nil {
		counts := map[string]int{}
		for _, task := range detail.Tasks {
			switch {
			case task.Status == "DONE" && task.Unchanged:
				counts["unchanged"]++
			case task.Status == "DONE":
				counts["changed"]++
			default:
				counts[statusWord(task.Status)]++
			}
		}
		lines = append(lines, joinCounts(counts, []string{"changed", "unchanged", "failed", "canceled", "skipped", "running", "pending", "not started"}))
	}
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
			parts = append(parts, fmt.Sprintf("%d %s", counts[key], key))
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
		parts = append(parts, fmt.Sprintf("%d %s", counts[key], key))
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
			parts = append(parts, "unchanged, nothing to apply")
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
			lines = append(lines, indented(commandLines(file.Commands, file.CommandCount, file.AffectedRows), "  ")...)
			continue
		}
		failed := failureLines(file.Commands, file.Error)
		errorShown = errorShown || len(failed) > 0
		lines = append(lines, indented(failed, "  ")...)
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
		detail, cut := shorten(t.Detail)
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

// displayName reads "employee on prod-instance".
func (t *Task) displayName() string {
	instance, database := t.instanceAndDatabase()
	if database == "" {
		return instance
	}
	return database + " on " + instance
}

func (t *Task) markdownRow() string {
	icons := map[string]string{"DONE": "✅", "FAILED": "❌", "CANCELED": "⛔", "SKIPPED": "⏭️"}
	status := t.Status
	if icon, ok := icons[t.Status]; ok {
		status = icon + " " + t.Status
	}
	if t.Unchanged {
		status += ", unchanged"
	}
	if t.Attempt > 1 {
		status += fmt.Sprintf(", attempt %d", t.Attempt)
	}
	files, statements, rows := "", "", ""
	if t.TaskRun != "" && !t.LogUnavailable {
		files = strconv.Itoa(len(t.Files))
		if t.FilesNotRun > 0 {
			files += fmt.Sprintf(" (%d not run)", t.FilesNotRun)
		}
		statements = strconv.Itoa(t.commandCount())
		if t.AffectedRows != nil {
			rows = formatCount(*t.AffectedRows)
		}
	}
	cell := func(s string) string {
		return strings.ReplaceAll(s, "|", "\\|")
	}
	instance, database := t.instanceAndDatabase()
	if database == "" {
		database = instance
	}
	return fmt.Sprintf("| [%s](%s) | %s | %s | %s | %s | %s | %s |\n", cell(database), t.URL, cell(instance), cell(status), files, statements, rows, t.duration())
}

func (t *Task) failureMarkdown() string {
	var sb strings.Builder
	file, statement, errText := t.failure()
	_, _ = fmt.Fprintf(&sb, "#### %s %s\n\n", t.displayName(), statusWord(t.Status))
	if file != nil {
		_, _ = fmt.Fprintf(&sb, "`%s` %s\n\n", file.name(), file.note())
	}
	statementLines, statementCut := shorten(statement)
	if len(statementLines) > 0 {
		_, _ = sb.WriteString(fenced("sql", strings.Join(statementLines, "\n")))
	}
	errorLines, errorCut := shorten(errText)
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

func (f *File) name() string {
	if f.Path != "" {
		return f.Path
	}
	return "version " + f.Version
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
		lines, statementCut := shorten(command.Statement)
		if len(lines) == 0 {
			lines = []string{commandLabel(command)}
		}
		errorLines, errorCut := shorten(command.Error)
		lines = append(lines, errorLines...)
		if statementCut || errorCut {
			lines = append(lines, shortenedNote)
		}
		return lines
	}
	lines, cut := shorten(scopeErr)
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

// shorten returns the text's first lines, each cut to shownLineWidth, and
// whether anything was left out.
func shorten(text string) ([]string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	lines := strings.Split(text, "\n")
	var cut bool
	if len(lines) > shownLines {
		rest := len(lines) - shownLines
		lines = append(lines[:shownLines:shownLines], "... "+more(rest, "line"))
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

func indented(lines []string, indent string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, indent+line)
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
