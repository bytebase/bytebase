// Package snowflake is the advisor for snowflake database.
package snowflake

// currentConstraintAction is the action of current constraint.
type currentConstraintAction int

const (
	currentConstraintActionNone currentConstraintAction = iota
	currentConstraintActionAdd
	currentConstraintActionDrop
)
