package batch

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBuildGoCommands(t *testing.T) {
	testCases := []struct {
		input     string
		wantNil   bool
		wantCount uint
	}{
		{
			input:   "",
			wantNil: true,
		},
		{
			input:     "GO 123",
			wantNil:   false,
			wantCount: 123,
		},
		{
			input:   "GO [123]",
			wantNil: true,
		},
		{
			input:   "GO -1",
			wantNil: true,
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		got := buildGoCommand(tc.input)
		if tc.wantNil {
			a.Nil(got)
		} else {
			goCmd, ok := got.(*GoCommand)
			a.True(ok)
			a.Equalf(tc.wantCount, goCmd.Count, "Input: %s", tc.input)
		}
	}
}

type batchResult struct {
	Statements Batch  `yaml:"statements"`
	Command    string `yaml:"command"`
}

func TestBatch(t *testing.T) {
	type TestCase struct {
		// input is the input string.
		// The input string can be a single SQL statement or a batch of SQL statements.
		Input       string        `yaml:"input"`
		Description string        `yaml:"description"`
		Batches     []batchResult `yaml:"batches"`
	}

	a := require.New(t)

	filePath := "testdata/test_batch.yaml"
	f, err := os.Open(filePath)
	a.NoError(err)
	defer f.Close()

	bytes, err := io.ReadAll(f)
	a.NoError(err)

	var testCases []TestCase
	a.NoError(yaml.Unmarshal(bytes, &testCases))

	const (
		record = false
	)
	for i, tc := range testCases {
		batchResults := getBatchResults(a, tc.Input)
		if record {
			testCases[i].Batches = batchResults
		} else {
			a.Equalf(tc.Batches, batchResults, "Input: %s", tc.Input)
		}
	}
	if record {
		bytes, err := yaml.Marshal(testCases)
		a.NoError(err)
		a.NoError(os.WriteFile(filePath, bytes, 0644))
	}
}

func getBatchResults(a *require.Assertions, input string) []batchResult {
	batch := NewBatcher(input)
	var batchResults []batchResult
	for {
		command, err := batch.Next()
		if err != nil {
			if err == io.EOF {
				if v := batch.Batch(); v != nil && len(v.Text) > 0 {
					// If meet the end of file, get the last batch.
					batchResults = append(batchResults, batchResult{
						Statements: *v,
					})
				}
				batch.Reset(nil)
				return batchResults
			}
			a.NoError(err)
		}
		if command != nil {
			batchResults = append(batchResults, batchResult{
				Statements: *batch.Batch(),
				Command:    command.String(),
			})
			batch.Reset(nil)
		}
	}
}
