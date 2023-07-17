package v1

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type test struct {
	Engine   v1pb.Engine
	Schema   string
	Metadata *v1pb.DatabaseMetadata
}

func TestTransformSchemaString(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/schema.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []test{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := transformSchemaStringToDatabaseMetadata(t.Engine, t.Schema)
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
