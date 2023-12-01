package batch

import "unicode"

/*
 * parse.go contains the logic of parsing a SQL batch.
 * The original code can be found at https://sourcegraph.com/github.com/microsoft/go-sqlcmd@fb920dc0da056e58969696128b440e2bf99c105b/-/blob/pkg/sqlcmd/batch.go?L76-170.
 */

// grab grabs i from r, or returns 0 if 1 >= end.
func grab(r []rune, i, end int) rune {
	if i < end {
		return r[i]
	}
	return 0
}

// readCommand reads to the next control character to find
// a command in the string. Command regexes constrain matches
// to the beginning of the string, and all commands consume
// an entire line.
func readCommand(r []rune, i, end int) (Command, int) {
	for ; i < end; i++ {
		next := grab(r, i, end)
		if next == 0 || (unicode.IsControl(next) && next != '\t') {
			break
		}
	}

	for _, builder := range commandBuilders {
		cmd := builder(string(r[:i]))
		if cmd != nil {
			return cmd, i
		}
	}
	return nil, i
}

// readMultilineComment finds the end of a multiline comment (ie, '*/').
func readMultilineComment(r []rune, i, end int) (int, bool) {
	i++
	for ; i < end; i++ {
		if r[i-1] == '*' && r[i] == '/' {
			return i, true
		}
	}
	return end, false
}
