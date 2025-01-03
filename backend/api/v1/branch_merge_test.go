package v1

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	testFilepath := "testdata/branch_merge/try_merge.yaml"
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
		mergedSchemaMetadata, _, err := tryMerge(ancestorSchemaMetadata, headSchemaMetadata, baseSchemaMetadata, nil, nil, nil /* TODO(zp): fix test */, storepb.Engine_MYSQL)
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
			diff := cmp.Diff(expectedSchemaMetadata, mergedSchemaMetadata, protocmp.Transform())
			a.Empty(diff)
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

func TestNormalizeMySQLViewDefinition(t *testing.T) {
	for i, test := range []struct {
		query string
		want  string
	}{
		{
			query: "select `p`.`id` AS `id`,extract(year_month from `p`.`yyy`) AS `extract(year_month from ``yyy``)` from `p` order by `p`.`id` limit 1000000000000",
			want:  "select `id`,extract(year_month from `yyy`) from `p` order by `id` limit 1000000000000;",
		},
		{
			query: "select 12 AS `12`",
			want:  "select 12;",
		},
		{
			query: "select 12 AS `12`;   ",
			want:  "select 12;",
		},
		{
			query: "select `t1`.`id` AS `id` from `t1`;",
			want:  "select `id` from `t1`;",
		},
	} {
		got := normalizeMySQLViewDefinition(test.query)
		require.Equal(t, test.want, got, i)
	}
}

func TestAutoRandomEqual(t *testing.T) {
	testCases := []struct {
		aExpr string
		bExpr string
		equal bool
	}{
		{
			aExpr: "AUTO_RANDOM",
			bExpr: "auto_random(5, 64)",
			equal: true,
		},
		{
			aExpr: "AUTO_RANDOM(5)",
			bExpr: "auto_random(5, 64)",
			equal: true,
		},
		{
			aExpr: "AUTO_RANDOM(10, 32)",
			bExpr: "auto_random",
			equal: false,
		},
	}

	for _, tc := range testCases {
		a := buildAutoRandomDefaultValue(tc.aExpr)
		require.NotNil(t, a)
		b := buildAutoRandomDefaultValue(tc.bExpr)
		require.NotNil(t, b)
		got := isAutoRandomEquivalent(a, b)
		require.Equal(t, tc.equal, got)
	}
}

func TestDeriveUpdateInfoFromMetadataDiff(t *testing.T) {
	time1 := timestamppb.Now()
	time2 := timestamppb.New(time1.AsTime().Add(1 * time.Second))
	testCases := []struct {
		metadataDiff *metadataDiffRootNode
		config       *storepb.DatabaseConfig
		want         *updateInfoDiffRootNode
	}{
		{
			metadataDiff: &metadataDiffRootNode{
				schemas: map[string]*metadataDiffSchemaNode{
					"": {
						tables: map[string]*metadataDiffTableNode{
							"t1": {
								diffBaseNode: diffBaseNode{
									action: diffActionCreate,
								},
							},
							"t2": {
								diffBaseNode: diffBaseNode{
									action: diffActionUpdate,
								},
							},
							"t3": {
								diffBaseNode: diffBaseNode{
									action: diffActionDrop,
								},
							},
						},
					},
				},
			},
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "",
						Tables: []*storepb.TableCatalog{
							{
								Name:         "t1",
								Updater:      "anonymous+01@bytebase.com",
								UpdateTime:   time1,
								SourceBranch: "feat/01",
							},
							{
								Name:         "t2",
								Updater:      "anonymous+02@bytebase.com",
								UpdateTime:   time2,
								SourceBranch: "feat/02",
							},
						},
					},
				},
			},
			want: &updateInfoDiffRootNode{
				schemas: map[string]*updateInfoDiffSchemaNode{
					"": {
						tables: map[string]*updateInfoDiffTableNode{
							"t1": {
								name: "t1",
								diffBaseNode: diffBaseNode{
									action: diffActionCreate,
								},
								updateInfo: &updateInfo{
									lastUpdatedTime: time1,
									lastUpdater:     "anonymous+01@bytebase.com",
									sourceBranch:    "feat/01",
								},
							},
							"t2": {
								name: "t2",
								diffBaseNode: diffBaseNode{
									action: diffActionUpdate,
								},
								updateInfo: &updateInfo{
									lastUpdatedTime: time2,
									lastUpdater:     "anonymous+02@bytebase.com",
									sourceBranch:    "feat/02",
								},
							},
							"t3": {
								name: "t3",
								diffBaseNode: diffBaseNode{
									action: diffActionDrop,
								},
								updateInfo: nil,
							},
						},
						views:      map[string]*updateInfoDiffViewNode{},
						functions:  map[string]*updateInfoDiffFunctionNode{},
						procedures: map[string]*updateInfoDiffProcedureNode{},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		got := deriveUpdateInfoFromMetadataDiff(tc.metadataDiff, tc.config)
		require.Equal(t, tc.want, got)
	}
}

func TestApplyUpdateInfoDiffRootNode(t *testing.T) {
	time1 := timestamppb.Now()
	time2 := timestamppb.New(time1.AsTime().Add(1 * time.Second))
	testCases := []struct {
		updateInfoDiffRootNode *updateInfoDiffRootNode
		config                 *storepb.DatabaseConfig
		want                   *storepb.DatabaseConfig
	}{
		{
			updateInfoDiffRootNode: &updateInfoDiffRootNode{
				schemas: map[string]*updateInfoDiffSchemaNode{
					"": {
						tables: map[string]*updateInfoDiffTableNode{
							"t1": {
								name: "t1",
								diffBaseNode: diffBaseNode{
									action: diffActionCreate,
								},
								updateInfo: &updateInfo{
									lastUpdatedTime: time2,
									lastUpdater:     "anonymous+01@bytebase.com",
									sourceBranch:    "feat/01",
								},
							},
							"t2": {
								name: "t2",
								diffBaseNode: diffBaseNode{
									action: diffActionDrop,
								},
								updateInfo: nil,
							},
						},
						views:      map[string]*updateInfoDiffViewNode{},
						functions:  map[string]*updateInfoDiffFunctionNode{},
						procedures: map[string]*updateInfoDiffProcedureNode{},
					},
				},
			},
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "",
						Tables: []*storepb.TableCatalog{
							{
								Name:       "t1",
								Updater:    "my",
								UpdateTime: time1,
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "t",
										Classification: "1-1-1",
									},
								},
							},
							{
								Name:       "t2",
								Updater:    "my",
								UpdateTime: time1,
							},
						},
					},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "",
						Tables: []*storepb.TableCatalog{
							{
								Name:         "t1",
								UpdateTime:   time2,
								Updater:      "anonymous+01@bytebase.com",
								SourceBranch: "feat/01",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "t",
										Classification: "1-1-1",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		got := applyUpdateInfoDiffRootNode(tc.updateInfoDiffRootNode, tc.config)
		require.Equal(t, tc.want, got)
	}
}
