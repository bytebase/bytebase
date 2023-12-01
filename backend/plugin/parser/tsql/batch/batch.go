package batch

import (
	"fmt"
	"regexp"
	"strconv"
)

/*
 * batch.go provides the ability to split a Transact-SQL script into batches.
 * It is based on the github.com/microsoft/go-sqlcmd package, we can move to
 * use it when it's documented and stable.
 * Currently, we references the code from https://sourcegraph.com/github.com/microsoft/go-sqlcmd@fb920dc0da056e58969696128b440e2bf99c105b/-/blob/pkg/sqlcmd/batch.go?L76-170.
 */

const (
	// minCapIncrease is the minimum number of bytes to grow the buffer by.
	minCapIncrease = 512
)

type Command interface {
	// isCommand is a dummy method to prevent users from implementing the Command interface.
	isCommand()
	// String returns the string representation of the command.
	String() string
}

type baseCommand struct{}

func (*baseCommand) isCommand() {}

var (
	// lineEnd is the slice to use when appending a line.
	lineEnd = []rune("\n")

	// commandsBuilder is a map of command names to functions that builds the command from input.
	commandBuilders = map[string]func(string) Command{
		"GO": buildGoCommand,
	}

	// goCommandMatcher matches a Go Command.
	goCommandMatcher = regexp.MustCompile(fmt.Sprintf(`(?im)^[\t ]*?%s(?:[ ]+(.*$)|$)`, regexp.QuoteMeta("GO")))
)

// GoCommand signals the end of a batch of Transact-SQL statements to the SQL Server Utilities.
// https://learn.microsoft.com/en-us/sql/t-sql/language-elements/sql-server-utilities-statements-go?view=sql-server-ver16
type GoCommand struct {
	baseCommand
	// Count is a positive integer, the batch preceding the GO command will execute the specified number of times.
	// The default value of Count is 1.
	Count uint
}

// String returns the string representation of the command.
func (g *GoCommand) String() string {
	return fmt.Sprintf("GO %d", g.Count)
}

func buildGoCommand(input string) Command {
	matches := goCommandMatcher.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}
	if len(matches) != 2 {
		return nil
	}

	var n int
	var err error
	if matches[1] == "" {
		n = 1
	} else {
		n, err = strconv.Atoi(matches[1])
		if err != nil {
			return nil
		}
		if n < 1 {
			return nil
		}
	}

	return &GoCommand{
		Count: uint(n),
	}
}

type Scan func() (string, error)

type Batch struct {
	// read provides the next chunk of runes.
	read Scan
	// buffer is the current batch text.
	buffer []rune
	// length is the length of the statement.
	length int
	// raw is the unprocessed runes.
	raw []rune
	// rawLen is the number of unprocessed rune.
	rawLen int
	// quote indicates currently processing a quoted string.
	quote rune
	// comment is the state of multi-line comment.
	comment bool
}

// NewBatch returns a new Batch.
func NewBatch(read Scan) *Batch {
	return &Batch{
		read: read,
	}
}

// String returns the current SQL batch next.
func (b *Batch) String() string {
	return string(b.buffer)
}

// Next returns the next command in the batch.
func (b *Batch) Next() (Command, error) {
	var i int

	if b.rawLen == 0 {
		s, err := b.read()
		if err != nil {
			return nil, err
		}
		b.raw = []rune(s)
		b.rawLen = len(b.raw)
	}

	var command Command
	var ok bool
	var scannedCommand bool

parse:
	for ; i < b.rawLen; i++ {
		c, next := b.raw[i], grab(b.raw, i+1, b.rawLen)
		switch {
		// we're in a quoted string
		case b.quote != 0:
			i, ok = b.readString(b.raw, i, b.rawLen, b.quote)
			if ok {
				b.quote = 0
			}
		// inside a multiline comment
		case b.comment:
			i, ok = readMultilineComment(b.raw, i, b.rawLen)
			b.comment = !ok
		// start of a string
		case c == '\'' || c == '"' || c == '[':
			b.quote = c
		// inline sql comment, skip to end of line
		case c == '-' && next == '-':
			i = b.rawLen
		// start a multi-line comment
		case c == '/' && next == '*':
			b.comment = true
			i++
		// continue processing quoted string or multiline comment
		case b.quote != 0 || b.comment:

		// We do not care about the variable reference for now.
		// Handle variable references
		// case c == '$' && next == '(':
		// 	vi, ok := readVariableReference(b.raw, i+2, b.rawLen)
		// 	if ok {
		// 		b.addVariableLocation(i, string(b.raw[i+2:vi]))
		// 		i = vi

		// 	} else {
		// 		err = syntaxError(b.linecount)
		// 		break parse
		// 	}

		// Commands have to be alone on the line
		case !scannedCommand:
			var cend int
			scannedCommand = true
			command, cend = readCommand(b.raw, i, b.rawLen)
			if command != nil {
				// remove the command from raw
				b.raw = append(b.raw[:i], b.raw[cend:]...)
				break parse
			}
		}
	}

	if command == nil {
		i = min(i, b.rawLen)
		b.append(b.raw[:i], lineEnd)
	}
	b.raw = b.raw[i:]
	b.rawLen = len(b.raw)

	return command, nil
}

// append appends r to b.Buffer separated by sep when b.Buffer is not already empty.
//
// Dynamically grows b.Buf as necessary to accommodate r and the separator.
// Specifically, when b.Buf is not empty, b.Buf will grow by increments of
// MinCapIncrease.
//
// After a call to append, b.Len will be len(b.Buf)+len(sep)+len(r). Call Reset
// to reset the Buf.
func (b *Batch) append(r, sep []rune) {
	rlen := len(r)
	// initial
	if b.buffer == nil {
		b.buffer, b.length = r, rlen
		return
	}
	blen, seplen := b.length, len(sep)
	tlen := blen + rlen + seplen
	// grow
	if bcap := cap(b.buffer); tlen > bcap {
		n := tlen + 2*rlen
		n += minCapIncrease - (n % minCapIncrease)
		z := make([]rune, blen, n)
		copy(z, b.buffer)
		b.buffer = z
	}
	b.buffer = b.buffer[:tlen]
	copy(b.buffer[blen:], sep)
	copy(b.buffer[blen+seplen:], r)
	b.length = tlen
}

// readString seeks to the end of a string returning the position and whether
// or not the string's end was found.
// If the string's terminator was not found, then the result will be the passed
// end.
func (*Batch) readString(r []rune, i, end int, quote rune) (int, bool) {
	var prev, c, next rune
	for ; i < end; i++ {
		c, next = r[i], grab(r, i+1, end)
		switch {
		case quote == '\'' && c == '\'' && next == '\'',
			quote == '[' && c == ']' && next == ']':
			i++
			continue
		case quote == '\'' && c == '\'' && prev != '\'',
			quote == '"' && c == '"',
			quote == '[' && c == ']':
			return i, true
		}
		prev = c
	}
	return end, false
}

// Reset clears the current batch text and replaces it with new runes.
func (b *Batch) Reset(r []rune) {
	b.buffer, b.length = nil, 0
	b.quote = 0
	b.comment = false
	if r != nil {
		b.raw, b.rawLen = r, len(r)
	} else {
		b.rawLen = 0
	}
}
