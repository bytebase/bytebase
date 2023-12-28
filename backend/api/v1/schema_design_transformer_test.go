package v1

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type transformTest struct {
	Engine   storepb.Engine
	Schema   string
	Metadata string
}

type designTest struct {
	Engine   storepb.Engine
	Baseline string
	Target   string
	Result   string
}

type checkTest struct {
	Engine   storepb.Engine
	Metadata string
	Err      string
}

func TestCheckDatabaseMetadata(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/check.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []checkTest{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		meta := &storepb.DatabaseSchemaMetadata{}
		a.NoError(protojson.Unmarshal([]byte(t.Metadata), meta))
		err := checkDatabaseMetadata(t.Engine, meta)
		if record {
			if err != nil {
				tests[i].Err = err.Error()
			} else {
				tests[i].Err = ""
			}
		} else {
			if t.Err == "" {
				a.NoError(err)
			} else {
				a.NotNil(err, t.Err)
				a.Equal(t.Err, err.Error())
			}
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}
