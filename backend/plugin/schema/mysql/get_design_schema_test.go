package mysql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetDesignSchema_Debug(t *testing.T) {
	if true {
		t.Skip()
	}

	TestGetDesignSchema_Basic(t)
	TestGetDesignSchema_Partition(t)
	TestGetDesignSchema_View(t)
	TestGetDesignSchema_Function(t)
	TestGetDesignSchema_Procedure(t)
}

func TestGetDesignSchema_Basic(t *testing.T) {
	testGetDesignSchema(t, []string{
		"testdata/get-design-schema/get_design_schema.yaml",
	}, false)
}

func TestGetDesignSchema_Partition(t *testing.T) {
	testGetDesignSchema(t, []string{
		"testdata/get-design-schema/partition/range.yaml",
		"testdata/get-design-schema/partition/list.yaml",
		"testdata/get-design-schema/partition/hash.yaml",
		"testdata/get-design-schema/partition/key.yaml",
		"testdata/get-design-schema/partition/write-default.yaml",
		"testdata/get-design-schema/partition/keep-algorithm.yaml",
		"testdata/get-design-schema/partition/keep-version-comment.yaml",
	}, false)
}

func TestGetDesignSchema_View(t *testing.T) {
	testGetDesignSchema(t, []string{
		"testdata/get-design-schema/view/add-new-view.yaml",
		"testdata/get-design-schema/view/drop-view.yaml",
		"testdata/get-design-schema/view/modify-view.yaml",
	}, false)
}

func TestGetDesignSchema_Function(t *testing.T) {
	testGetDesignSchema(t, []string{
		"testdata/get-design-schema/function/create-new-function.yaml",
		"testdata/get-design-schema/function/delete-function.yaml",
		"testdata/get-design-schema/function/delete-function-without-set.yaml",
		"testdata/get-design-schema/function/modify-function.yaml",
	}, false)
}

func TestGetDesignSchema_Procedure(t *testing.T) {
	testGetDesignSchema(t, []string{
		"testdata/get-design-schema/procedure/create-new-procedure.yaml",
		"testdata/get-design-schema/procedure/delete-procedure.yaml",
		"testdata/get-design-schema/procedure/delete-procedure-without-set.yaml",
		"testdata/get-design-schema/procedure/modify-procedure.yaml",
	}, false)
}

type designTest struct {
	Description string
	Target      string
	Result      string
}

func testGetDesignSchema(t *testing.T, filepaths []string, writeback bool) {
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
			a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(t.Target), targetMeta))
			result, err := GetDesignSchema("", targetMeta)
			a.NoError(err)

			// Addintional parse stage to verify the result is parsable.
			_, err = parser.ParseMySQL(result)
			a.NoErrorf(err, "[%s] test case %d: %s\n content:\n%s", filepath, i, t.Description, result)

			if writeback {
				tests[i].Result = result
			} else {
				a.Equalf(t.Result, result, "[%s] test case %d: %s", filepath, i, t.Description)
			}
		}

		if writeback {
			byteValue, err := yaml.Marshal(tests)
			a.NoError(err)
			err = os.WriteFile(filepath, byteValue, 0644)
			a.NoError(err)
		}
	}
}

func TestNormalizeOnUpdate(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{
			s:    "current_timestamp(6)",
			want: "CURRENT_TIMESTAMP(6)",
		},
		{
			s:    "current_timestamp",
			want: "CURRENT_TIMESTAMP",
		},
		{
			s:    "hello",
			want: "hello",
		},
	}
	for _, tc := range tests {
		got := normalizeOnUpdate(tc.s)
		require.Equal(t, tc.want, got, tc.s)
	}
}
