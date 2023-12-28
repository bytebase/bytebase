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

func TestParseToMetadata(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/schema.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []transformTest{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := ParseToMetadata(t.Schema)
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
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

type designTest struct {
	Baseline string
	Target   string
	Result   string
}

func TestGetDesignSchema(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/design.yaml"
	)

	a := require.New(t)
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
