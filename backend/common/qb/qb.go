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

	for _, param := range params {
		// Check if the parameter is a *Query
		if nestedQuery, ok := param.(*Query); ok {
			// Expand the nested query inline - use toRawSql which keeps ? placeholders
			nestedSQL, nestedParams, err := nestedQuery.toRawSQL()
			if err != nil {
				errs = append(errs, errors.Wrap(err, "failed to expand nested query"))
				continue
			}
			// Replace the first ? with the nested SQL
			text = strings.Replace(text, "?", nestedSQL, 1)
			// Add the nested params to our param list
			newParams = append(newParams, nestedParams...)
		} else {
			// Regular parameter
			newParams = append(newParams, param)
		}
	}

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

	// Second pass: replace ? placeholders with $1, $2, etc.
	sqlParts := strings.Split(sql, "?")
	if len(sqlParts)-1 != len(params) {
		return "", nil, errors.New("mismatched parameters: ? placeholders count does not match params count")
	}

	var builder strings.Builder
	for i := range params {
		builder.WriteString(sqlParts[i])
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(i + 1))
	}
	builder.WriteString(sqlParts[len(sqlParts)-1])

	return builder.String(), params, nil
}
