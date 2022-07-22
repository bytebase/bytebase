package parser

import "fmt"

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]string, error) {
	switch engineType {
	case Postgres:
		t := newTokenizer(statement)
		return t.splitPostgreSQLMultiSQL()
	default:
		return nil, fmt.Errorf("engine type is not supported: %s", engineType)
	}
}
