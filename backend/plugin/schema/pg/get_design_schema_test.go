package pg

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	// Import PostgreSQL parser.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type getSchemaDesignTest struct {
	Baseline string
	Target   string
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
		targetSchema := &storepb.DatabaseSchemaMetadata{}
		a.NoError(protojson.Unmarshal([]byte(t.Target), targetSchema))
		result, err := GetDesignSchema(t.Baseline, targetSchema)
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
