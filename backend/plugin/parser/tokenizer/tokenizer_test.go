package tokenizer

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSplitTiDBMultiSQL(t *testing.T) {
	type testCase struct {
		Description string
		Statement   string
		Expected    []base.SingleSQL
	}

	const (
		fp     = "testdata/test_split_tidb_multi_sql.yaml"
		record = false
	)

	a := require.New(t)

	f, err := os.Open(fp)
	a.NoError(err)
	content, err := io.ReadAll(f)
	a.NoError(err)
	err = f.Close()
	a.NoError(err)

	var testCases []testCase
	err = yaml.Unmarshal(content, &testCases)
	a.NoError(err)

	for idx, tc := range testCases {
		tokenizer := NewTokenizer(tc.Statement)
		list, err := tokenizer.SplitTiDBMultiSQL()
		a.NoError(err)

		if record {
			testCases[idx].Expected = list
		} else {
			a.Equal(tc.Expected, list, tc.Description)
		}
	}

	if record {
		content, err := yaml.Marshal(testCases)
		a.NoError(err)
		f, err := os.OpenFile(fp, os.O_WRONLY|os.O_TRUNC, 0644)
		a.NoError(err)
		_, err = f.Write(content)
		a.NoError(err)
	}
}
