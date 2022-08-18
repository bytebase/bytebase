package parser

import "fmt"

// SingleSQL is a separate SQL split from multi-SQL.
type SingleSQL struct {
	Text string
	Line int
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]SingleSQL, error) {
	switch engineType {
	case Postgres:
		t := newTokenizer(statement)
		return t.splitPostgreSQLMultiSQL()
	default:
		return nil, fmt.Errorf("engine type is not supported: %s", engineType)
	}
}
