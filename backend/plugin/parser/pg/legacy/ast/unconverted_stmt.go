package ast

// UnconvertedStmt is the struct for unconverted statement definition.
// TODO(rebelice): remove it.
// We define this because we cannot convert all statement types now.
type UnconvertedStmt struct {
	node
}
