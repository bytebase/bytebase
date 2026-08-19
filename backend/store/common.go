package store

import (
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
