package pg

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	// Import PostgreSQL parser.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type parseToMetadataTest struct {
	Schema   string
	Metadata *v1pb.DatabaseMetadata
}

func TestParseToMetadata(t *testing.T) {
	const (
		record = true
	)
	var (
		filepath = "testdata/parse_to_metadata.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []parseToMetadataTest{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := ParseToMetadata(t.Schema)
		a.NoError(err)
		if record {
			tests[i].Metadata = result
		} else {
			a.Equal(t.Metadata, result)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

type getSchemaDesignTest struct {
	Baseline string
	Target   *v1pb.DatabaseMetadata
	Result   string
}

func TestGetSchemaDesign(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/get_design_schema.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []getSchemaDesignTest{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := GetDesignSchema(t.Baseline, t.Target)
		a.NoError(err)
		if record {
			tests[i].Result = result
		} else {
			a.Equal(t.Result, result)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}
