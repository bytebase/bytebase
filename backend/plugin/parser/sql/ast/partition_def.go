package ast

import (
	pgquery "github.com/pganalyze/pg_query_go/v5"
)

// PartitionDef is the struct for partition specification.
type PartitionDef struct {
	node

	Strategy pgquery.PartitionStrategy
	KeyList  []*PartitionKeyDef
}
