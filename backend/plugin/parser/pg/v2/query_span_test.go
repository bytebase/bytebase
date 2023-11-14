package v2

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetQuerySpanResult(t *testing.T) {
	type testCase struct {
		Description       string `yaml:"description,omitempty"`
		Statement         string `yaml:"statement,omitempty"`
		ConnectedDatabase string `yaml:"connectedDatabase,omitempty"`
		// Metadata is the protojson encoded storepb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata  string          `yaml:"metadata,omitempty"`
		QuerySpan *base.QuerySpan `yaml:"querySpan,omitempty"`
	}

	const (
		record       = false
		testDataPath = "testdata/query_span_result.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(testDataPath)
	a.NoError(err)

	var testCases []testCase
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yamlFile.Close())
	a.NoError(yaml.Unmarshal(byteValue, &testCases))

	for i, tc := range testCases {
		metadata := &storepb.DatabaseSchemaMetadata{}
		a.NoError(protojson.Unmarshal([]byte(tc.Metadata), metadata))
		databaseMetadataGetter := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
		result, err := GetQuerySpan(context.TODO(), tc.Statement, tc.ConnectedDatabase, databaseMetadataGetter)
		a.NoError(err)
		if record {
			testCases[i].QuerySpan = result
		} else {
			a.Equal(tc.QuerySpan, result, "statement: %s", tc.Statement)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(testCases)
		a.NoError(err)
		err = os.WriteFile(testDataPath, byteValue, 0644)
		a.NoError(err)
	}
}

func buildMockDatabaseMetadataGetter(databaseMetadata []*storepb.DatabaseSchemaMetadata) base.GetDatabaseMetadataFunc {
	return func(_ context.Context, databaseName string) (*model.DatabaseMetadata, error) {
		m := make(map[string]*model.DatabaseMetadata)
		for _, metadata := range databaseMetadata {
			m[metadata.Name] = model.NewDatabaseMetadata(metadata)
		}

		if databaseMetadata, ok := m[databaseName]; ok {
			return databaseMetadata, nil
		}

		return nil, errors.Errorf("database %q not found", databaseName)
	}
}
