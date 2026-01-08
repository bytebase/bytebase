package standard

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStandardSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
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

	total := 0
	countStatements := func(string) error {
		total++
		return nil
	}

	for _, test := range tests {
		total = 0
		err := applyMultiStatements(strings.NewReader(test.statement), countStatements)
		require.NoError(t, err)
		require.Equal(t, test.total, total)
	}
}
