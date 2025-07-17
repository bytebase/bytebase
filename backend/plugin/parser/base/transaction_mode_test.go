package base

import (
	"testing"
)

func TestParseTransactionMode(t *testing.T) {
	tests := []struct {
		name           string
		script         string
		expectedMode   TransactionMode
		expectedScript string
	}{
		{
			name:           "transaction mode on",
			script:         "-- txn-mode = on\nCREATE TABLE foo (id INT);",
			expectedMode:   TransactionModeOn,
			expectedScript: "CREATE TABLE foo (id INT);",
		},
		{
			name:           "transaction mode off",
			script:         "-- txn-mode = off\nCREATE INDEX CONCURRENTLY idx ON foo(id);",
			expectedMode:   TransactionModeOff,
			expectedScript: "CREATE INDEX CONCURRENTLY idx ON foo(id);",
		},
		{
			name:           "transaction mode uppercase",
			script:         "-- TXN-MODE = OFF\nVACUUM ANALYZE;",
			expectedMode:   TransactionModeOff,
			expectedScript: "VACUUM ANALYZE;",
		},
		{
			name:           "transaction mode with extra spaces",
			script:         "  --   txn-mode   =   on  \nINSERT INTO foo VALUES (1);",
			expectedMode:   TransactionModeOn,
			expectedScript: "INSERT INTO foo VALUES (1);",
		},
		{
			name:           "no transaction mode directive",
			script:         "CREATE TABLE bar (id INT);",
			expectedMode:   TransactionModeUnspecified,
			expectedScript: "CREATE TABLE bar (id INT);",
		},
		{
			name:           "transaction mode not on first line",
			script:         "CREATE TABLE baz (id INT);\n-- txn-mode = on\nINSERT INTO baz VALUES (1);",
			expectedMode:   TransactionModeUnspecified,
			expectedScript: "CREATE TABLE baz (id INT);\n-- txn-mode = on\nINSERT INTO baz VALUES (1);",
		},
		{
			name:           "invalid transaction mode value",
			script:         "-- txn-mode = maybe\nCREATE TABLE qux (id INT);",
			expectedMode:   TransactionModeUnspecified,
			expectedScript: "-- txn-mode = maybe\nCREATE TABLE qux (id INT);",
		},
		{
			name:           "empty script",
			script:         "",
			expectedMode:   TransactionModeUnspecified,
			expectedScript: "",
		},
		{
			name:           "only directive",
			script:         "-- txn-mode = on",
			expectedMode:   TransactionModeOn,
			expectedScript: "",
		},
		{
			name:           "multiline script with directive",
			script:         "-- txn-mode = off\nCREATE TABLE t1 (id INT);\n\nCREATE TABLE t2 (id INT);\n-- This is a comment\nINSERT INTO t1 VALUES (1);",
			expectedMode:   TransactionModeOff,
			expectedScript: "CREATE TABLE t1 (id INT);\n\nCREATE TABLE t2 (id INT);\n-- This is a comment\nINSERT INTO t1 VALUES (1);",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, script := ParseTransactionMode(tt.script)
			if mode != tt.expectedMode {
				t.Errorf("expected mode %q, got %q", tt.expectedMode, mode)
			}
			if script != tt.expectedScript {
				t.Errorf("expected script %q, got %q", tt.expectedScript, script)
			}
		})
	}
}
