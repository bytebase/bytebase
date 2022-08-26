package parser

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode"

	"github.com/bytebase/bytebase/plugin/parser/ast"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"
)

const (
	eofRune = rune(-1)
)

var (
	beginRuneList     = []rune{'B', 'E', 'G', 'I', 'N'}
	atomicRuneList    = []rune{'A', 'T', 'O', 'M', 'I', 'C'}
	delimiterRuneList = []rune{'D', 'E', 'L', 'I', 'M', 'I', 'T', 'E', 'R'}
)

type tokenizer struct {
	buffer    []rune
	cursor    uint
	len       uint
	line      int
	startLine int

	// steaming API specific field
	scanner *bufio.Scanner
	f       func(string) error
	scanErr error
}

// newTokenizer creates a new tokenizer.
// Notice: we append an additional eofRune in the statement. This is a sentinel rune.
func newTokenizer(statement string) *tokenizer {
	t := &tokenizer{
		buffer:    []rune(statement),
		cursor:    0,
		line:      1,
		startLine: 1,
	}
	t.len = uint(len(t.buffer))
	// append an additional eofRune.
	t.buffer = append(t.buffer, eofRune)
	return t
}

// scanLines is a split function for a Scanner that each line of text, also contains trailing end-of-line marker.
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data[0 : i+1], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func newStreamTokenizer(src io.Reader, f func(string) error) *tokenizer {
	t := &tokenizer{
		cursor:    0,
		line:      1,
		startLine: 1,
		scanner:   bufio.NewScanner(src),
		f:         f,
	}
	t.scanner.Split(scanLines)
	if t.scanner.Scan() {
		t.buffer = []rune(t.scanner.Text())
		t.len = uint(len(t.buffer))
	} else {
		t.scanner = nil
		t.buffer = append(t.buffer, eofRune)
	}
	return t
}

func (t *tokenizer) setLineForMySQLCreateTableStmt(node *tidbast.CreateTableStmt) error {
	// We assume that the parser will parse the columns and table constraints according to the order of the raw SQL statements
	// and the identifiers don't equal any keywords in CREATE TABLE statements.
	// If it breaks our assumption, we set the line for columns and table constraints to the first line of the CREATE TABLE statement.
	for _, col := range node.Cols {
		col.SetOriginTextPosition(node.OriginTextPosition())
	}
	for _, cons := range node.Constraints {
		cons.SetOriginTextPosition(node.OriginTextPosition())
	}

	columnPos := 0
	constraintPos := 0
	// find the '(' for CREATE TABLE ... ( ... )
	if err := t.scanTo([]rune{'('}); err != nil {
		return err
	}

	// parentheses is the flag for matching parentheses.
	parentheses := 1
	t.skipBlank()
	startPos := t.pos()
	t.startLine = t.line
	for {
		switch t.char(0) {
		case '\n':
			t.line++
			t.skip(1)
		case '\'', '"':
			if err := t.scanString(t.char(0)); err != nil {
				return err
			}
		case '(':
			parentheses++
			t.skip(1)
		case ')':
			parentheses--
			if parentheses == 0 {
				// This means we find the corresponding ')' for the first '(' in CREATE TABLE statements.
				// We need to check the definition and return.
				def := strings.ToLower(t.getString(startPos, t.pos()-startPos))
				if columnPos < len(node.Cols) &&
					strings.Contains(def, node.Cols[columnPos].Name.Name.L) {
					node.Cols[columnPos].SetOriginTextPosition(t.startLine + node.OriginTextPosition() - 1)
				} else if constraintPos < len(node.Constraints) &&
					matchMySQLTableConstraint(def, node.Constraints[constraintPos]) {
					node.Constraints[constraintPos].SetOriginTextPosition(t.startLine + node.OriginTextPosition() - 1)
				}
				return nil
			}
			t.skip(1)
		case ',':
			// e.g. CREATE TABLE t(
			//   a int,
			//   b int,
			//   UNIQUE(a, b),
			//   UNIQUE(b)
			// )
			// We don't need to consider the ',' in UNIQUE(a, b)
			if parentheses > 1 {
				t.skip(1)
				continue
			}
			def := strings.ToLower(t.getString(startPos, t.pos()-startPos))
			if columnPos < len(node.Cols) &&
				strings.Contains(def, node.Cols[columnPos].Name.Name.L) {
				node.Cols[columnPos].SetOriginTextPosition(t.startLine + node.OriginTextPosition() - 1)
				columnPos++
			} else if constraintPos < len(node.Constraints) &&
				matchMySQLTableConstraint(def, node.Constraints[constraintPos]) {
				node.Constraints[constraintPos].SetOriginTextPosition(t.startLine + node.OriginTextPosition() - 1)
				constraintPos++
			}
			t.skip(1)
			t.skipBlank()
			startPos = t.pos()
			t.startLine = t.line
		case eofRune:
			return nil
		default:
			t.skip(1)
		}
	}
}

func matchMySQLTableConstraint(text string, cons *tidbast.Constraint) bool {
	text = strings.ToLower(text)
	if cons.Name != "" {
		return strings.Contains(text, strings.ToLower(cons.Name))
	}
	switch cons.Tp {
	case tidbast.ConstraintCheck:
		return strings.Contains(text, "check")
	case tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex:
		return strings.Contains(text, "unique")
	case tidbast.ConstraintPrimaryKey:
		return strings.Contains(text, "primary key")
	case tidbast.ConstraintForeignKey:
		return strings.Contains(text, "foreign key")
	case tidbast.ConstraintIndex:
		notUnique := !strings.Contains(text, "unique")
		notPrimary := !strings.Contains(text, "primary key")
		notForeign := !strings.Contains(text, "foreign key")
		isIndex := strings.Contains(text, "index") || strings.Contains(text, "key")
		return notUnique && notPrimary && notForeign && isIndex
	}
	return false
}

// setLineForPGCreateTableStmt sets the line for columns and table constraints in CREATE TABLE statements.
func (t *tokenizer) setLineForPGCreateTableStmt(node *ast.CreateTableStmt) error {
	// We assume that the parser will parse the columns and table constraints according to the order of the raw SQL statements
	// and the identifiers don't equal any keywords in CREATE TABLE statements.
	// If it breaks our assumption, we set the line for columns and table constraints to the first line of the CREATE TABLE statement.
	for _, col := range node.ColumnList {
		col.SetLine(node.Line())
		for _, inlineCons := range col.ConstraintList {
			inlineCons.SetLine(node.Line())
		}
	}
	for _, cons := range node.ConstraintList {
		cons.SetLine(node.Line())
	}

	columnPos := 0
	constraintPos := 0
	// find the '(' for CREATE TABLE ... ( ... )
	if err := t.scanTo([]rune{'('}); err != nil {
		return err
	}

	// parentheses is the flag for matching parentheses.
	parentheses := 1
	t.skipBlank()
	startPos := t.pos()
	t.startLine = t.line
	for {
		switch t.char(0) {
		case '\n':
			t.line++
			t.skip(1)
		case '\'':
			if err := t.scanString('\''); err != nil {
				return err
			}
		case '(':
			parentheses++
			t.skip(1)
		case ')':
			parentheses--
			if parentheses == 0 {
				// This means we find the corresponding ')' for the first '(' in CREATE TABLE statements.
				// We need to check the definition and return.
				def := strings.ToLower(t.getString(startPos, t.pos()-startPos))
				if columnPos < len(node.ColumnList) &&
					strings.Contains(def, strings.ToLower(node.ColumnList[columnPos].ColumnName)) {
					node.ColumnList[columnPos].SetLine(t.startLine + node.Line() - 1)
					for _, inlineConstraint := range node.ColumnList[columnPos].ConstraintList {
						inlineConstraint.SetLine(node.ColumnList[columnPos].Line())
					}
				} else if constraintPos < len(node.ConstraintList) &&
					matchTableConstraint(def, node.ConstraintList[constraintPos]) {
					node.ConstraintList[constraintPos].SetLine(t.startLine + node.Line() - 1)
				}
				return nil
			}
			t.skip(1)
		case ',':
			// e.g. CREATE TABLE t(
			//   a int,
			//   b int,
			//   UNIQUE(a, b),
			//   UNIQUE(b)
			// )
			// We don't need to consider the ',' in UNIQUE(a, b)
			if parentheses > 1 {
				t.skip(1)
				continue
			}
			def := strings.ToLower(t.getString(startPos, t.pos()-startPos))
			if columnPos < len(node.ColumnList) &&
				strings.Contains(def, strings.ToLower(node.ColumnList[columnPos].ColumnName)) {
				node.ColumnList[columnPos].SetLine(t.startLine + node.Line() - 1)
				for _, inlineConstraint := range node.ColumnList[columnPos].ConstraintList {
					inlineConstraint.SetLine(node.ColumnList[columnPos].Line())
				}
				columnPos++
			} else if constraintPos < len(node.ConstraintList) &&
				matchTableConstraint(def, node.ConstraintList[constraintPos]) {
				node.ConstraintList[constraintPos].SetLine(t.startLine + node.Line() - 1)
				constraintPos++
			}
			t.skip(1)
			t.skipBlank()
			startPos = t.pos()
			t.startLine = t.line
		case eofRune:
			return nil
		default:
			t.skip(1)
		}
	}
}

// matchTableConstraint matches text as lowercase.
func matchTableConstraint(text string, cons *ast.ConstraintDef) bool {
	text = strings.ToLower(text)
	if cons.Name != "" {
		return strings.Contains(text, strings.ToLower(cons.Name))
	}
	switch cons.Type {
	case ast.ConstraintTypeCheck:
		return strings.Contains(text, "check")
	case ast.ConstraintTypeUnique:
		return strings.Contains(text, "unique")
	case ast.ConstraintTypePrimary:
		return strings.Contains(text, "primary key")
	case ast.ConstraintTypeForeign:
		return strings.Contains(text, "foreign key")
	}
	return false
}

// splitMySQLMultiSQL splits the statement to a string slice.
func (t *tokenizer) splitMySQLMultiSQL() ([]SingleSQL, error) {
	var res []SingleSQL
	delimiter := []rune{';'}

	t.skipBlank()
	t.startLine = t.line
	startPos := t.cursor
	for {
		switch {
		case t.char(0) == eofRune:
			s := t.getString(startPos, t.pos())
			if !emptyString(s) {
				if t.f == nil {
					res = append(res, SingleSQL{
						Text: s,
						Line: t.startLine,
					})
				}
				if err := t.processStreaming(s); err != nil {
					return nil, err
				}
			}
			return res, t.scanErr
		case t.equalWordCaseInsensitive(delimiter):
			t.skip(uint(len(delimiter)))
			text := t.getString(startPos, t.pos()-startPos)
			if t.f == nil {
				res = append(res, SingleSQL{
					Text: text,
					Line: t.startLine,
				})
			}
			t.skipBlank()
			if err := t.processStreaming(text); err != nil {
				return nil, err
			}
			t.startLine = t.line
			startPos = t.pos()
		// deal with the DELIMITER statement, see https://dev.mysql.com/doc/refman/8.0/en/stored-programs-defining.html
		case t.equalWordCaseInsensitive(delimiterRuneList):
			t.skip(uint(len(delimiterRuneList)))
			t.skipBlank()
			delimiterStart := t.pos()
			t.skipToBlank()
			delimiter = t.runeList(delimiterStart, t.pos()-delimiterStart)
			text := t.getString(startPos, t.pos()-startPos)
			if t.f == nil {
				res = append(res, SingleSQL{
					Text: text,
					Line: t.startLine,
				})
			}
			t.skipBlank()
			if err := t.processStreaming(text); err != nil {
				return nil, err
			}
			t.startLine = t.line
			startPos = t.pos()
		case t.char(0) == '/' && t.char(1) == '*':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
		case t.char(0) == '-' && t.char(1) == '-':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
		case t.char(0) == '#':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
		case t.char(0) == '\'' || t.char(0) == '"':
			if err := t.scanString(t.char(0)); err != nil {
				return nil, err
			}
		case t.char(0) == '`':
			if err := t.scanIdentifier('`'); err != nil {
				return nil, err
			}
		case t.char(0) == '\n':
			t.line++
			t.skip(1)
		default:
			t.skip(1)
		}
	}
}

// splitPostgreSQLMultiSQL splits the statement to a string slice.
// We mainly considered:
//
//	comments
//	- style /* comments */
//	- style -- comments
//	string
//	- style 'string'
//	- style $$ string $$
//	identifier
//	- style "indentifier"
//
// Notice:
//   - We support PostgreSQL CREATE PROCEDURE statement with $$ $$ style,
//     but do not support BEGIN ATOMIC ... END; style.
//     See https://www.postgresql.org/docs/14/sql-createprocedure.html.
func (t *tokenizer) splitPostgreSQLMultiSQL() ([]SingleSQL, error) {
	var res []SingleSQL

	t.skipBlank()
	t.startLine = t.line
	startPos := t.cursor
	for {
		switch {
		case t.char(0) == '/' && t.char(1) == '*':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
		case t.char(0) == '-' && t.char(1) == '-':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
		case t.char(0) == '\'':
			if err := t.scanString('\''); err != nil {
				return nil, err
			}
		case t.char(0) == '$':
			if err := t.scanDoubleDollarQuotedString(); err != nil {
				return nil, err
			}
		case t.char(0) == '"':
			if err := t.scanIdentifier('"'); err != nil {
				return nil, err
			}
		case t.char(0) == ';':
			t.skip(1)
			text := t.getString(startPos, t.pos()-startPos)
			if t.f == nil {
				res = append(res, SingleSQL{
					Text: text,
					Line: t.startLine,
				})
			}
			t.skipBlank()
			if err := t.processStreaming(text); err != nil {
				return nil, err
			}
			t.startLine = t.line
			startPos = t.pos()
		case t.char(0) == eofRune:
			s := t.getString(startPos, t.pos())
			if !emptyString(s) {
				if t.f == nil {
					res = append(res, SingleSQL{
						Text: s,
						Line: t.startLine,
					})
				}
				if err := t.processStreaming(s); err != nil {
					return nil, err
				}
			}
			return res, t.scanErr
		// return error when meeting BEGIN ATOMIC.
		case t.equalWordCaseInsensitive(beginRuneList):
			t.skip(uint(len(beginRuneList)))
			t.skipBlank()
			if t.equalWordCaseInsensitive(atomicRuneList) {
				return nil, errors.Errorf("not support BEGIN ATOMIC ... END in PostgreSQL CREATE PROCEDURE statement, please use double doller style($$ or $tag$) instead of it")
			}
		case t.char(0) == '\n':
			t.line++
			t.skip(1)
		default:
			t.skip(1)
		}
	}
}

// Assume that identifier only contains letters, underscores, digits (0-9), or dollar signs ($).
// See https://www.postgresql.org/docs/current/sql-syntax-lexical.html.
func (t *tokenizer) scanIdentifier(delimiter rune) error {
	if t.char(0) != delimiter {
		return errors.Errorf("delimiter doesn't start with delimiter: %c, but found: %c", delimiter, t.char(0))
	}

	t.skip(1)
	for {
		switch t.char(0) {
		case delimiter:
			t.skip(1)
			return nil
		case eofRune:
			return errors.Errorf("invalid indentifier: not found delimiter: %c, but found EOF", delimiter)
		default:
			t.skip(1)
		}
	}
}

// There are two ways to include a single quote('), using \' or two single-quotes.
// We only handle the case \', because the second case does not require special handling.
// And this is extensible.
// For MySQL, user can enclose string within double quote(").
func (t *tokenizer) scanString(delimiter rune) error {
	if t.char(0) != delimiter {
		return errors.Errorf("string doesn't start with delimiter: %c, but found: %c", delimiter, t.char(0))
	}

	t.skip(1)
	for {
		switch t.char(0) {
		case delimiter:
			t.skip(1)
			return nil
		case eofRune:
			return errors.Errorf("invalid string: not found delimiter: %c, but found EOF", delimiter)
		case '\\':
			// skip two because we want to skip \' and \\.
			t.skip(2)
		case '\n':
			t.line++
			t.skip(1)
		default:
			t.skip(1)
		}
	}
}

// Double dollar quoted string is a PostgreSQL-specific syntax.
// There are two syntax styles, tag and no tag:
// - $$ string $$
// - $tag$ string $tag$
// See https://www.postgresql.org/docs/current/sql-syntax-lexical.html.
func (t *tokenizer) scanDoubleDollarQuotedString() error {
	startPos := t.pos()
	// scan the tag string quoted by the dollar sign($)
	if err := t.scanString('$'); err != nil {
		return err
	}
	// here tag means $$ or $tag_string$ which means include the dollar sign($).
	tag := t.runeList(startPos, t.pos()-startPos)
	return t.scanTo(tag)
}

func (t *tokenizer) scanComment() error {
	switch {
	case t.char(0) == '/' && t.char(1) == '*':
		t.skip(2)
		for {
			switch {
			case t.char(0) == '*' && t.char(1) == '/':
				t.skip(2)
				return nil
			case t.char(0) == eofRune:
				return errors.Errorf("invalid comment: not found */, but found EOF")
			case t.char(0) == '\n':
				t.line++
				t.skip(1)
			default:
				t.skip(1)
			}
		}
	case t.char(0) == '-' && t.char(1) == '-':
		t.skip(2)
		t.skipToNewLine()
		return nil
	case t.char(0) == '#':
		t.skip(1)
		t.skipToNewLine()
		return nil
	}
	return errors.Errorf("no comment found")
}

// scanTo scans to delimiter. Use KMP algorithm.
func (t *tokenizer) scanTo(delimiter []rune) error {
	if len(delimiter) == 0 {
		return errors.Errorf("scanTo failed: delimiter can not be nil")
	}

	// KMP algorithm.
	// Build the next array.
	var next []int
	next = append(next, 0)
	now := 0
	i := 1
	for i < len(delimiter) {
		if delimiter[i] == delimiter[now] {
			now++
			next = append(next, now)
			i++
			continue
		}
		if now == 0 {
			next = append(next, 0)
			i++
			continue
		}
		now = next[now-1]
	}

	// Search delimiter
	pos := 0
	for {
		// find delimiter successfully
		if pos == len(delimiter) {
			return nil
		}
		if t.char(0) == eofRune {
			return errors.Errorf("scanTo failed: delimiter %q not found", string(delimiter))
		}
		if t.char(0) == '\n' {
			t.line++
		}
		if t.char(0) == delimiter[pos] {
			pos++
			t.skip(1)
		} else {
			if pos == 0 {
				t.skip(1)
			} else {
				pos = next[pos-1]
			}
		}
	}
}

func (t *tokenizer) scan() {
	if t.scanner != nil {
		if t.scanner.Scan() {
			t.buffer = append(t.buffer, []rune(t.scanner.Text())...)
			t.len = uint(len(t.buffer))
		} else {
			t.buffer = append(t.buffer, eofRune)
			t.scanErr = t.scanner.Err()
			t.scanner = nil
		}
	}
}

func (t *tokenizer) processStreaming(statement string) error {
	if t.f == nil {
		return nil
	}

	if err := t.f(statement); err != nil {
		return errors.Wrapf(err, "execute query %q failed", statement)
	}

	t.truncate(t.pos())
	return nil
}

// truncate will return the buffer after pos.
/*
Before:
buffer [.............]
              |
             pos
After:        |
buffer       [.......].
*/
func (t *tokenizer) truncate(pos uint) {
	if pos > t.len {
		pos = t.len
	}

	t.buffer = t.buffer[pos:]
	t.len = uint(len(t.buffer))
	if t.len > 0 && t.buffer[t.len-1] == eofRune {
		t.len--
	}
	t.cursor -= pos
}

func (t *tokenizer) char(after uint) rune {
	for t.cursor+after >= t.len && t.scanner != nil {
		t.scan()
	}

	if t.cursor+after >= t.len {
		return eofRune
	}

	return t.buffer[t.cursor+after]
}

func (t *tokenizer) skip(step uint) {
	t.cursor += step
	for t.cursor > t.len && t.scanner != nil {
		t.scan()
	}
	if t.cursor > t.len {
		t.cursor = t.len
	}
}

func (t *tokenizer) skipBlank() {
	r := t.char(0)
	for r == ' ' || r == '\n' || r == '\r' || r == '\t' {
		t.skip(1)
		if r == '\n' {
			t.line++
		}
		r = t.char(0)
	}
}

func (t *tokenizer) skipToBlank() {
	r := t.char(0)
	for r != ' ' && r != '\n' && r != '\r' && r != '\t' {
		t.skip(1)
		r = t.char(0)
	}
}

func (t *tokenizer) skipToNewLine() {
	r := t.char(0)
	for r != '\n' {
		if r == eofRune {
			return
		}
		t.skip(1)
		r = t.char(0)
	}
	t.line++
	t.skip(1)
}

func (t *tokenizer) pos() uint {
	return t.cursor
}

func (t *tokenizer) getString(startPos uint, length uint) string {
	return string(t.runeList(startPos, length))
}

func (t *tokenizer) runeList(startPos uint, length uint) []rune {
	endPos := startPos + length
	if endPos > t.len {
		endPos = t.len
	}
	return t.buffer[startPos:endPos]
}

func (t *tokenizer) equalWordCaseInsensitive(word []rune) bool {
	for i := range word {
		if unicode.ToLower(t.char(uint(i))) != unicode.ToLower(word[i]) {
			return false
		}
	}
	return true
}

func emptyRune(r rune) bool {
	return r != ' ' && r != '\n' && r != '\t' && r != '\r'
}

func emptyString(s string) bool {
	for _, c := range s {
		if !emptyRune(c) {
			return false
		}
	}

	return true
}
