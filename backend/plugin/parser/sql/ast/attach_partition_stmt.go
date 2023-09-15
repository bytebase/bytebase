package ast

// AttachPartitionStmt is the attach partition statement.
type AttachPartitionStmt struct {
	node

	Table     *TableDef
	Partition *TableDef
	// TODO(rebelice): convert more attach partition statement fields.
}
