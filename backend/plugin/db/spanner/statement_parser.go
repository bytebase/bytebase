package spanner

import (
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ddlStatements = map[string]bool{"CREATE": true, "DROP": true, "ALTER": true}
var selectStatements = map[string]bool{"WITH": true, "SELECT": true}

// RemoveCommentsAndTrim removes any comments in the query string and trims any
// spaces at the beginning and end of the query. This makes checking what type
// of query a string is a lot easier, as only the first word(s) need to be
// checked after this has been removed.
// source: https://github.com/googleapis/go-sql-spanner/blob/e33bd23e1ebfa2fe1b947bced9eacdc6454595eb/statement_parser.go
func removeCommentsAndTrim(sql string) (string, error) {
	const singleQuote = '\''
	const doubleQuote = '"'
	const backtick = '`'
	const hyphen = '-'
	const dash = '#'
	const slash = '/'
	const asterisk = '*'
	isInQuoted := false
	isInSingleLineComment := false
	isInMultiLineComment := false
	var startQuote rune
	lastCharWasEscapeChar := false
	isTripleQuoted := false
	res := strings.Builder{}
	res.Grow(len(sql))
	index := 0
	runes := []rune(sql)
	for index < len(runes) {
		c := runes[index]
		if isInQuoted {
			if (c == '\n' || c == '\r') && !isTripleQuoted {
				return "", spanner.ToSpannerError(status.Errorf(codes.InvalidArgument, "statement contains an unclosed literal: %s", sql))
			} else if c == startQuote {
				if lastCharWasEscapeChar {
					lastCharWasEscapeChar = false
				} else if isTripleQuoted {
					if len(runes) > index+2 && runes[index+1] == startQuote && runes[index+2] == startQuote {
						isInQuoted = false
						startQuote = 0
						isTripleQuoted = false
						_, _ = res.WriteRune(c)
						_, _ = res.WriteRune(c)
						index += 2
					}
				} else {
					isInQuoted = false
					startQuote = 0
				}
			} else if c == '\\' {
				lastCharWasEscapeChar = true
			} else {
				lastCharWasEscapeChar = false
			}
			_, _ = res.WriteRune(c)
		} else {
			// We are not in a quoted string.
			if isInSingleLineComment {
				if c == '\n' {
					isInSingleLineComment = false
					// Include the line feed in the result.
					_, _ = res.WriteRune(c)
				}
			} else if isInMultiLineComment {
				if len(runes) > index+1 && c == asterisk && runes[index+1] == slash {
					isInMultiLineComment = false
					index++
				}
			} else {
				if c == dash || (len(runes) > index+1 && c == hyphen && runes[index+1] == hyphen) {
					// This is a single line comment.
					isInSingleLineComment = true
				} else if len(runes) > index+1 && c == slash && runes[index+1] == asterisk {
					isInMultiLineComment = true
					index++
				} else {
					if c == singleQuote || c == doubleQuote || c == backtick {
						isInQuoted = true
						startQuote = c
						// Check whether it is a triple-quote.
						if len(runes) > index+2 && runes[index+1] == startQuote && runes[index+2] == startQuote {
							isTripleQuoted = true
							_, _ = res.WriteRune(c)
							_, _ = res.WriteRune(c)
							index += 2
						}
					}
					_, _ = res.WriteRune(c)
				}
			}
		}
		index++
	}
	if isInQuoted {
		return "", spanner.ToSpannerError(status.Errorf(codes.InvalidArgument, "statement contains an unclosed literal: %s", sql))
	}
	trimmed := strings.TrimSpace(res.String())
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == ';' {
		return trimmed[:len(trimmed)-1], nil
	}
	return trimmed, nil
}

func splitStatement(sql string) ([]string, error) {
	var stmts []string
	const singleQuote = '\''
	const doubleQuote = '"'
	const backtick = '`'
	const delimiter = ';'
	isInQuoted := false
	var startQuote rune
	lastCharWasEscapeChar := false
	isTripleQuoted := false
	res := strings.Builder{}
	res.Grow(len(sql))
	index := 0
	runes := []rune(sql)
	for index < len(runes) {
		c := runes[index]
		if isInQuoted {
			if (c == '\n' || c == '\r') && !isTripleQuoted {
				return nil, spanner.ToSpannerError(status.Errorf(codes.InvalidArgument, "statement contains an unclosed literal: %s", sql))
			} else if c == startQuote {
				if lastCharWasEscapeChar {
					lastCharWasEscapeChar = false
				} else if isTripleQuoted {
					if len(runes) > index+2 && runes[index+1] == startQuote && runes[index+2] == startQuote {
						isInQuoted = false
						startQuote = 0
						isTripleQuoted = false
						_, _ = res.WriteRune(c)
						_, _ = res.WriteRune(c)
						index += 2
					}
				} else {
					isInQuoted = false
					startQuote = 0
				}
			} else if c == '\\' {
				lastCharWasEscapeChar = true
			} else {
				lastCharWasEscapeChar = false
			}
			_, _ = res.WriteRune(c)
		} else {
			// We are not in a quoted string.
			if c == singleQuote || c == doubleQuote || c == backtick {
				isInQuoted = true
				startQuote = c
				// Check whether it is a triple-quote.
				if len(runes) > index+2 && runes[index+1] == startQuote && runes[index+2] == startQuote {
					isTripleQuoted = true
					_, _ = res.WriteRune(c)
					_, _ = res.WriteRune(c)
					index += 2
				}
				_, _ = res.WriteRune(c)
			} else if c == delimiter {
				stmt := strings.Trim(res.String(), " \n\t")
				if stmt != "" {
					stmts = append(stmts, stmt)
				}
				res.Reset()
				res.Grow(len(sql))
			} else {
				_, _ = res.WriteRune(c)
			}
		}
		index++
	}
	if isInQuoted {
		return nil, spanner.ToSpannerError(status.Errorf(codes.InvalidArgument, "statement contains an unclosed literal: %s", sql))
	}
	if res.Len() > 0 {
		stmt := strings.Trim(res.String(), " \n\t")
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
	}
	return stmts, nil
}

// sanitizeSQL removes comments, splits the sql by `;` and returns the trimmed sql statement array.
func sanitizeSQL(sql string) ([]string, error) {
	query, err := removeCommentsAndTrim(sql)
	if err != nil {
		return nil, err
	}
	stmts, err := splitStatement(query)
	if err != nil {
		return nil, err
	}
	return stmts, nil
}

// isDDL returns true if the given sql string is a DDL statement.
func isDDL(query string) bool {
	for ddl := range ddlStatements {
		if len(query) >= len(ddl) && strings.EqualFold(query[:len(ddl)], ddl) {
			return true
		}
	}
	return false
}

// isSelect returns true if the given sql string is a SELECT statement.
func isSelect(query string) bool {
	for keyword := range selectStatements {
		if len(query) >= len(keyword) && strings.EqualFold(query[:len(keyword)], keyword) {
			return true
		}
	}
	return false
}
