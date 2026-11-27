package result

import (
	"crypto/sha256"
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"
)

// whatRan is what the finished databases of the detailed stage ran: one
// script per group of databases that ran the same SQL, and the databases it
// counts apart.
type whatRan struct {
	// scripts are ordered by how many databases ran them, most first.
	scripts []*script
	// retried databases have a script for their latest run only.
	retried int
	// unreadable databases finished, but their log could not be read.
	unreadable int
	// tooLarge databases ran more declarative SQL than maxDatabaseSQL.
	tooLarge int
	// dropped means the result's declarative SQL passed maxResultSQL, so no
	// script is shown.
	dropped bool
}

// script is everything a group of databases ran, in order.
type script struct {
	letter string
	// tasks are in the stage's task order.
	tasks []*Task
	units []*unit
}

// unit is one step of a script: a migration file or a sheet, which runs all
// its statements wherever it finishes, or one statement of a declarative
// file, whose SQL the deploy generates for each database.
type unit struct {
	// header names a file. It is empty for a sheet and a declarative statement.
	header string
	// ran holds each database's run of the unit, in the order of the
	// script's tasks.
	ran []unitRun
}

// unitRun is one database's run of a unit.
type unitRun struct {
	// commands are a file's or a sheet's final-attempt commands, and count
	// how many of them ran, which can be more than it holds.
	commands []*Command
	count    int
	rows     *int64
	// sql is a declarative statement.
	sql *sdlStatement
}

func (r *Result) whatRan() *whatRan {
	w := &whatRan{dropped: r.sqlDropped}
	stage := r.detail()
	if stage == nil {
		return w
	}
	byKey := map[[sha256.Size]byte]*script{}
	for _, task := range stage.Tasks {
		if !task.changed() {
			continue
		}
		switch {
		case task.LogUnavailable:
			w.unreadable++
			continue
		case task.TaskRun == "":
			continue
		case slices.ContainsFunc(task.Files, func(f *File) bool { return f.sqlOmitted }):
			w.tooLarge++
			continue
		default:
		}
		runs := task.unitRuns()
		if len(runs) == 0 {
			continue
		}
		if task.Attempt > 1 {
			w.retried++
		}
		key := scriptKey(runs)
		s, ok := byKey[key]
		if !ok {
			s = &script{}
			for _, run := range runs {
				s.units = append(s.units, &unit{header: run.header})
			}
			byKey[key] = s
			w.scripts = append(w.scripts, s)
		}
		s.tasks = append(s.tasks, task)
		for i, run := range runs {
			s.units[i].ran = append(s.units[i].ran, run.unitRun)
		}
	}
	if w.dropped {
		w.scripts = nil
		return w
	}
	// Stable, so scripts run by as many databases keep the stage's order.
	slices.SortStableFunc(w.scripts, func(x, y *script) int {
		return len(y.tasks) - len(x.tasks)
	})
	for i, s := range w.scripts {
		s.letter = scriptLetter(i)
	}
	return w
}

// scriptKey identifies a script by a hash of its steps.
func scriptKey(runs []keyedRun) [sha256.Size]byte {
	h := sha256.New()
	for _, run := range runs {
		_, _ = h.Write([]byte(strconv.Itoa(len(run.key))))
		_, _ = h.Write([]byte{':'})
		_, _ = h.Write([]byte(run.key))
	}
	var key [sha256.Size]byte
	copy(key[:], h.Sum(nil))
	return key
}

// scriptLetter reads A to Z, then AA.
func scriptLetter(i int) string {
	letter := ""
	for i++; i > 0; i = (i - 1) / 26 {
		letter = string(rune('A'+(i-1)%26)) + letter
	}
	return letter
}

type keyedRun struct {
	unitRun
	key    string
	header string
}

// unitRuns lists what the task ran, in order.
func (t *Task) unitRuns() []keyedRun {
	var runs []keyedRun
	for _, file := range t.Files {
		if file.declarative {
			for i := range file.sql {
				runs = append(runs, keyedRun{key: "sql\x00" + strconv.Itoa(file.sql[i].id), unitRun: unitRun{sql: &file.sql[i]}})
			}
			continue
		}
		commands, count := finalAttempt(file.Commands, file.CommandCount)
		runs = append(runs, keyedRun{
			key:     "file\x00" + file.Version + "\x00" + file.Path,
			header:  file.name(),
			unitRun: unitRun{commands: commands, count: count, rows: file.AffectedRows},
		})
	}
	if len(t.Files) == 0 && (t.CommandCount > 0 || len(t.Commands) > 0) {
		sheet := t.sheet
		if sheet == "" {
			sheet = t.Name
		}
		commands, count := finalAttempt(t.Commands, t.CommandCount)
		runs = append(runs, keyedRun{
			key:     "sheet\x00" + sheet,
			unitRun: unitRun{commands: commands, count: count, rows: t.AffectedRows},
		})
	}
	return runs
}

// finalAttempt drops the commands of attempts the executor retried, which
// the result keeps first, from the commands and from their count.
func finalAttempt(commands []*Command, count int) ([]*Command, int) {
	var final []*Command
	var earlier int
	for _, command := range commands {
		if command.EarlierAttempt {
			earlier++
			continue
		}
		final = append(final, command)
	}
	return final, max(count, len(commands)) - earlier
}

// statementCount is how many statements the script ran.
func (s *script) statementCount() int {
	var n int
	for _, u := range s.units {
		if u.ran[0].sql != nil {
			n++
			continue
		}
		n += u.ran[0].count
	}
	return n
}

// shownStatement is one statement as a script shows it.
type shownStatement struct {
	text string
	// rows are each database's count, nil when the counts do not line up.
	rows []*int64
}

// statements are the unit's statements, with each database's count. A file
// whose text was not read has none.
func (u *unit) statements() []shownStatement {
	first := u.ran[0]
	if first.sql != nil {
		rows := make([]*int64, 0, len(u.ran))
		for _, run := range u.ran {
			rows = append(rows, run.sql.rows)
		}
		return []shownStatement{{text: first.sql.text, rows: rows}}
	}
	if u.textMissing() {
		return nil
	}
	// Counts line up by position only where every database ran the same
	// number of statements; databases on different engines can split one
	// file differently.
	aligned := !slices.ContainsFunc(u.ran, func(run unitRun) bool {
		return run.count != first.count || len(run.commands) != len(first.commands)
	})
	shown := make([]shownStatement, 0, len(first.commands))
	for i, command := range first.commands {
		s := shownStatement{text: strings.TrimSpace(command.Statement)}
		if aligned {
			for _, run := range u.ran {
				s.rows = append(s.rows, standingRows(run.commands[i]))
			}
		}
		shown = append(shown, s)
	}
	return shown
}

func (u *unit) textMissing() bool {
	return slices.ContainsFunc(u.ran[0].commands, func(command *Command) bool {
		return strings.TrimSpace(command.Statement) == ""
	})
}

// rows are each database's count for a file or a sheet.
func (u *unit) rows() []*int64 {
	var rows []*int64
	for _, run := range u.ran {
		rows = append(rows, run.rows)
	}
	return rows
}

// sumAll adds the counts, and returns nil when any is missing.
func sumAll(counts []*int64) *int64 {
	var total int64
	for _, count := range counts {
		if count == nil {
			return nil
		}
		total += *count
	}
	return &total
}

// rowsAcross reads one count for a statement the script's databases each
// ran: "25 rows" for one database, "25 rows on each database" when they
// agree, and their sum otherwise. Any missing count leaves the total unknown.
func rowsAcross(counts []*int64) string {
	total := sumAll(counts)
	if total == nil || len(counts) == 1 {
		return rowsText(total)
	}
	if !slices.ContainsFunc(counts, func(count *int64) bool { return *count != *counts[0] }) {
		return rowsText(counts[0]) + " on each database"
	}
	return rowsText(total) + " across " + plural(len(counts), "database")
}

// sqlLines are the script's SQL within shownScriptStatements, with a file's
// path and a count of changed rows as comments. It also returns whether a
// statement was cut, and how many statements it left out.
func (s *script) sqlLines() ([]string, bool, int) {
	var lines []string
	var cut bool
	var shown int
	for _, u := range s.units {
		if shown >= shownScriptStatements {
			break
		}
		if u.textMissing() {
			// One line stands for the file's statements.
			count := u.ran[0].count
			line := plural(count, "statement") + ", " + rowsAcross(u.rows()) + "; the text could not be read"
			if u.header != "" {
				line = u.header + ": " + line
			}
			lines = append(lines, "-- "+line)
			shown += count
			continue
		}
		if u.header != "" {
			lines = append(lines, "-- "+u.header)
		}
		for _, st := range u.statements() {
			if shown == shownScriptStatements {
				break
			}
			text, c := shorten(withSemicolon(st.text), sqlCut)
			if len(text) == 0 {
				continue
			}
			shown++
			cut = cut || c
			if total := sumAll(st.rows); total != nil && *total > 0 {
				lines = append(lines, "-- "+rowsAcross(st.rows))
			}
			lines = append(lines, text...)
		}
	}
	return lines, cut, max(s.statementCount()-shown, 0)
}

func withSemicolon(statement string) string {
	if statement == "" || strings.HasSuffix(statement, ";") {
		return statement
	}
	return statement + ";"
}

func (s *script) title() string {
	return fmt.Sprintf("Script %s · %s", s.letter, plural(len(s.tasks), "database"))
}

// names reads "a, b, c, d, e and 295 more", or with a noun "… and 295 more
// databases".
func names(tasks []*Task, n int, noun string, name func(*Task) string) string {
	shown := make([]string, 0, n)
	for _, task := range tasks[:min(len(tasks), n)] {
		shown = append(shown, name(task))
	}
	list := strings.Join(shown, ", ")
	if rest := len(tasks) - len(shown); rest > 0 && noun == "" {
		list += " and " + formatCount(int64(rest)) + " more"
	} else if rest > 0 {
		list += " and " + more(rest, noun)
	}
	return list
}

// render returns the parts the summary and the log both show: the title, the
// SQL lines, the notes after them and whether the script folds. A folded
// script's title counts its statements.
func (s *script) render(inBytebase string) (string, []string, []string, bool) {
	lines, cut, rest := s.sqlLines()
	title := s.title()
	long := len(lines) > openScriptLines
	if long {
		title += " · " + plural(s.statementCount(), "statement")
	}
	var notes []string
	if cut {
		notes = append(notes, scriptShortenedNote)
	}
	if rest > 0 {
		notes = append(notes, fmt.Sprintf("and %s, %s.", more(rest, "statement"), inBytebase))
	}
	return title, lines, notes, long
}

func (s *script) markdown(inBytebase string, name func(*Task) string) string {
	title, lines, notes, long := s.render(inBytebase)
	open := " open"
	if long {
		open = ""
	}
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "<details%s><summary><strong>%s</strong></summary>\n\n", open, title)
	_, _ = sb.WriteString(names(s.tasks, namedDatabases, "", name) + "\n\n")
	if len(s.tasks) > namedDatabases {
		_, _ = fmt.Fprintf(&sb, "<details><summary>All %s</summary>\n\n%s\n\n</details>\n\n", plural(len(s.tasks), "database"), names(s.tasks, listedDatabases, "database", name))
	}
	if len(lines) > 0 {
		_, _ = sb.WriteString(fenced("sql", strings.Join(lines, "\n")))
	}
	for _, note := range notes {
		_, _ = sb.WriteString(note + "\n\n")
	}
	_, _ = sb.WriteString("</details>\n\n")
	return sb.String()
}

// text returns the script's title, its lines under the title, and whether it
// folds. The log cannot fold everywhere, so it lists the databases once.
func (s *script) text(inBytebase string) (string, []string, bool) {
	title, lines, notes, long := s.render(inBytebase)
	detail := []string{"  " + names(s.tasks, listedDatabases, "database", func(t *Task) string { return t.Target })}
	detail = append(detail, indented(lines)...)
	detail = append(detail, indented(notes)...)
	return title, detail, long
}

// notes are the sentences under the counts line. inBytebase is the words
// that point to Bytebase, a link in the summary.
func (w *whatRan) notes(inBytebase string) []string {
	var notes []string
	if w.retried > 0 {
		their := "their earlier runs are"
		if w.retried == 1 {
			their = "its earlier runs are"
		}
		notes = append(notes, fmt.Sprintf("%s retried; %s %s.", wereDatabases(w.retried), their, inBytebase))
	}
	if w.unreadable > 0 {
		owner := plural(w.unreadable, "database") + "'"
		if w.unreadable == 1 {
			owner = "1 database's"
		}
		notes = append(notes, fmt.Sprintf("%s statements could not be read; they are %s.", owner, inBytebase))
	}
	if w.tooLarge > 0 {
		notes = append(notes, fmt.Sprintf("Statements run on %s are not shown; they are %s.", plural(w.tooLarge, "database"), inBytebase))
	}
	return notes
}

// wereDatabases reads "3 databases were" or "1 database was".
func wereDatabases(n int) string {
	if n == 1 {
		return "1 database was"
	}
	return plural(n, "database") + " were"
}

// moreScripts names the scripts left out by their letters, which the
// database table shows: "and scripts F–H on 5 databases, in Bytebase."
func moreScripts(rest []*script, inBytebase string) string {
	var databases int
	for _, s := range rest {
		databases += len(s.tasks)
	}
	first, last := rest[0].letter, rest[len(rest)-1].letter
	var scripts string
	switch len(rest) {
	case 1:
		scripts = "script " + first
	case 2:
		scripts = "scripts " + first + " and " + last
	default:
		scripts = "scripts " + first + "–" + last
	}
	return fmt.Sprintf("and %s on %s, %s.", scripts, plural(databases, "database"), inBytebase)
}

func droppedScripts(inBytebase string) string {
	return "The scripts are too large to show here; they are " + inBytebase + "."
}

func (w *whatRan) textLines(fold Folder, inBytebase string) []string {
	if w.dropped {
		return []string{droppedScripts(inBytebase)}
	}
	var lines []string
	for i, s := range w.scripts {
		if i == shownScripts {
			break
		}
		if i > 0 {
			lines = append(lines, "")
		}
		title, detail, long := s.text(inBytebase)
		if fold != nil && long {
			lines = append(lines, fold(i, title, detail)...)
			continue
		}
		lines = append(lines, title)
		lines = append(lines, detail...)
	}
	if len(w.scripts) > shownScripts {
		lines = append(lines, "", moreScripts(w.scripts[shownScripts:], inBytebase))
	}
	return lines
}

// markdown is the scripts in at most budget bytes. A script that does not
// fit ends them, so a later, smaller one never takes its place.
func (w *whatRan) markdown(budget int, inBytebase string, name func(*Task) string) string {
	if w.dropped {
		return droppedScripts(inBytebase) + "\n\n"
	}
	var sb strings.Builder
	// Room for the closing line.
	budget -= 200
	var shown int
	for _, s := range w.scripts {
		if shown == shownScripts {
			break
		}
		block := s.markdown(inBytebase, name)
		if len(block) > budget {
			break
		}
		budget -= len(block)
		_, _ = sb.WriteString(block)
		shown++
	}
	if shown < len(w.scripts) {
		_, _ = sb.WriteString(moreScripts(w.scripts[shown:], inBytebase) + "\n\n")
	}
	return sb.String()
}

// letters maps each database with a script to its script's letter.
func (w *whatRan) letters() map[*Task]string {
	letters := map[*Task]string{}
	for _, s := range w.scripts {
		for _, task := range s.tasks {
			letters[task] = s.letter
		}
	}
	return letters
}

// markdownNames names a database alone, adding its instance only when
// another database in the stage has the same name.
func markdownNames(tasks []*Task) func(*Task) string {
	seen := map[string]int{}
	for _, task := range tasks {
		_, database := task.instanceAndDatabase()
		seen[database]++
	}
	return func(task *Task) string {
		instance, database := task.instanceAndDatabase()
		switch {
		case database == "":
			return html.EscapeString("instance " + instance)
		case seen[database] > 1:
			return html.EscapeString(database + " (instance " + instance + ")")
		default:
			return html.EscapeString(database)
		}
	}
}
