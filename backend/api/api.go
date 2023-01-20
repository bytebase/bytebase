// Package api provides the definition of all APIs.
package api

import (
	"strings"

	"github.com/pkg/errors"
)

// UnknownID is the ID for unknowns.
const UnknownID = -1

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

const (
	// DefaultPageSize is the default number of items in a page.
	DefaultPageSize = 1000
)

// StringToSortOrder converts a valid string to SortOrder.
func StringToSortOrder(s string) (SortOrder, error) {
	switch strings.ToUpper(s) {
	case string(ASC):
		return ASC, nil
	case string(DESC):
		return DESC, nil
	}
	return SortOrder(""), errors.Errorf("%q cannot be converted to SortOrder", s)
}
