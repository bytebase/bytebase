package mysql

import (
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type transformTest struct {
	Schema   string
	Metadata string
}

func TestParseToMetadata_Partition(t *testing.T) {
	filepaths := []string{
		"testdata/parse-to-metadata/partition.yaml",
	}

	runParseToMetadataTest(t, filepaths, false)
}

func TestParseToMetadata_Standard(t *testing.T) {
	filepaths := []string{
		"testdata/parse-to-metadata/standard.yaml",
	}

	runParseToMetadataTest(t, filepaths, false)
}

func runParseToMetadataTest(t *testing.T, filepaths []string, record bool) {
	a := require.New(t)
	for _, fp := range filepaths {
		yamlFile, err := os.Open(fp)
		a.NoError(err)

		tests := []transformTest{}
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(yamlFile.Close())
		a.NoError(err)
		a.NoError(yaml.Unmarshal(byteValue, &tests))

		for i, t := range tests {
			result, err := ParseToMetadata("", t.Schema)
			a.NoError(err)
			if record {
				tests[i].Metadata = protojson.MarshalOptions{Multiline: true, Indent: "  "}.Format(result)
			} else {
				want := &storepb.DatabaseSchemaMetadata{}
				err = protojson.Unmarshal([]byte(t.Metadata), want)
				a.NoError(err)
				diff := cmp.Diff(want, result, protocmp.Transform())
				a.Equal("", diff)
			}
		}

		if record {
			byteValue, err := yaml.Marshal(tests)
			a.NoError(err)
			err = os.WriteFile(fp, byteValue, 0644)
			a.NoError(err)
		}
	}
}
