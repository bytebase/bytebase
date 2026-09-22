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

func TestQuery_Len(t *testing.T) {
	// Test Len() method
	q := Q()
	require.Equal(t, 0, q.Len())

	q.Space("SELECT * FROM users")
	require.Equal(t, 1, q.Len())

	q.Where("id = ?", 123)
	require.Equal(t, 2, q.Len())

	q.And("active = ?", true)
	require.Equal(t, 3, q.Len())

	// Nil query
	var nilQ *Query
	require.Equal(t, 0, nilQ.Len())
}

func TestQuery_DeeplyNestedComposition(t *testing.T) {
	// Test 3+ levels of nested queries
	innermost := Q().Space("SELECT category_id FROM products WHERE price > ?", 100)
	middle := Q().Space("SELECT user_id FROM orders WHERE product_id IN (?)", innermost)
	outer := Q().Space("SELECT * FROM users WHERE id IN (?)", middle)

	sql, args, err := outer.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE product_id IN (SELECT category_id FROM products WHERE price > $1))", sql)
	require.Equal(t, []any{100}, args)
}

func TestQuery_DeeplyNestedWithMultipleParams(t *testing.T) {
	// Test deeply nested with multiple parameters at each level
	level3 := Q().Space("SELECT id FROM items WHERE stock > ? AND category = ?", 10, "electronics")
	level2 := Q().Space("SELECT order_id FROM order_items WHERE item_id IN (?) AND quantity > ?", level3, 2)
	level1 := Q().Space("SELECT * FROM orders WHERE id IN (?) AND status = ?", level2, "completed")

	sql, args, err := level1.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM orders WHERE id IN (SELECT order_id FROM order_items WHERE item_id IN (SELECT id FROM items WHERE stock > $1 AND category = $2) AND quantity > $3) AND status = $4", sql)
	require.Equal(t, []any{10, "electronics", 2, "completed"}, args)
}

func TestQuery_Reuse(t *testing.T) {
	// Test reusing the same query object multiple times
	baseWhere := Q().Space("deleted_at IS NULL")

	// First use
	q1 := Q().Space("SELECT * FROM users WHERE ?", baseWhere).And("active = ?", true)
	sql1, args1, err := q1.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE deleted_at IS NULL AND active = $1", sql1)
	require.Equal(t, []any{true}, args1)

	// Second use - baseWhere should be reusable
	q2 := Q().Space("SELECT * FROM posts WHERE ?", baseWhere).And("published = ?", true)
	sql2, args2, err := q2.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM posts WHERE deleted_at IS NULL AND published = $1", sql2)
	require.Equal(t, []any{true}, args2)

	// baseWhere should still work independently
	sql3, args3, err := baseWhere.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "deleted_at IS NULL", sql3)
	require.Empty(t, args3)
}

func TestQuery_MixedEscapedAndPlaceholders(t *testing.T) {
	// Test mixing ?? (escaped) with ? (placeholders)
	q := Q().Space("SELECT * FROM data").
		Where("metadata ?? 'key'").
		And("value = ?", "test").
		And("tags ??| ?::TEXT[]", []string{"important"})

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM data WHERE metadata ? 'key' AND value = $1 AND tags ?| $2::TEXT[]", sql)
	require.Equal(t, []any{"test", []string{"important"}}, args)
}

func TestQuery_MultipleConsecutiveEscapes(t *testing.T) {
	// Test multiple ?? in various positions
	q := Q().Space("SELECT").
		Join(", ", "data ?? 'a'").
		Join(", ", "data ?? 'b'").
		Join(", ", "data ?? 'c'").
		Space("FROM table")

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT, data ? 'a', data ? 'b', data ? 'c' FROM table", sql)
	require.Empty(t, args)
}

func TestQuery_ComplexMixedOperators(t *testing.T) {
	// Complex query with mixed ??, ?, and nested queries
	subquery := Q().Space("SELECT id FROM items WHERE tags ??& ?::TEXT[]", []string{"urgent"})

	q := Q().Space("SELECT * FROM tasks").
		Where("metadata ?? 'version'").
		And("status = ?", "active").
		And("metadata ??| '{priority, deadline}'").
		And("item_id IN (?)", subquery).
		And("assignee = ?", "alice")

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM tasks WHERE metadata ? 'version' AND status = $1 AND metadata ?| '{priority, deadline}' AND item_id IN (SELECT id FROM items WHERE tags ?& $2::TEXT[]) AND assignee = $3", sql)
	require.Equal(t, []any{"active", []string{"urgent"}, "alice"}, args)
}

func TestQuery_ParameterOrdering(t *testing.T) {
	// Ensure parameters maintain correct order through chaining
	// Use separate Space/And calls since nested queries expand into first ? only
	part1 := Q().Space("a = ?", 1)
	part2 := Q().Space("b = ?", 2)
	part3 := Q().Space("c = ?", 3)

	q := Q().Space("SELECT * FROM test").
		Where("(?)", part1).
		And("(?)", part2).
		And("(?)", part3).
		And("d = ?", 4)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM test WHERE (a = $1) AND (b = $2) AND (c = $3) AND d = $4", sql)
	require.Equal(t, []any{1, 2, 3, 4}, args)
}

func TestQuery_EmptyNestedQuery(t *testing.T) {
	// Test empty nested query
	empty := Q()
	q := Q().Space("SELECT * FROM users WHERE ?", empty)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM users WHERE ", sql)
	require.Empty(t, args)
}

func TestQuery_MultipleNilQueries(t *testing.T) {
	// Test multiple nil nested queries trigger errors
	var nilQ1 *Query
	var nilQ2 *Query

	q := Q().Space("SELECT * FROM users WHERE id IN (?) OR status IN (?)", nilQ1, nilQ2)

	_, _, err := q.ToSQL()
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil Query")
}

func TestQuery_ComplexRealWorldExample(t *testing.T) {
	// Realistic complex query combining multiple patterns
	instanceID := "prod-db"
	filterDeleted := true
	filterStatus := "active"
	hasTypeFilter := true
	types := []string{"MIGRATE", "DATA"}

	// Build SELECT with JSONB operators
	q := Q().Space("SELECT id, payload->>'name' as name, payload->>'status' as status").
		Space("FROM tasks").
		Where("instance = ?", instanceID)

	if filterDeleted {
		q.And("deleted_at IS NULL")
	}

	if filterStatus != "" {
		q.And("payload->>'status' = ?", filterStatus)
	}

	if hasTypeFilter {
		q.And("payload ?? 'type'").
			And("payload->>'type' = ANY(?)", types)
	}

	// Add subquery for counting related items
	countSubquery := Q().Space("SELECT COUNT(*) FROM task_runs WHERE task_id = tasks.id AND status = ?", "completed")
	q.And("(?) > ?", countSubquery, 0)

	q.Space("ORDER BY created_at DESC").
		Space("LIMIT ?", 50)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Contains(t, sql, "SELECT id, payload->>'name' as name")
	require.Contains(t, sql, "WHERE instance = $1")
	require.Contains(t, sql, "AND deleted_at IS NULL")
	require.Contains(t, sql, "AND payload->>'status' = $2")
	require.Contains(t, sql, "AND payload ? 'type'")
	require.Contains(t, sql, "AND payload->>'type' = ANY($3)")
	require.Contains(t, sql, "AND (SELECT COUNT(*) FROM task_runs WHERE task_id = tasks.id AND status = $4) > $5")
	require.Contains(t, sql, "ORDER BY created_at DESC")
	require.Contains(t, sql, "LIMIT $6")
	require.Equal(t, []any{instanceID, filterStatus, types, "completed", 0, 50}, args)
}

func TestQuery_UpdateWithComplexSet(t *testing.T) {
	// Complex UPDATE with conditional SET fields and JSONB
	id := 123
	name := "Updated Name"
	updateMetadata := true
	metadataKey := "version"
	metadataValue := "2.0"

	set := Q()
	set.Comma("name = ?", name)
	set.Comma("updated_at = NOW()")

	if updateMetadata {
		set.Comma("metadata = jsonb_set(metadata, ?, ?)", "{"+metadataKey+"}", metadataValue)
	}

	where := Q().Space("id = ?", id).And("deleted_at IS NULL")

	q := Q().Space("UPDATE tasks SET ?", set).
		Space("WHERE ?", where).
		Space("RETURNING *")

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "UPDATE tasks SET name = $1, updated_at = NOW(), metadata = jsonb_set(metadata, $2, $3) WHERE id = $4 AND deleted_at IS NULL RETURNING *", sql)
	require.Equal(t, []any{name, "{" + metadataKey + "}", metadataValue, id}, args)
}

func TestQuery_InsertWithReturning(t *testing.T) {
	// INSERT with RETURNING clause
	q := Q().Space("INSERT INTO users (name, email, metadata)").
		Space("VALUES (?, ?, ?::JSONB)", "Alice", "alice@example.com", `{"role": "admin"}`).
		Space("RETURNING id, created_at")

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "INSERT INTO users (name, email, metadata) VALUES ($1, $2, $3::JSONB) RETURNING id, created_at", sql)
	require.Equal(t, []any{"Alice", "alice@example.com", `{"role": "admin"}`}, args)
}

func TestQuery_CTE(t *testing.T) {
	// Common Table Expression (CTE) - must be built in one query to maintain proper structure
	q := Q().Space("WITH recent_orders AS (").
		Space("SELECT user_id, COUNT(*) as order_count").
		Space("FROM orders").
		Space("WHERE created_at > ?", "2024-01-01").
		Space("GROUP BY user_id").
		Space(")").
		Space("SELECT u.*, ro.order_count").
		Space("FROM users u").
		Space("JOIN recent_orders ro ON u.id = ro.user_id").
		Where("ro.order_count > ?", 5)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Contains(t, sql, "WITH recent_orders AS")
	require.Contains(t, sql, "WHERE created_at > $1")
	require.Contains(t, sql, "WHERE ro.order_count > $2")
	require.Equal(t, []any{"2024-01-01", 5}, args)
}

func TestQuery_RegressionTest_BatchUpdateDatabase(t *testing.T) {
	// Multiple database conditions
	databases := []struct {
		InstanceID   string
		DatabaseName string
	}{
		{"inst-1", "db-1"},
		{"inst-2", "db-2"},
	}

	set := Q().Comma("project = ?", "proj-123")

	where := Q()
	for _, db := range databases {
		where.Or("(db.instance = ? AND db.name = ?)", db.InstanceID, db.DatabaseName)
	}

	q := Q().Space("UPDATE db SET ? WHERE ?", set, where)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "UPDATE db SET project = $1 WHERE (db.instance = $2 AND db.name = $3) OR (db.instance = $4 AND db.name = $5)", sql)
	require.Equal(t, []any{"proj-123", "inst-1", "db-1", "inst-2", "db-2"}, args)
}

func TestQuery_ConditionalOrWhereClause(t *testing.T) {
	// Test building WHERE clause with only Or() calls - pattern from actual codebase
	// This tests that the first Or() doesn't add " OR " prefix
	where := Q()

	// First condition
	environmentID := "prod"
	where.Or("environment = ?", environmentID)

	// Multiple database conditions
	databases := []struct {
		InstanceID   string
		DatabaseName string
	}{
		{"inst-1", "db-1"},
		{"inst-2", "db-2"},
	}

	for _, db := range databases {
		where.Or("(db.instance = ? AND db.name = ?)", db.InstanceID, db.DatabaseName)
	}

	// Build final UPDATE query
	set := Q().Comma("project = ?", "proj-123")
	q := Q().Space("UPDATE db SET ?", set).Space("WHERE ?", where)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	require.Equal(t, "UPDATE db SET project = $1 WHERE environment = $2 OR (db.instance = $3 AND db.name = $4) OR (db.instance = $5 AND db.name = $6)", sql)
	require.Equal(t, []any{"proj-123", "prod", "inst-1", "db-1", "inst-2", "db-2"}, args)
}

func TestQuery_ConditionalOrOnlyDatabases(t *testing.T) {
	// Test when only database conditions exist (no environment filter)
	where := Q()

	databases := []struct {
		InstanceID   string
		DatabaseName string
	}{
		{"inst-1", "db-1"},
		{"inst-2", "db-2"},
	}

	for _, db := range databases {
		where.Or("(db.instance = ? AND db.name = ?)", db.InstanceID, db.DatabaseName)
	}

	set := Q().Comma("project = ?", "proj-123")
	q := Q().Space("UPDATE db SET ?", set).Space("WHERE ?", where)

	sql, args, err := q.ToSQL()
	require.NoError(t, err)
	// First Or() should NOT have " OR " prefix
	require.Equal(t, "UPDATE db SET project = $1 WHERE (db.instance = $2 AND db.name = $3) OR (db.instance = $4 AND db.name = $5)", sql)
	require.Equal(t, []any{"proj-123", "inst-1", "db-1", "inst-2", "db-2"}, args)
}
