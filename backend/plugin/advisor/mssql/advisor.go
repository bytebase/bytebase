// Package mssql is the advisor for MSSQL database.
package mssql

// currentConstraintAction is the action of current constraint.
type currentConstraintAction int

const (
	currentConstraintActionNone currentConstraintAction = iota
	currentConstraintActionAdd
	currentConstraintActionDrop
)
