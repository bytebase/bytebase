package store

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// RowStatus is the status for a row.
type RowStatus string

const (
	// Normal is the status for a normal row.
	Normal RowStatus = "NORMAL"
	// Archived is the status for an archived row.
	Archived RowStatus = "ARCHIVED"
)

// SortOrder is the sort order for the returned list.
type SortOrder string

const (
	// ASC is the sort order to return in ascending order.
	ASC SortOrder = "ASC"
	// DESC is the sort order to return in descending order.
	DESC SortOrder = "DESC"
)

func (s SortOrder) String() string {
	return string(s)
}

type OrderByKey struct {
	Key       string
	SortOrder SortOrder
}

// buildStableOrderBy renders the ORDER BY clause for an offset-paginated list.
//
// Offset pagination reads every page with a fresh LIMIT/OFFSET query. When the
// sort key is not unique under the query's scope, PostgreSQL may order tied
// rows differently in each of those queries, so a tied row can cross the page
// boundary between two reads and the caller then skips it or sees it twice.
// Ties are not hypothetical: `created_at` defaults to `now()`, which is the
// transaction timestamp, so rows written by one batch insert all share it.
//
// tieBreak names columns that are unique under the query's scope; together with
// keys they make the ordering total, which is what keeps offset pages stable.
// Pick them from the table's primary key or a declared non-partial unique key
// in LATEST.sql, and include every scope column of a composite key — `id` alone
// does not identify a row in a `(project, id)` table. They are appended in the
// last sort key's direction so a composite index can still serve the ordering.
//
// See backend/store/README.md#pagination-ordering.
func buildStableOrderBy(keys []*OrderByKey, tieBreak ...string) string {
	direction := ASC
	if len(keys) > 0 {
		direction = keys[len(keys)-1].SortOrder
	}
	// A column already ordered on cannot break its own ties, so a tiebreak the
	// caller also sorts by is dropped rather than repeated.
	seen := make(map[string]bool, len(keys)+len(tieBreak))
	parts := make([]string, 0, len(keys)+len(tieBreak))
	appendPart := func(column string, sortOrder SortOrder) {
		if seen[column] {
			return
		}
		seen[column] = true
		parts = append(parts, fmt.Sprintf("%s %s", column, sortOrder))
	}
	for _, key := range keys {
		appendPart(key.Key, key.SortOrder)
	}
	for _, column := range tieBreak {
		appendPart(column, direction)
	}
	return "ORDER BY " + strings.Join(parts, ", ")
}

// getOrderByKeys parses an AIP-132 order_by string against a whitelist
// mapping API field names to SQL columns. Strict where parseOrderBy is not:
// every comma-separated entry must be a whitelisted field with an optional
// "asc"/"desc" suffix, and a repeated field is rejected — malformed input
// errors instead of being silently reinterpreted.
func getOrderByKeys(orderBy string, columns map[string]string) ([]*OrderByKey, error) {
	if orderBy == "" {
		return nil, nil
	}

	var result []*OrderByKey
	seen := make(map[string]bool)
	for entry := range strings.SplitSeq(orderBy, ",") {
		parts := strings.Fields(entry)
		if len(parts) == 0 || len(parts) > 2 {
			return nil, errors.Errorf("invalid order_by entry %q", strings.TrimSpace(entry))
		}
		column, ok := columns[parts[0]]
		if !ok {
			return nil, errors.Errorf("unsupported order field %q", parts[0])
		}
		if seen[parts[0]] {
			return nil, errors.Errorf("duplicate order field %q", parts[0])
		}
		seen[parts[0]] = true
		sortOrder := ASC
		if len(parts) == 2 {
			switch parts[1] {
			case "asc":
			case "desc":
				sortOrder = DESC
			default:
				return nil, errors.Errorf("invalid order direction %q, expect asc or desc", parts[1])
			}
		}
		result = append(result, &OrderByKey{Key: column, SortOrder: sortOrder})
	}
	return result, nil
}

func parseOrderBy(orderBy string) ([]*OrderByKey, error) {
	if orderBy == "" {
		return nil, nil
	}

	var result []*OrderByKey
	re := regexp.MustCompile(`(\w+)\s*(asc|desc)?`)
	matches := re.FindAllStringSubmatch(orderBy, -1)
	for _, match := range matches {
		if len(match) > 3 {
			return nil, errors.Errorf("invalid order by %q", orderBy)
		}
		key := &OrderByKey{
			Key:       match[1],
			SortOrder: ASC,
		}
		if len(match) == 3 && match[2] == "desc" {
			key.SortOrder = DESC
		}
		result = append(result, key)
	}
	return result, nil
}
