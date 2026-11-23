package standard

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStandardSplitSQL(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   base.SplitMultiSQLFunc
	}{
		{name: "clickhouse", fn: SplitSQL},
		{name: "hive", fn: hiveSplitSQL},
		{name: "databricks", fn: databricksSplitSQL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
				SplitFunc: tc.fn,
			})
		})
	}
}

func generateOneMBInsert() string {
	letterList := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, 1024*1024)
	for i := range b {
		// Use crypto/rand for secure random generation
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterList))))
		if err != nil {
			// Fallback to a simple pattern if crypto/rand fails
			b[i] = letterList[i%26]
		} else {
			b[i] = letterList[randomIndex.Int64()]
		}
	}
	return fmt.Sprintf("INSERT INTO t values('%s')", string(b))
}

func TestApplyMultiStatements(t *testing.T) {
	type testData struct {
		statement string
		total     int
	}
	tests := []testData{
		{
			statement: `
			CREATE TABLE t(
				a int,
				b int,
				c int);


			/* This is a comment */
			CREATE TABLE t1(
				a int, b int c)`,
			total: 2,
		},
		{
			statement: `
			CREATE TABLE t(
				a int,
				b int,
				c int);


			CREATE TABLE t1(
				a int, b int c);
			` + generateOneMBInsert(),
			total: 3,
		},
	}

	for _, test := range tests {
		stmts, err := SplitSQL(test.statement)
		require.NoError(t, err)
		require.Equal(t, test.total, len(stmts))
	}
}
