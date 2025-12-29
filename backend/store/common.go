package store

import (
	"regexp"

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
