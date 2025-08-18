package ast

// PartitionKeyType is the type of partition key.
type PartitionKeyType int

const (
	// PartitionKeyTypeColumn is the type of partition key column.
	PartitionKeyTypeColumn PartitionKeyType = iota
	// PartitionKeyTypeExpression is the type of partition key expression.
	PartitionKeyTypeExpression
)

// PartitionKeyDef is the struct for partition key definition.
type PartitionKeyDef struct {
	node

	Type PartitionKeyType
	Key  string
}
