package api

import (
	"fmt"
	"strings"
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

// StringToSortOrder converts a valid string to SortOrder.
func StringToSortOrder(s string) (SortOrder, error) {
	switch strings.ToUpper(s) {
	case string(ASC):
		return ASC, nil
	case string(DESC):
		return DESC, nil
	}
	return SortOrder(""), fmt.Errorf("%q cannot be converted to SortOrder", s)
}
