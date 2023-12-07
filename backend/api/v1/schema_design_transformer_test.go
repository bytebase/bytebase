package v1

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type deparseTest struct {
	Engine   storepb.Engine
	Metadata string
	Schema   string
}

func TestDeparseSchemaString(t *testing.T) {
	const (
		record = false
	)
	var (
		filepath = "testdata/deparse.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	tests := []deparseTest{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		metadata := &storepb.DatabaseSchemaMetadata{}
		a.NoError(protojson.Unmarshal([]byte(t.Metadata), metadata))
		result, err := transformDatabaseMetadataToSchemaString(t.Engine, metadata)
		a.NoError(err)
		if record {
			tests[i].Schema = strings.TrimSpace(result)
		} else {
			a.Equal(strings.TrimSpace(t.Schema), strings.TrimSpace(result))
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

type transformTest struct {
	Engine   storepb.Engine
	Schema   string
	Metadata string
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

	tests := []transformTest{}
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := TransformSchemaStringToDatabaseMetadata(t.Engine, t.Schema)
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
	Engine   storepb.Engine
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
