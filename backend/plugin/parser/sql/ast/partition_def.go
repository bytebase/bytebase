package ast

// PartitionDef is the struct for partition specification.
type PartitionDef struct {
	node

	Strategy string
	KeyList  []*PartitionKeyDef
}
