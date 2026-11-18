package mysql

import "strings"

// UnescapeExpressionDefault unescapes the default expression for MySQL: the
// second layer of escaping MySQL applies when it stores expression defaults.
func UnescapeExpressionDefault(s string) string {
	s = strings.ReplaceAll(s, `\'`, `'`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// IsCurrentTimestampLike reports whether s names a current-time default.
func IsCurrentTimestampLike(s string) bool {
	upper := strings.ToUpper(s)
	return strings.HasPrefix(upper, "CURRENT_TIMESTAMP") || strings.HasPrefix(upper, "CURRENT_DATE")
}

func stripSingleQuote(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

// UnquoteMySQLString unescapes a string that MySQL's QUOTE() function escaped.
func UnquoteMySQLString(s string) string {
	if len(s) == 0 {
		return s
	}
	s = stripSingleQuote(s)
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\\':
				result = append(result, '\\')
				i++
			case '\'':
				result = append(result, '\'')
				i++
			case '0':
				result = append(result, 0)
				i++
			case 'Z':
				result = append(result, 0x1A)
				i++
			case 'n':
				result = append(result, '\n')
				i++
			case 't':
				result = append(result, '\t')
				i++
			case 'r':
				result = append(result, '\r')
				i++
			case 'b':
				result = append(result, '\b')
				i++
			default:
				result = append(result, s[i])
			}
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}
