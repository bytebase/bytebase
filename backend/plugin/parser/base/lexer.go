package base

// CalculateLineAndColumn calculates the 0-based line number and 0-based column (character offset)
// for a given byte offset in the statement.
func CalculateLineAndColumn(statement string, byteOffset int) (line, column int) {
	if byteOffset > len(statement) {
		byteOffset = len(statement)
	}
	// Range over string iterates over runes (code points), not bytes.
	// \r\n is treated as a single line break; standalone \r is a line break.
	s := statement[:byteOffset]
	for i, r := range s {
		switch r {
		case '\r':
			// \r\n is one line break; standalone \r is also a line break.
			// In both cases, increment line here. For \r\n, the \n case
			// below will see that the previous char was \r and skip.
			line++
			column = 0
		case '\n':
			if i > 0 && s[i-1] == '\r' {
				// Already counted by the \r above.
				continue
			}
			line++
			column = 0
		default:
			column++
		}
	}
	return line, column
}
