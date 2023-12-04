package v1

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

func TestTiDBTransformSchemaString(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/tidb/schema.yaml"
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
		result, err := transformSchemaStringToDatabaseMetadata(t.Engine, t.Schema)
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

func TestTiDBGetDesignSchema(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/tidb/design.yaml"
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
		result, err := getDesignSchema(t.Engine, t.Baseline, targetMeta)
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

func TestTiDBCheckDatabaseMetadata(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/tidb/check.yaml"
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
