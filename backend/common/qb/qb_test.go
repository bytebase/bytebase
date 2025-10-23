package qb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuery_Basic(t *testing.T) {
	q := Q().Space("SELECT * FROM users")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users", sql)
	require.Empty(t, args)
}

func TestQuery_WithParameters(t *testing.T) {
	q := Q().Space("SELECT * FROM users").Where("id = ?", 123)
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE id = $1", sql)
	require.Equal(t, []any{123}, args)
}

func TestQuery_MultipleParameters(t *testing.T) {
	q := Q().Space("SELECT * FROM users").
		Where("id = ?", 123).
		And("active = ?", true).
		And("role = ?", "admin")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE id = $1 AND active = $2 AND role = $3", sql)
	require.Equal(t, []any{123, true, "admin"}, args)
}

func TestQuery_AndOr(t *testing.T) {
	q := Q().Space("SELECT * FROM users").
		Where("status = ?", "active").
		And("(role = ?", "admin").
		Or("role = ?)", "moderator")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE status = $1 AND (role = $2 OR role = $3)", sql)
	require.Equal(t, []any{"active", "admin", "moderator"}, args)
}

func TestQuery_Join(t *testing.T) {
	q := Q().Space("SELECT").
		Join(", ", "id").
		Join(", ", "name").
		Join(", ", "email").
		Space("FROM users")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT, id, name, email FROM users", sql)
	require.Empty(t, args)
}

func TestQuery_JoinWithParams(t *testing.T) {
	q := Q().Space("INSERT INTO users (name, email) VALUES").
		Join(", ", "(?, ?)", "Alice", "alice@example.com").
		Join(", ", "(?, ?)", "Bob", "bob@example.com")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "INSERT INTO users (name, email) VALUES, ($1, $2), ($3, $4)", sql)
	require.Equal(t, []any{"Alice", "alice@example.com", "Bob", "bob@example.com"}, args)
}

func TestQuery_NoParameters(t *testing.T) {
	q := Q().Space("SELECT * FROM users").
		Where("TRUE").
		And("deleted_at IS NULL")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE TRUE AND deleted_at IS NULL", sql)
	require.Empty(t, args)
}

func TestQuery_ComplexConditions(t *testing.T) {
	q := Q().Space("SELECT * FROM revision").
		Where("TRUE").
		And("instance = ?", "prod").
		And("db_name = ?", "main").
		And("payload->>'type' = ?", "MIGRATE").
		And("deleted_at IS NULL").
		Space("ORDER BY version DESC").
		Space("LIMIT ?", 10).
		Space("OFFSET ?", 20)
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM revision WHERE TRUE AND instance = $1 AND db_name = $2 AND payload->>'type' = $3 AND deleted_at IS NULL ORDER BY version DESC LIMIT $4 OFFSET $5", sql)
	require.Equal(t, []any{"prod", "main", "MIGRATE", 10, 20}, args)
}

func TestQuery_ArrayParameter(t *testing.T) {
	versions := []string{"v1", "v2", "v3"}
	q := Q().Space("SELECT * FROM revision").
		Where("version = ANY(?)", versions)
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM revision WHERE version = ANY($1)", sql)
	require.Equal(t, []any{versions}, args)
}

func TestQuery_NilQuery(t *testing.T) {
	var q *Query
	q = q.Space("SELECT * FROM users")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users", sql)
	require.Empty(t, args)
}

func TestQuery_EmptyQuery(t *testing.T) {
	q := Q()
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "", sql)
	require.Empty(t, args)
}

func TestQuery_MismatchedParameters(t *testing.T) {
	q := Q().Space("SELECT * FROM users WHERE id = ? AND name = ?", 123)
	_, _, err := q.ToSQL()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mismatched parameters")
}

func TestQuery_MultilineSQL(t *testing.T) {
	q := Q().Space(`
		SELECT
			id,
			name,
			email
		FROM users
	`).Where("id = ?", 123)
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Contains(t, sql, "SELECT")
	require.Contains(t, sql, "WHERE id = $1")
	require.Equal(t, []any{123}, args)
}

func TestQuery_ChainedCalls(t *testing.T) {
	// Test that we can chain calls continuously
	sql, args, err := Q().
		Space("SELECT * FROM users").
		Where("status = ?", "active").
		And("role = ?", "admin").
		Space("ORDER BY created_at DESC").
		ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE status = $1 AND role = $2 ORDER BY created_at DESC", sql)
	require.Equal(t, []any{"active", "admin"}, args)
}

func TestQuery_ConditionalBuilding(t *testing.T) {
	// Simulate building a query with optional filters
	id := 123
	name := "Alice"
	var status *string // nil

	q := Q().Space("SELECT * FROM users").Where("TRUE")

	if id != 0 {
		q.And("id = ?", id)
	}
	if name != "" {
		q.And("name = ?", name)
	}
	if status != nil { //nolint:govet
		q.And("status = ?", *status)
	}

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE TRUE AND id = $1 AND name = $2", sql)
	require.Equal(t, []any{123, "Alice"}, args)
}

func TestQuery_Composition(t *testing.T) {
	// Build query parts separately
	sel := Q().Space("SELECT *")
	from := Q().Space("FROM users")
	where := Q().Space("WHERE id = ?", 123)

	// Compose them together
	q := Q().Space("? ? ?", sel, from, where).Space("LIMIT ?", 10)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE id = $1 LIMIT $2", sql)
	require.Equal(t, []any{123, 10}, args)
}

func TestQuery_ConditionalComposition(t *testing.T) {
	getName := true
	getID := false
	filterAdult := true
	ageCheck := true
	filterChild := false

	// Build SELECT clause
	sel := Q().Space("SELECT")
	if getName {
		sel.Join(", ", "name")
	}
	if getID {
		sel.Join(", ", "id")
	}
	if !getName && !getID {
		sel.Join(", ", "*")
	}

	from := Q().Space("FROM my_table")

	// Build WHERE clause
	where := Q()
	hasConditions := false

	if filterAdult {
		adultCond := Q().Space("name = ?", "adult")
		if ageCheck {
			adultCond.And("age > ?", 20)
		}
		where.Space("WHERE (?)", adultCond)
		hasConditions = true
	}

	if filterChild {
		if hasConditions {
			where.Or("(name = ? AND age < ?)", "youth", 21)
		} else {
			where.Space("WHERE (name = ? AND age < ?)", "youth", 21)
		}
	}

	// Compose final query
	q := Q().Space("? ? ?", sel, from, where).Space("LIMIT ?", 10)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT, name FROM my_table WHERE (name = $1 AND age > $2) LIMIT $3", sql)
	require.Equal(t, []any{"adult", 20, 10}, args)
}

func TestQuery_NestedComposition(t *testing.T) {
	// Build a subquery
	subquery := Q().Space("SELECT user_id FROM orders WHERE total > ?", 100)

	// Use it in main query
	q := Q().Space("SELECT * FROM users").
		Where("id IN (?)", subquery)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE total > $1)", sql)
	require.Equal(t, []any{100}, args)
}

func TestQuery_PracticalComposition(t *testing.T) {
	// Realistic example: build a complex query with reusable parts
	getUser := true
	getEmail := true
	filterActive := true
	filterRole := "admin"

	// Build SELECT clause dynamically
	sel := Q().Space("SELECT id")
	if getUser {
		sel.Join(", ", "name")
	}
	if getEmail {
		sel.Join(", ", "email")
	}

	// Build WHERE clause
	where := Q()
	if filterActive {
		where.Space("WHERE active = ?", true)
	}
	if filterRole != "" {
		if where.Len() > 0 {
			where.And("role = ?", filterRole)
		} else {
			where.Space("WHERE role = ?", filterRole)
		}
	}

	// Compose final query
	q := Q().Space("? FROM users ?", sel, where).
		Space("ORDER BY created_at DESC").
		Space("LIMIT ?", 10)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT id, name, email FROM users WHERE active = $1 AND role = $2 ORDER BY created_at DESC LIMIT $3", sql)
	require.Equal(t, []any{true, "admin", 10}, args)
}

func TestQuery_ErrorHandling(t *testing.T) {
	// Create a query with nil nested query to trigger error
	var nilQuery *Query
	q := Q().Space("SELECT * FROM users WHERE id IN (?)", nilQuery)

	_, _, err := q.ToSQL()
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil Query")
}

func TestQuery_JSONBExistsOperator(t *testing.T) {
	// Test the ? operator (key existence check)
	q := Q().Space("SELECT * FROM tasks").
		Where("payload ?? 'schema_version'")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM tasks WHERE payload ? 'schema_version'", sql)
	require.Empty(t, args)
}

func TestQuery_JSONBExistsAnyOperator(t *testing.T) {
	// Test the ?| operator (any key exists)
	q := Q().Space("DELETE FROM issue_comment").
		Where("(payload->'taskUpdate')??|'{toEarliestAllowedTime, fromEarliestAllowedTime}'")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "DELETE FROM issue_comment WHERE (payload->'taskUpdate')?|'{toEarliestAllowedTime, fromEarliestAllowedTime}'", sql)
	require.Empty(t, args)
}

func TestQuery_JSONBExistsAllOperator(t *testing.T) {
	// Test the ?& operator (all keys exist)
	q := Q().Space("SELECT * FROM issues").
		Where("payload->'labels' ??& ?::TEXT[]", []string{"bug", "critical"})
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM issues WHERE payload->'labels' ?& $1::TEXT[]", sql)
	require.Equal(t, []any{[]string{"bug", "critical"}}, args)
}

func TestQuery_JSONBMixedOperators(t *testing.T) {
	// Test mixing JSONB operators with regular parameters
	q := Q().Space("SELECT * FROM tasks").
		Where("instance = ?", "prod").
		And("payload ?? 'type'").
		And("payload->>'status' = ?", "completed")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM tasks WHERE instance = $1 AND payload ? 'type' AND payload->>'status' = $2", sql)
	require.Equal(t, []any{"prod", "completed"}, args)
}

func TestQuery_JSONBComplexQuery(t *testing.T) {
	// Realistic example from the codebase
	q := Q().Space("SELECT * FROM revision").
		Where("TRUE").
		And("instance = ?", "prod").
		And("db_name = ?", "main").
		And("payload->>'type' = ?", "MIGRATE").
		And("payload ?? 'schema_version'").
		And("deleted_at IS NULL").
		Space("ORDER BY version DESC").
		Space("LIMIT ?", 10)
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM revision WHERE TRUE AND instance = $1 AND db_name = $2 AND payload->>'type' = $3 AND payload ? 'schema_version' AND deleted_at IS NULL ORDER BY version DESC LIMIT $4", sql)
	require.Equal(t, []any{"prod", "main", "MIGRATE", 10}, args)
}

func TestQuery_EscapedQuestionMarks(t *testing.T) {
	// Test multiple ?? in a row - need to escape each ?
	q := Q().Space("SELECT '??????' as question_marks")
	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT '???' as question_marks", sql)
	require.Empty(t, args)
}

func TestQuery_JSONBWithComposition(t *testing.T) {
	// Test JSONB operators with query composition
	labelFilter := Q().Space("payload->'labels' ??& ?::TEXT[]", []string{"important"})
	typeFilter := Q().Space("payload ?? 'type'")

	q := Q().Space("SELECT * FROM issues").
		Where("?", labelFilter).
		And("?", typeFilter).
		And("status = ?", "open")

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM issues WHERE payload->'labels' ?& $1::TEXT[] AND payload ? 'type' AND status = $2", sql)
	require.Equal(t, []any{[]string{"important"}, "open"}, args)
}

func TestQuery_Comma(t *testing.T) {
	// Build a SET clause with comma separators
	set := Q()
	set.Comma("name = ?", "Alice")
	set.Comma("email = ?", "alice@example.com")
	set.Comma("active = ?", true)

	sql, args, err := set.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "name = $1, email = $2, active = $3", sql)
	require.Equal(t, []any{"Alice", "alice@example.com", true}, args)
}

func TestQuery_CommaInUpdate(t *testing.T) {
	// Realistic UPDATE with conditional SET fields
	name := "Alice"
	active := true

	set := Q()
	if name != "" {
		set.Comma("name = ?", name)
	}
	set.Comma("active = ?", active)

	q := Q().Space("UPDATE users SET ?", set).
		Space("WHERE id = ?", 123)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "UPDATE users SET name = $1, active = $2 WHERE id = $3", sql)
	require.Equal(t, []any{"Alice", true, 123}, args)
}

func TestQuery_CommaWithNullHandling(t *testing.T) {
	// UPDATE with NULL assignment
	projectID := "proj-1"
	environmentID := "" // Empty string means NULL

	set := Q()
	set.Comma("project = ?", projectID)
	if environmentID == "" {
		set.Comma("environment = NULL")
	} else {
		set.Comma("environment = ?", environmentID)
	}

	q := Q().Space("UPDATE db SET ?", set).
		Space("WHERE instance = ? AND name = ?", "inst-1", "db-1")

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "UPDATE db SET project = $1, environment = NULL WHERE instance = $2 AND name = $3", sql)
	require.Equal(t, []any{"proj-1", "inst-1", "db-1"}, args)
}
