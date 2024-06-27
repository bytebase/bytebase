package v1

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestTryMerge(t *testing.T) {
	type testCase struct {
		Description string `yaml:"description"`
		// Merge Head to Base with Ancestor as the common ancestor.
		// Base is the base storepb.DatabaseSchemaMetadata protojson marshalled string.
		Ancestor string `yaml:"ancestor"`
		// Head is the head storepb.DatabaseSchemaMetadata protojson marshalled string.
		Head string `yaml:"head"`
		// Ancestor is the ancestor storepb.DatabaseSchemaMetadata protojson marshalled string.
		Base string `yaml:"base"`
		// Expected is the expected storepb.DatabaseSchemaMetadata protojson marshalled string.
		Expected string `yaml:"expected"`
	}

	const record = false

	a := require.New(t)

	testFilepath := "testdata/schema_design_merge/try_merge.yaml"
	content, err := os.ReadFile(testFilepath)
	a.NoError(err)
	var testCases []testCase
	err = yaml.Unmarshal(content, &testCases)
	a.NoError(err)

	for idx, tc := range testCases {
		ancestorSchemaMetadata := new(storepb.DatabaseSchemaMetadata)
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Ancestor), ancestorSchemaMetadata)
		a.NoErrorf(err, "test case %d: %s", idx, tc.Description)
		headSchemaMetadata := new(storepb.DatabaseSchemaMetadata)
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Head), headSchemaMetadata)
		a.NoErrorf(err, "test case %d: %s", idx, tc.Description)
		baseSchemaMetadata := new(storepb.DatabaseSchemaMetadata)
		err := common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Base), baseSchemaMetadata)
		a.NoErrorf(err, "test case %d: %s", idx, tc.Description)
		mergedSchemaMetadata, err := tryMerge(ancestorSchemaMetadata, headSchemaMetadata, baseSchemaMetadata, storepb.Engine_MYSQL)
		a.NoErrorf(err, "test case %d: %s", idx, tc.Description)
		a.NotNil(mergedSchemaMetadata, "test case %d: %s, mergedSchemaMetadata should not be nil if there is no error", idx, tc.Description)

		s := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Format(mergedSchemaMetadata)
		a.NoErrorf(err, "test case %d: %s", idx, tc.Description)
		if record {
			testCases[idx].Expected = strings.TrimSpace(s)
		} else {
			expectedSchemaMetadata := new(storepb.DatabaseSchemaMetadata)
			err = common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Expected), expectedSchemaMetadata)
			a.NoErrorf(err, "test case %d: %s", idx, tc.Description)
			if diff := cmp.Diff(expectedSchemaMetadata, mergedSchemaMetadata, protocmp.Transform()); diff != "" {
				a.Failf("Failed", "mismatch (-want +got):\n%s", diff)
			}
		}
	}

	if record {
		f, err := os.OpenFile(testFilepath, os.O_WRONLY|os.O_TRUNC, 0644)
		a.NoError(err)
		defer f.Close()
		content, err := yaml.Marshal(testCases)
		a.NoError(err)
		_, err = f.Write(content)
		a.NoError(err)
	}
}
