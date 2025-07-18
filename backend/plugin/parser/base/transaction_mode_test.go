package base

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
)

// TestParseTransactionMode tests the transaction mode parser functionality.
func TestParseTransactionMode(t *testing.T) {
	tests := []struct {
		name         string
		script       string
		expectedMode common.TransactionMode
		expectedSQL  string
	}{
		{
			name:         "Transaction mode on",
			script:       "-- txn-mode = on\nSELECT 1;",
			expectedMode: common.TransactionModeOn,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "Transaction mode off",
			script:       "-- txn-mode = off\nSELECT 1;",
			expectedMode: common.TransactionModeOff,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "No transaction mode directive",
			script:       "SELECT 1;",
			expectedMode: common.TransactionModeUnspecified,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "Case insensitive ON",
			script:       "-- TXN-MODE = ON\nSELECT 1;",
			expectedMode: common.TransactionModeOn,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "Case insensitive off",
			script:       "-- Txn-Mode = Off\nSELECT 1;",
			expectedMode: common.TransactionModeOff,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "Extra spaces",
			script:       "--  txn-mode  =  on  \nSELECT 1;",
			expectedMode: common.TransactionModeOn,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "Multiple lines after directive",
			script:       "-- txn-mode = off\n\n-- Another comment\nSELECT 1;\nSELECT 2;",
			expectedMode: common.TransactionModeOff,
			expectedSQL:  "-- Another comment\nSELECT 1;\nSELECT 2;",
		},
		{
			name:         "Directive not on first line",
			script:       "-- Regular comment\n-- txn-mode = on\nSELECT 1;",
			expectedMode: common.TransactionModeUnspecified,
			expectedSQL:  "-- Regular comment\n-- txn-mode = on\nSELECT 1;",
		},
		{
			name:         "Invalid directive value",
			script:       "-- txn-mode = invalid\nSELECT 1;",
			expectedMode: common.TransactionModeUnspecified,
			expectedSQL:  "-- txn-mode = invalid\nSELECT 1;",
		},
		{
			name:         "Empty script",
			script:       "",
			expectedMode: common.TransactionModeUnspecified,
			expectedSQL:  "",
		},
		{
			name:         "Only directive",
			script:       "-- txn-mode = on",
			expectedMode: common.TransactionModeOn,
			expectedSQL:  "",
		},
		{
			name:         "Windows line endings",
			script:       "-- txn-mode = on\r\nSELECT 1;",
			expectedMode: common.TransactionModeOn,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "Tab after comment",
			script:       "--\ttxn-mode = on\nSELECT 1;",
			expectedMode: common.TransactionModeOn,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "No space around equals",
			script:       "-- txn-mode=on\nSELECT 1;",
			expectedMode: common.TransactionModeOn,
			expectedSQL:  "SELECT 1;",
		},
		{
			name:         "Multiple spaces in comment",
			script:       "-- -- txn-mode = on\nSELECT 1;",
			expectedMode: common.TransactionModeUnspecified,
			expectedSQL:  "-- -- txn-mode = on\nSELECT 1;",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mode, sql := ParseTransactionMode(test.script)
			require.Equal(t, test.expectedMode, mode, "Transaction mode mismatch")
			require.Equal(t, test.expectedSQL, sql, "SQL mismatch")
		})
	}
}
