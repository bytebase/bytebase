package mysql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetDesignSchema(t *testing.T) {
	type designTest struct {
		Description string
		Baseline    string
		Target      string
		Result      string
	}

	const (
		record = true
	)
	var (
		filepaths = []string{
			"testdata/get-design-schema/get_design_schema.yaml",
			"testdata/get-design-schema/partition.yaml",
		}
	)

	a := require.New(t)
	for _, filepath := range filepaths {
		yamlFile, err := os.Open(filepath)
		a.NoError(err)

		tests := []designTest{}
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(yamlFile.Close())
		a.NoError(err)
		a.NoError(yaml.Unmarshal(byteValue, &tests))

		for i, t := range tests {
			targetMeta := &storepb.DatabaseSchemaMetadata{}
			a.NoError(protojson.Unmarshal([]byte(t.Target), targetMeta))
			result, err := GetDesignSchema(t.Baseline, targetMeta)
			a.NoError(err)

			// Addintional parse stage to verify the result is parsable.
			_, err = parser.ParseMySQL(t.Result)
			a.NoError(err)

			if record {
				tests[i].Result = result
			} else {
				a.Equalf(t.Result, result, "test case %d: %s", i, t.Description)
			}
		}

		if record {
			byteValue, err := yaml.Marshal(tests)
			a.NoError(err)
			err = os.WriteFile(filepath, byteValue, 0644)
			a.NoError(err)
		}
	}
}
