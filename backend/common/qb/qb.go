package qb

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// QueryPart represents a segment of a SQL query with text, parameters, and errors.
type QueryPart struct {
	Text   string
	Params []any
	Errs   []error
}

// Query builds SQL queries safely without string concatenation of user data.
// It stays close to raw SQL - you write SQL fragments with ? placeholders,
// and it handles converting them to PostgreSQL's $1, $2, $3 format.
type Query struct {
	parts []QueryPart
}

// Q creates a new empty Query.
func Q() *Query {
	return &Query{}
}

// Join adds a query part with a custom separator.
// The separator is prepended to the text (except for the first part).
func (q *Query) Join(separator, text string, params ...any) *Query {
	if q == nil {
		q = Q()
	}
	// Prepend separator if not the first part
	if len(q.parts) > 0 {
		text = separator + text
	}
	q.parts = append(q.parts, makePart(text, params...))
	return q
}

// makePart creates a QueryPart, expanding any *Query parameters inline.
func makePart(text string, params ...any) QueryPart {
	var newParams []any
	var errs []error

	// Use unique delimiters to avoid conflicts with nested query ? characters
	const placeholder = "\x00QB_PLACEHOLDER\x00"
	const escapeMarker = "\x00QB_ESCAPE\x00"

	// First, temporarily replace ?? (escaped ?) with a marker to preserve it
	text = strings.ReplaceAll(text, "??", escapeMarker)

	// Then, mark all remaining single ? placeholders with a unique marker
	// This prevents nested queries' ? characters from interfering with subsequent replacements
	text = strings.ReplaceAll(text, "?", placeholder)

	for _, param := range params {
		// Check if the parameter is a *Query
		if nestedQuery, ok := param.(*Query); ok {
			// Expand the nested query inline - use toRawSQL which keeps ? placeholders
			nestedSQL, nestedParams, err := nestedQuery.toRawSQL()
			if err != nil {
				errs = append(errs, errors.Wrap(err, "failed to expand nested query"))
				continue
			}
			// Replace the first placeholder with the nested SQL
			text = strings.Replace(text, placeholder, nestedSQL, 1)
			// Add the nested params to our param list
			newParams = append(newParams, nestedParams...)
		} else {
			// Regular parameter - replace placeholder with ?
			text = strings.Replace(text, placeholder, "?", 1)
			newParams = append(newParams, param)
		}
	}

	// Check for leftover placeholders (more ? than parameters provided)
	if strings.Contains(text, placeholder) {
		errs = append(errs, errors.New("mismatched parameters: more ? placeholders than parameters provided"))
	}

	// Restore ?? escape sequences
	text = strings.ReplaceAll(text, escapeMarker, "??")

	return QueryPart{
		Text:   text,
		Params: newParams,
		Errs:   errs,
	}
}

// toRawSQL concatenates all parts and returns SQL with ? placeholders (not $1, $2, etc).
// This is used internally for query composition.
func (q *Query) toRawSQL() (string, []any, error) {
	if q == nil {
		return "", nil, errors.New("cannot generate SQL from nil Query")
	}

	var sqlBuilder strings.Builder
	params := make([]any, 0)

	for _, part := range q.parts {
		// Check for errors in this part
		if len(part.Errs) > 0 {
			// Return the first error encountered
			return "", nil, part.Errs[0]
		}
		sqlBuilder.WriteString(part.Text)
		params = append(params, part.Params...)
	}

	return sqlBuilder.String(), params, nil
}

// Space adds a query part with a space separator. Convenience wrapper for Join(" ", ...).
func (q *Query) Space(text string, params ...any) *Query {
	return q.Join(" ", text, params...)
}

// And adds an AND condition.
func (q *Query) And(text string, params ...any) *Query {
	return q.Join(" AND ", text, params...)
}

// Or adds an OR condition.
func (q *Query) Or(text string, params ...any) *Query {
	return q.Join(" OR ", text, params...)
}

// Comma adds a query part with a comma separator. Convenience wrapper for Join(", ", ...).
func (q *Query) Comma(text string, params ...any) *Query {
	return q.Join(", ", text, params...)
}

// Where adds a WHERE clause.
func (q *Query) Where(text string, params ...any) *Query {
	return q.Join(" WHERE ", text, params...)
}

// Len returns the number of parts in the query.
func (q *Query) Len() int {
	if q == nil {
		return 0
	}
	return len(q.parts)
}

// ToSQL generates PostgreSQL-compatible SQL with $1, $2, ... placeholders.
// Returns the SQL string, parameters slice, and any error.
//
// Placeholder rules:
// - Single ? → parameter placeholder (converted to $1, $2, etc.)
// - Double ?? → literal ? in SQL (for PostgreSQL JSONB operators like ?, ?|, ?&)
func (q *Query) ToSQL() (string, []any, error) {
	if q == nil {
		return "", nil, errors.New("cannot generate SQL from nil Query")
	}

	// First pass: concatenate all parts and collect params
	var sqlBuilder strings.Builder
	params := make([]any, 0)

	for _, part := range q.parts {
		// Check for errors in this part
		if len(part.Errs) > 0 {
			// Return the first error encountered
			return "", nil, part.Errs[0]
		}
		sqlBuilder.WriteString(part.Text)
		params = append(params, part.Params...)
	}

	sql := sqlBuilder.String()

	// Second pass: replace ? placeholders with $1, $2, etc., handling ?? escape sequence
	// Use a unique delimiter that won't appear in SQL
	const delimiter = "\x00"
	// Replace ?? with delimiter temporarily
	sql = strings.ReplaceAll(sql, "??", delimiter)
	// Split on remaining ? (these are all placeholders)
	parts := strings.Split(sql, "?")

	// Verify parameter count matches
	if len(parts)-1 != len(params) {
		return "", nil, errors.New("mismatched parameters: ? placeholders count does not match params count")
	}

	// Build final SQL with $1, $2, etc.
	var builder strings.Builder
	for i := range params {
		builder.WriteString(parts[i])
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(i + 1))
	}
	builder.WriteString(parts[len(parts)-1])

	// Replace delimiter back to ?
	finalSQL := strings.ReplaceAll(builder.String(), delimiter, "?")

	return finalSQL, params, nil
}
