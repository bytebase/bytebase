package ast

// SortOrderType is the sort order type for the index.
type SortOrderType int

const (
	// SortOrderTypeDefault is the sort order type for default value.
	SortOrderTypeDefault = iota
	// SortOrderTypeAscending is the sort order type for ASC.
	SortOrderTypeAscending
	// SortOrderTypeDescending is the sort order type for DESC.
	SortOrderTypeDescending
)

// NullOrderType is the null sort order type for the index.
type NullOrderType int

const (
	// NullOrderTypeDefault is the default null sort order type.
	NullOrderTypeDefault = iota
	// NullOrderTypeFirst is the null sort order type that nulls sort before non-nulls.
	// This is the default when DESC is specified.
	NullOrderTypeFirst
	// NullOrderTypeLast is the null sort order type that nulls sort after non-nulls.
	// This is the default when DESC is not specified.
	NullOrderTypeLast
)

// IndexKeyType is the type for index key.
type IndexKeyType int

const (
	// IndexKeyTypeColumn is the type for the column.
	IndexKeyTypeColumn IndexKeyType = iota
	// IndexKeyTypeExpression is the type for the expression.
	IndexKeyTypeExpression
)

// IndexKeyDef is the struct for index key definition.
type IndexKeyDef struct {
	node

	Type      IndexKeyType
	Key       string
	SortOrder SortOrderType
	NullOrder NullOrderType
}
