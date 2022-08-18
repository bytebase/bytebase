package parser

import (
	"strings"
	"unicode"

	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/pkg/errors"
)

const (
	eofRune = rune(-1)
)

var (
	beginRuneList  = []rune{'B', 'E', 'G', 'I', 'N'}
	atomicRuneList = []rune{'A', 'T', 'M', 'I', 'C'}
)

type tokenizer struct {
	statement []rune
	cursor    uint
	len       uint
	line      int
	startLine int
}

// newTokenizer creates a new tokenizer.
// Notice: we append an additional eofRune in the statement. This is a sentinel rune.
func newTokenizer(statement string) *tokenizer {
	t := &tokenizer{
		statement: []rune(statement),
		cursor:    0,
		line:      1,
		startLine: 1,
	}
	t.len = uint(len(t.statement))
	// append an additional eofRune.
	t.statement = append(t.statement, eofRune)
	return t
}

// setLineForCreateTableStmt sets the line for columns and table constraints in CREATE TABLE statements.
func (t *tokenizer) setLineForCreateTableStmt(node *ast.CreateTableStmt) error {
	// We assume that the parser will parse the columns and table constraints according to the order of the raw SQL statements
	// and the identifiers don't equal any keywords in CREATE TABLE statements.
	// If it breaks our assumption, we set the line for columns and table constraints to the first line of the CREATE TABLE statement.
	for _, col := range node.ColumnList {
		col.SetLine(node.Line())
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
				} else if constraintPos < len(node.ConstraintList) &&
					matchTableConstraint(def, node.ConstraintList[constraintPos]) {
					node.ConstraintList[constraintPos].SetLine(t.startLine + node.Line() - 1)
				}
				return nil
			}
			t.skip(1)
		case ',':
			def := strings.ToLower(t.getString(startPos, t.pos()-startPos))
			if columnPos < len(node.ColumnList) &&
				strings.Contains(def, strings.ToLower(node.ColumnList[columnPos].ColumnName)) {
				node.ColumnList[columnPos].SetLine(t.startLine + node.Line() - 1)
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

// splitPostgreSQLMultiSQL splits the statement to a string slice.
// We mainly considered:
//   - comments
//     - style /* comments */
//     - style -- comments
//   - string
//     - style 'string'
//     - style $$ string $$
//   - identifier
//     - style "indentifier"
//
// Notice:
//   - We support PostgreSQL CREATE PROCEDURE statement with $$ $$ style,
//       but do not support BEGIN ATOMIC ... END; style.
//       See https://www.postgresql.org/docs/14/sql-createprocedure.html.
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
			res = append(res, SingleSQL{
				Text: t.getString(startPos, t.pos()-startPos),
				Line: t.startLine,
			})
			t.skipBlank()
			t.startLine = t.line
			startPos = t.pos()
		case t.char(0) == eofRune:
			s := t.getString(startPos, t.pos())
			if !emptyString(s) {
				res = append(res, SingleSQL{
					Text: s,
					Line: t.startLine,
				})
			}
			return res, nil
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

// There are two ways to include a single quote('), using \' or ''.
// We only handle the case \', because the second case '' does not require special handling.
// In more detail, we can think of
//     'this is a string contains ''.'
// as
//     'this is a string contains '
// and
//     '.'
//.
// And this is extensible.
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
		for {
			switch t.char(0) {
			case '\n':
				t.line++
				t.skip(1)
				return nil
			case eofRune:
				return nil
			default:
				t.skip(1)
			}
		}
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

func (t *tokenizer) char(after uint) rune {
	if t.cursor+after >= t.len {
		return eofRune
	}

	return t.statement[t.cursor+after]
}

func (t *tokenizer) skip(step uint) {
	t.cursor += step
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
	return t.statement[startPos:endPos]
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
