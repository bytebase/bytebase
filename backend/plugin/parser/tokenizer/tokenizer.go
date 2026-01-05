package tokenizer

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	eofRune = rune(-1)
)

var (
	delimiterRuneList = []rune{'D', 'E', 'L', 'I', 'M', 'I', 'T', 'E', 'R'}
)

type Tokenizer struct {
	buffer           []rune
	bufferByteOffset []int
	cursor           uint
	len              uint
	line             int
	emptyStatement   bool

	// steaming API specific field
	reader  *bufio.Reader
	f       func(string) error
	readErr error
}

// NewTokenizer creates a new tokenizer.
// Notice: we append an additional eofRune in the statement. This is a sentinel rune.
func NewTokenizer(statement string) *Tokenizer {
	t := &Tokenizer{
		buffer: []rune(statement),
		cursor: 0,
		line:   1,
	}
	for i := range statement {
		t.bufferByteOffset = append(t.bufferByteOffset, i)
	}
	t.len = uint(len(t.buffer))
	// append an additional eofRune.
	t.buffer = append(t.buffer, eofRune)
	t.bufferByteOffset = append(t.bufferByteOffset, len(statement))

	return t
}

func (t *Tokenizer) SetLineForMySQLCreateTableStmt(node *tidbast.CreateTableStmt, firstLine int) error {
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
	for {
		switch {
		case t.char(0) == '\n':
			t.line++
			t.skip(1)
		case t.char(0) == '/' && t.char(1) == '*':
			if err := t.scanComment(); err != nil {
				return err
			}
		case t.char(0) == '-' && t.char(1) == '-':
			if err := t.scanComment(); err != nil {
				return err
			}
		case t.char(0) == '#':
			if err := t.scanComment(); err != nil {
				return err
			}
		case t.char(0) == '\'' || t.char(0) == '"':
			if err := t.scanString(t.char(0)); err != nil {
				return err
			}
		case t.char(0) == '`':
			if err := t.scanIdentifier('`'); err != nil {
				return err
			}
		case t.char(0) == '(':
			parentheses++
			t.skip(1)
		case t.char(0) == ')':
			parentheses--
			if parentheses == 0 {
				// This means we find the corresponding ')' for the first '(' in CREATE TABLE statements.
				// We need to check the definition and return.
				def := strings.ToLower(t.getString(startPos, t.pos()-startPos))
				if columnPos < len(node.Cols) &&
					strings.Contains(def, node.Cols[columnPos].Name.Name.L) {
					// Consider this text:
					// CREATE TABLE t(
					//   a int
					// )
					//
					// Our current location is ')'.
					// The line (t.line + firstLine - 1) is the line of ')',
					// but we want to get the line of 'a int'.
					// So we need minus the aboveNonBlankLineDistance.
					node.Cols[columnPos].SetOriginTextPosition(t.line + firstLine - 1 - t.aboveNonBlankLineDistance())
				} else if constraintPos < len(node.Constraints) &&
					matchMySQLTableConstraint(def, node.Constraints[constraintPos]) {
					// Consider this text:
					// CREATE TABLE t(
					//   a int,
					//   UNIQUE (a)
					// )
					//
					// Our current location is ')'.
					// The line (t.line + firstLine - 1) is the line of ')',
					// but we want to get the line of 'UNIQUE (a)'.
					// So we need minus the aboveNonBlankLineDistance.
					node.Constraints[constraintPos].SetOriginTextPosition(t.line + firstLine - 1 - t.aboveNonBlankLineDistance())
				}
				return nil
			}
			t.skip(1)
		case t.char(0) == ',':
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
				node.Cols[columnPos].SetOriginTextPosition(t.line + firstLine - 1)
				columnPos++
			} else if constraintPos < len(node.Constraints) &&
				matchMySQLTableConstraint(def, node.Constraints[constraintPos]) {
				node.Constraints[constraintPos].SetOriginTextPosition(t.line + firstLine - 1)
				constraintPos++
			}
			t.skip(1)
			t.skipBlank()
			startPos = t.pos()
		case t.char(0) == eofRune:
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
	default:
		// Unknown constraint type
		return false
	}
}

func (t *Tokenizer) aboveNonBlankLineDistance() int {
	pos := uint(1)
	dis := 0
	for {
		c := t.preChar(pos)
		if c == '\n' {
			dis++
		} else if !emptyRune(c) {
			return dis
		}
		pos++
	}
}

// SplitTiDBMultiSQL splits the statement to a string slice.
func (t *Tokenizer) SplitTiDBMultiSQL() ([]base.Statement, error) {
	var res []base.Statement
	delimiter := []rune{';'}

	t.emptyStatement = true
	// Record position BEFORE skipping whitespace to include leading whitespace in Text
	startPos := t.cursor
	startLine := t.line          // Track the starting line number (1-based)
	startColumn := t.getColumn() // Track the starting column (1-based)
	t.skipBlank()
	for {
		switch {
		case t.char(0) == eofRune:
			s := t.getString(startPos, t.pos()-startPos)
			if !emptyString(s) {
				if t.f == nil {
					res = append(res, base.Statement{
						Text: s,
						Start: &store.Position{
							Line:   int32(startLine),
							Column: int32(startColumn),
						},
						// Consider this text:
						// CREATE TABLE t(
						//   a int
						// )
						//
						// EOF line
						//
						// Our current location is the EOF line.
						// The line t.line is the line of ')',
						// but we want to get the line of last line of the SQL
						// which means the line of ')'.
						// So we need minus the aboveNonBlankLineDistance.
						End: &store.Position{
							Line:   int32(t.line - t.aboveNonBlankLineDistance()),
							Column: int32(t.getLastContentColumn()),
						},
						Empty: t.emptyStatement,
						Range: &store.Range{
							Start: int32(t.getByteOffset(int(startPos))),
							End:   int32(t.getByteOffset(int(t.pos()))),
						},
					})
				}
				if err := t.processStreaming(s); err != nil {
					return nil, err
				}
			}

			return res, t.readErr
		case t.equalWordCaseInsensitive(delimiter):
			t.skip(uint(len(delimiter)))
			text := t.getString(startPos, t.pos()-startPos)
			if t.f == nil {
				res = append(res, base.Statement{
					Text: text,
					Start: &store.Position{
						Line:   int32(startLine),
						Column: int32(startColumn),
					},
					End: &store.Position{
						Line:   int32(t.line),
						Column: int32(t.getColumn()),
					},
					Empty: t.emptyStatement,
					Range: &store.Range{
						Start: int32(t.getByteOffset(int(startPos))),
						End:   int32(t.getByteOffset(int(t.pos()))),
					},
				})
			}
			if err := t.processStreaming(text); err != nil {
				return nil, err
			}
			// Record position BEFORE skipping whitespace to include leading whitespace in Text
			startPos = t.pos()
			startLine = t.line          // Update startLine for next statement
			startColumn = t.getColumn() // Update startColumn for next statement
			t.skipBlank()
			t.emptyStatement = true
		// deal with the DELIMITER statement, see https://dev.mysql.com/doc/refman/8.0/en/stored-programs-defining.html
		case t.equalWordCaseInsensitive(delimiterRuneList):
			t.skip(uint(len(delimiterRuneList)))
			t.skipBlank()
			delimiterStart := t.pos()
			t.skipToBlank()
			delimiter = t.runeList(delimiterStart, t.pos()-delimiterStart)
			text := t.getString(startPos, t.pos()-startPos)
			if t.f == nil {
				res = append(res, base.Statement{
					Text: text,
					Start: &store.Position{
						Line:   int32(startLine),
						Column: int32(startColumn),
					},
					End: &store.Position{
						Line:   int32(t.line),
						Column: int32(t.getColumn()),
					},
					Empty: false,
					Range: &store.Range{
						Start: int32(t.getByteOffset(int(startPos))),
						End:   int32(t.getByteOffset(int(t.pos()))),
					},
				})
			}
			if err := t.processStreaming(text); err != nil {
				return nil, err
			}
			// Record position BEFORE skipping whitespace to include leading whitespace in Text
			startPos = t.pos()
			startLine = t.line          // Update startLine for next statement
			startColumn = t.getColumn() // Update startColumn for next statement
			t.skipBlank()
			t.emptyStatement = true
		case t.char(0) == '/' && t.char(1) == '*':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
			t.skipBlank()
		case t.char(0) == '-' && t.char(1) == '-':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
			t.skipBlank()
		case t.char(0) == '#':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
			t.skipBlank()
		case t.char(0) == '\'' || t.char(0) == '"':
			if err := t.scanString(t.char(0)); err != nil {
				return nil, err
			}
			t.emptyStatement = false
		case t.char(0) == '`':
			if err := t.scanIdentifier('`'); err != nil {
				return nil, err
			}
			t.emptyStatement = false
		case t.char(0) == '\n':
			t.line++
			t.skip(1)
		default:
			t.skip(1)
			t.emptyStatement = false
		}
	}
}

// SplitStandardMultiSQL splits the statement to a string slice.
// We mainly considered:
//
//	comments
//	- style /* comments */
//	- style -- comments
//	string
//	- style 'string'
//	identifier
//	- style "identifier"
//
// The difference between PostgreSQL and Oracle is that PostgreSQL supports
// dollar-quoted string, but Oracle does not.
func (t *Tokenizer) SplitStandardMultiSQL() ([]base.Statement, error) {
	var res []base.Statement

	t.emptyStatement = true
	// Record position BEFORE skipping whitespace to include leading whitespace in Text
	startPos := t.cursor
	firstStatementLine := t.line
	firstStatementColumn := t.getColumn()
	t.skipBlank()
	for {
		switch {
		case t.char(0) == '/' && t.char(1) == '*':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
			t.skipBlank()
		case t.char(0) == '-' && t.char(1) == '-':
			if err := t.scanComment(); err != nil {
				return nil, err
			}
			t.skipBlank()
		case t.char(0) == '\'':
			if err := t.scanString('\''); err != nil {
				return nil, err
			}
			t.emptyStatement = false
		case t.char(0) == '"':
			if err := t.scanIdentifier('"'); err != nil {
				return nil, err
			}
			t.emptyStatement = false
		case t.char(0) == ';':
			t.skip(1)
			text := t.getString(startPos, t.pos()-startPos)
			if t.f == nil {
				res = append(res, base.Statement{
					Text: text,
					Start: &store.Position{
						Line:   int32(firstStatementLine), // 1-based per proto spec
						Column: int32(firstStatementColumn),
					},
					End: &store.Position{
						Line:   int32(t.line),        // 1-based per proto spec
						Column: int32(t.getColumn()), // 1-based exclusive (after semicolon)
					},
					Empty: t.emptyStatement,
					Range: &store.Range{
						Start: int32(t.getByteOffset(int(startPos))),
						End:   int32(t.getByteOffset(int(t.pos()))),
					},
				})
			}
			if err := t.processStreaming(text); err != nil {
				return nil, err
			}
			// Record position BEFORE skipping whitespace to include leading whitespace in Text
			startPos = t.pos()
			firstStatementLine = t.line
			firstStatementColumn = t.getColumn()
			t.skipBlank()
			t.emptyStatement = true
		case t.char(0) == eofRune:
			s := t.getString(startPos, t.pos())
			if !emptyString(s) {
				if t.f == nil {
					// For EOF case, we need to find the position of the last content character.
					// The cursor might be past all content (on blank lines or at EOF).
					endLine := t.line - t.aboveNonBlankLineDistance()
					res = append(res, base.Statement{
						Text: s,
						Start: &store.Position{
							Line:   int32(firstStatementLine), // 1-based per proto spec
							Column: int32(firstStatementColumn),
						},
						End: &store.Position{
							Line:   int32(endLine),                  // 1-based per proto spec
							Column: int32(t.getLastContentColumn()), // 1-based exclusive
						},
						Empty: t.emptyStatement,
						Range: &store.Range{
							Start: int32(t.getByteOffset(int(startPos))),
							End:   int32(t.getByteOffset(int(t.pos()))),
						},
					})
				}
				if err := t.processStreaming(s); err != nil {
					return nil, err
				}
			}
			return res, t.readErr
		case t.char(0) == '\n':
			t.line++
			t.skip(1)
		default:
			t.skip(1)
			t.emptyStatement = false
		}
	}
}

// Assume that identifier only contains letters, underscores, digits (0-9), or dollar signs ($).
// See https://www.postgresql.org/docs/current/sql-syntax-lexical.html.
func (t *Tokenizer) scanIdentifier(delimiter rune) error {
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
func (t *Tokenizer) scanString(delimiter rune) error {
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

func (t *Tokenizer) scanComment() error {
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
	default:
		// Not a comment
	}
	return errors.Errorf("no comment found")
}

// scanTo scans to delimiter. Use KMP algorithm.
func (t *Tokenizer) scanTo(delimiter []rune) error {
	if len(delimiter) == 0 {
		return errors.Errorf("scanTo failed: delimiter cannot be nil")
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

func (t *Tokenizer) scan() {
	if t.reader != nil {
		s, err := t.reader.ReadString('\n')
		switch err {
		case nil:
			t.buffer = append(t.buffer, []rune(s)...)
			t.len = uint(len(t.buffer))
		case io.EOF:
			// bufio.Reader treates EOF as an error, we need special handling.
			t.buffer = append(t.buffer, []rune(s)...)
			t.len = uint(len(t.buffer))
			t.reader = nil
			t.buffer = append(t.buffer, eofRune)
		default:
			t.reader = nil
			t.buffer = append(t.buffer, eofRune)
			t.readErr = err
		}
	}
}

func (t *Tokenizer) processStreaming(statement string) error {
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
func (t *Tokenizer) truncate(pos uint) {
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

// char returns the rune after the cursor, if out of range, return eofRune.
func (t *Tokenizer) char(after uint) rune {
	for t.cursor+after >= t.len && t.reader != nil {
		t.scan()
	}

	if t.cursor+after >= t.len {
		return eofRune
	}

	return t.buffer[t.cursor+after]
}

func (t *Tokenizer) preChar(before uint) rune {
	if t.cursor < before {
		return eofRune
	}

	return t.buffer[t.cursor-before]
}

// skip skips the cursor to the cursor + step, if out of range, skip will be set to len.
func (t *Tokenizer) skip(step uint) {
	t.cursor += step
	for t.cursor > t.len && t.reader != nil {
		t.scan()
	}
	if t.cursor > t.len {
		t.cursor = t.len
	}
}

// skipBlank skips the cursor to the first non-blank rune.
func (t *Tokenizer) skipBlank() {
	r := t.char(0)
	for unicode.IsSpace(r) {
		t.skip(1)
		if r == '\n' {
			t.line++
		}
		r = t.char(0)
	}
}

// skipToBlank skips the cursor to the first blank rune.
func (t *Tokenizer) skipToBlank() {
	r := t.char(0)
	for r != eofRune && !unicode.IsSpace(r) {
		t.skip(1)
		r = t.char(0)
	}
}

func (t *Tokenizer) skipToNewLine() {
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

// getColumn returns the 1-based column position of the current cursor.
// Per the proto spec, the first character of a line is column 1.
func (t *Tokenizer) getColumn() int {
	return t.getColumnAt(t.cursor)
}

// getColumnAt returns the 1-based column position at a specific rune position.
func (t *Tokenizer) getColumnAt(pos uint) int {
	for i := int(pos) - 1; i >= 0; i-- {
		if t.buffer[i] == '\n' {
			return int(pos) - i // 1-based: first char after newline is column 1
		}
	}
	return int(pos) + 1 // 1-based: first char of first line is column 1
}

// getLastContentColumn returns the 1-based column after the last non-blank character.
// Used for EOF case where we need exclusive end position.
func (t *Tokenizer) getLastContentColumn() int {
	for i := int(t.cursor) - 1; i >= 0; i-- {
		if !emptyRune(t.buffer[i]) && t.buffer[i] != eofRune {
			return t.getColumnAt(uint(i)) + 1 // Exclusive: position after last content char
		}
	}
	return 1
}

func (t *Tokenizer) pos() uint {
	return t.cursor
}

func (t *Tokenizer) getString(startPos uint, length uint) string {
	return string(t.runeList(startPos, length))
}

func (t *Tokenizer) runeList(startPos uint, length uint) []rune {
	endPos := startPos + length
	if endPos > t.len {
		endPos = t.len
	}
	return t.buffer[startPos:endPos]
}

func (t *Tokenizer) getByteOffset(runeIndex int) int {
	return t.bufferByteOffset[runeIndex]
}

func (t *Tokenizer) equalWordCaseInsensitive(word []rune) bool {
	for i := range word {
		if unicode.ToLower(t.char(uint(i))) != unicode.ToLower(word[i]) {
			return false
		}
	}
	return true
}

func emptyRune(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '\r'
}

func emptyString(s string) bool {
	for _, c := range s {
		if !emptyRune(c) {
			return false
		}
	}

	return true
}

// StandardRemoveQuotedTextAndComment removes the quoted text and comment.
func StandardRemoveQuotedTextAndComment(statement string) (string, error) {
	var buf bytes.Buffer
	t := NewTokenizer(statement)

	t.skipBlank()
	startPos := t.pos()
	for {
		switch {
		case t.char(0) == '/' && t.char(1) == '*':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanComment(); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '-' && t.char(1) == '-':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanComment(); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '\'':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanString('\''); err != nil {
				return "'", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == '"':
			text := t.getString(startPos, t.pos()-startPos)
			if err := t.scanIdentifier('"'); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			if err := buf.WriteByte(' '); err != nil {
				return "", err
			}
			startPos = t.pos()
		case t.char(0) == eofRune:
			text := t.getString(startPos, t.pos()-startPos)
			if _, err := buf.WriteString(text); err != nil {
				return "", err
			}
			return buf.String(), nil
		default:
			t.skip(1)
		}
	}
}
