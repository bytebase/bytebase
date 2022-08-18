package parser

import "fmt"

// SeparateSQL is a separate SQL split from multi-SQL.
type SeparateSQL struct {
	Text string
	Line int
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]SeparateSQL, error) {
	switch engineType {
	case Postgres:
		t := newTokenizer(statement)
		return t.splitPostgreSQLMultiSQL()
	default:
		return nil, fmt.Errorf("engine type is not supported: %s", engineType)
	}
}
