package elasticsearch

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpan(t *testing.T) {
	type testCase struct {
		Description string         `yaml:"description,omitempty"`
		Statement   string         `yaml:"statement,omitempty"`
		QueryType   base.QueryType `yaml:"queryType,omitempty"`
	}

	const (
		record       = false
		testDataPath = "test-data/query_span.yaml"
	)

	a := require.New(t)

	yamlFile, err := os.Open(testDataPath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yamlFile.Close())

	var testCases []testCase
	a.NoError(yaml.Unmarshal(byteValue, &testCases))

	for i, tc := range testCases {
		t.Run(tc.Description, func(_ *testing.T) {
			span, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: tc.Statement}, "", "", false)
			a.NoError(err)
			a.NotNil(span)
			if record {
				testCases[i].QueryType = span.Type
			} else {
				a.Equalf(tc.QueryType, span.Type, "description: %s, statement: %s", tc.Description, tc.Statement)
			}
		})
	}

	if record {
		byteValue, err := yaml.Marshal(testCases)
		a.NoError(err)
		err = os.WriteFile(testDataPath, byteValue, 0644)
		a.NoError(err)
	}
}

func TestGetQuerySpan_Error(t *testing.T) {
	// MongoDB style query, definitely not ElasticSearch
	stmt := "db.users.find({})"
	span, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: stmt}, "", "", false)

	// New behavior: returns error
	require.Error(t, err)
	require.Nil(t, span)
	require.Contains(t, err.Error(), "Syntax error")
}
