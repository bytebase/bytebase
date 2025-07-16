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
			script:         "-- txn_mode = on\nCREATE TABLE foo (id INT);",
			expectedMode:   TransactionModeOn,
			expectedScript: "CREATE TABLE foo (id INT);",
		},
		{
			name:           "transaction mode off",
			script:         "-- txn_mode = off\nCREATE INDEX CONCURRENTLY idx ON foo(id);",
			expectedMode:   TransactionModeOff,
			expectedScript: "CREATE INDEX CONCURRENTLY idx ON foo(id);",
		},
		{
			name:           "transaction mode uppercase",
			script:         "-- TXN_MODE = OFF\nVACUUM ANALYZE;",
			expectedMode:   TransactionModeOff,
			expectedScript: "VACUUM ANALYZE;",
		},
		{
			name:           "transaction mode with extra spaces",
			script:         "  --   txn_mode   =   on  \nINSERT INTO foo VALUES (1);",
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
			script:         "CREATE TABLE baz (id INT);\n-- txn_mode = on\nINSERT INTO baz VALUES (1);",
			expectedMode:   TransactionModeUnspecified,
			expectedScript: "CREATE TABLE baz (id INT);\n-- txn_mode = on\nINSERT INTO baz VALUES (1);",
		},
		{
			name:           "invalid transaction mode value",
			script:         "-- txn_mode = maybe\nCREATE TABLE qux (id INT);",
			expectedMode:   TransactionModeUnspecified,
			expectedScript: "-- txn_mode = maybe\nCREATE TABLE qux (id INT);",
		},
		{
			name:           "empty script",
			script:         "",
			expectedMode:   TransactionModeUnspecified,
			expectedScript: "",
		},
		{
			name:           "only directive",
			script:         "-- txn_mode = on",
			expectedMode:   TransactionModeOn,
			expectedScript: "",
		},
		{
			name:           "multiline script with directive",
			script:         "-- txn_mode = off\nCREATE TABLE t1 (id INT);\n\nCREATE TABLE t2 (id INT);\n-- This is a comment\nINSERT INTO t1 VALUES (1);",
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

func TestTransactionModeIsValid(t *testing.T) {
	tests := []struct {
		mode    TransactionMode
		isValid bool
	}{
		{TransactionModeOn, true},
		{TransactionModeOff, true},
		{TransactionModeUnspecified, true},
		{TransactionMode("invalid"), false},
		{TransactionMode("maybe"), false},
		{TransactionMode(""), true}, // Empty is same as unspecified
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := tt.mode.IsValid(); got != tt.isValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.isValid)
			}
		})
	}
}

func TestPrependTransactionMode(t *testing.T) {
	tests := []struct {
		name           string
		script         string
		mode           TransactionMode
		expectedScript string
	}{
		{
			name:           "prepend on mode",
			script:         "CREATE TABLE foo (id INT);",
			mode:           TransactionModeOn,
			expectedScript: "-- txn_mode = on\nCREATE TABLE foo (id INT);",
		},
		{
			name:           "prepend off mode",
			script:         "CREATE INDEX CONCURRENTLY idx ON foo(id);",
			mode:           TransactionModeOff,
			expectedScript: "-- txn_mode = off\nCREATE INDEX CONCURRENTLY idx ON foo(id);",
		},
		{
			name:           "unspecified mode returns unchanged",
			script:         "CREATE TABLE bar (id INT);",
			mode:           TransactionModeUnspecified,
			expectedScript: "CREATE TABLE bar (id INT);",
		},
		{
			name:           "invalid mode returns unchanged",
			script:         "CREATE TABLE baz (id INT);",
			mode:           TransactionMode("invalid"),
			expectedScript: "CREATE TABLE baz (id INT);",
		},
		{
			name:           "empty script with mode",
			script:         "",
			mode:           TransactionModeOn,
			expectedScript: "-- txn_mode = on",
		},
		{
			name:           "empty script without mode",
			script:         "",
			mode:           TransactionModeUnspecified,
			expectedScript: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PrependTransactionMode(tt.script, tt.mode)
			if got != tt.expectedScript {
				t.Errorf("expected %q, got %q", tt.expectedScript, got)
			}
		})
	}
}