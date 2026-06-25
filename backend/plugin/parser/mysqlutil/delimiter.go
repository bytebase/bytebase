package mysqlutil

import (
	"strings"

	"github.com/pkg/errors"
)

// DealWithDelimiter removes client-side DELIMITER directives and converts
// statements terminated by a custom delimiter back to semicolon-terminated SQL.
func DealWithDelimiter(statement string) (string, error) {
	var result strings.Builder
	delimiter := ";"
	state := lexicalNormal
	atStatementStart := true
	seenDirective := false
	for _, line := range splitLinesAfter(statement) {
		if state == lexicalNormal && atStatementStart {
			nextDelimiter, ok, err := parseDelimiterDirective(line)
			if err != nil {
				return "", err
			}
			if ok {
				delimiter = nextDelimiter
				seenDirective = true
				result.WriteString(placeholderLine(line))
				continue
			}
		}
		writeLineWithDelimiter(&result, line, delimiter, &state, &atStatementStart)
	}
	if !seenDirective {
		return statement, nil
	}
	return result.String(), nil
}

func splitLinesAfter(statement string) []string {
	if statement == "" {
		return nil
	}

	var lines []string
	start := 0
	for i := 0; i < len(statement); i++ {
		if statement[i] == '\n' {
			lines = append(lines, statement[start:i+1])
			start = i + 1
		}
	}
	if start < len(statement) {
		lines = append(lines, statement[start:])
	}
	return lines
}

func lineBreakSuffix(line string) string {
	if strings.HasSuffix(line, "\r\n") {
		return "\r\n"
	}
	if strings.HasSuffix(line, "\n") {
		return "\n"
	}
	return ""
}

func placeholderLine(line string) string {
	lineBreak := lineBreakSuffix(line)
	return strings.Repeat(" ", len(line)-len(lineBreak)) + lineBreak
}

func parseDelimiterDirective(line string) (string, bool, error) {
	line = strings.TrimRight(line, "\r\n")
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t' || line[i] == '\r') {
		i++
	}

	if !isDelimiterKeyword(line[i:]) {
		return "", false, nil
	}
	i += len("DELIMITER")
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	start := i
	for i < len(line) && line[i] != ' ' && line[i] != '\t' && line[i] != '\r' {
		if line[i] == '\\' {
			return "", true, errors.Errorf("cannot extract delimiter from %q", line)
		}
		i++
	}
	if start == i {
		return "", true, errors.Errorf("cannot extract delimiter from %q", line)
	}
	return line[start:i], true, nil
}

func isDelimiterKeyword(s string) bool {
	const delimiter = "DELIMITER"
	if len(s) < len(delimiter) {
		return false
	}
	for i := range delimiter {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if c != delimiter[i] {
			return false
		}
	}
	return len(s) == len(delimiter) || s[len(delimiter)] == ' ' || s[len(delimiter)] == '\t'
}

type lexicalState int

const (
	lexicalNormal lexicalState = iota
	lexicalSingleQuote
	lexicalDoubleQuote
	lexicalBacktick
	lexicalBlockComment
)

func writeLineWithDelimiter(result *strings.Builder, line, delimiter string, state *lexicalState, atStatementStart *bool) {
	for i := 0; i < len(line); {
		switch *state {
		case lexicalSingleQuote:
			result.WriteByte(line[i])
			switch line[i] {
			case '\\':
				if i+1 < len(line) {
					result.WriteByte(line[i+1])
					i += 2
					continue
				}
			case '\'':
				if i+1 < len(line) && line[i+1] == '\'' {
					result.WriteByte(line[i+1])
					i += 2
					continue
				}
				*state = lexicalNormal
			default:
			}
			i++
		case lexicalDoubleQuote:
			result.WriteByte(line[i])
			switch line[i] {
			case '\\':
				if i+1 < len(line) {
					result.WriteByte(line[i+1])
					i += 2
					continue
				}
			case '"':
				if i+1 < len(line) && line[i+1] == '"' {
					result.WriteByte(line[i+1])
					i += 2
					continue
				}
				*state = lexicalNormal
			default:
			}
			i++
		case lexicalBacktick:
			result.WriteByte(line[i])
			if line[i] == '`' {
				if i+1 < len(line) && line[i+1] == '`' {
					result.WriteByte(line[i+1])
					i += 2
					continue
				}
				*state = lexicalNormal
			}
			i++
		case lexicalBlockComment:
			if strings.HasPrefix(line[i:], "*/") {
				result.WriteString("*/")
				i += len("*/")
				*state = lexicalNormal
				continue
			}
			result.WriteByte(line[i])
			i++
		default:
			switch {
			case line[i] == '\'':
				*state = lexicalSingleQuote
				*atStatementStart = false
				result.WriteByte(line[i])
				i++
			case line[i] == '"':
				*state = lexicalDoubleQuote
				*atStatementStart = false
				result.WriteByte(line[i])
				i++
			case line[i] == '`':
				*state = lexicalBacktick
				*atStatementStart = false
				result.WriteByte(line[i])
				i++
			case strings.HasPrefix(line[i:], "--") || line[i] == '#':
				result.WriteString(line[i:])
				return
			case strings.HasPrefix(line[i:], "/*"):
				*state = lexicalBlockComment
				result.WriteString("/*")
				i += len("/*")
			case delimiter != ";" && delimiter != "" && strings.HasPrefix(line[i:], delimiter):
				result.WriteString(";")
				result.WriteString(strings.Repeat(" ", len(delimiter)-1))
				i += len(delimiter)
				*atStatementStart = true
			default:
				result.WriteByte(line[i])
				if line[i] == ';' {
					*atStatementStart = true
				} else if line[i] != ' ' && line[i] != '\t' && line[i] != '\r' && line[i] != '\n' {
					*atStatementStart = false
				}
				i++
			}
		}
	}
}
