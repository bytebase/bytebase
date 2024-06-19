package tidb

import (
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type parseToMetadataTest struct {
	Schema   string
	Metadata string
}

func TestParseToMetadata(t *testing.T) {
	const (
		record = false
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
		result, err := ParseToMetadata("", t.Schema)
		a.NoError(err)
		if record {
			tests[i].Metadata = protojson.MarshalOptions{Multiline: true, Indent: "  "}.Format(result)
		} else {
			want := &storepb.DatabaseSchemaMetadata{}
			err = common.ProtojsonUnmarshaler.Unmarshal([]byte(t.Metadata), want)
			a.NoError(err)
			diff := cmp.Diff(want, result, protocmp.Transform())
			a.Equal("", diff)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}
