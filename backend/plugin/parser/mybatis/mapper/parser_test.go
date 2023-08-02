// Package mapper defines the sql extractor for mybatis mapper xml.
package mapper

import (
	"io"
	"os"
	"strings"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v4"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/mybatis/mapper/ast"
)

// TestData is the test data for mybatis parser. It contains the xml and the expected sql.
// And the sql is expected to be restored from the xml.
type TestData struct {
	XML string `yaml:"xml"`
	SQL string `yaml:"sql"`
}

// runTest is a helper function to run the test.
// It will parse the xml given by `filepath` and compare the restored sql with `sql`.
func runTest(t *testing.T, filepath string, record bool) {
	var testCases []TestData
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &testCases)
	require.NoError(t, err)

	for i, testCase := range testCases {
		parser := NewParser(testCase.XML)
		node, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, node)

		var stringsBuilder strings.Builder
		err = node.RestoreSQL(parser.GetRestoreContext(), &stringsBuilder)
		require.NoError(t, err)
		require.NoError(t, err)
		if record {
			testCases[i].SQL = stringsBuilder.String()
		} else {
			require.Equal(t, testCase.SQL, stringsBuilder.String())
		}
		// The result should be parsed correctly by MySQL parser.
		_, err = pg_query.Parse(testCases[i].SQL)
		require.NoError(t, err, "failed to parse restored sql: %s", testCases[i].SQL)
	}

	if record {
		err := yamlFile.Close()
		require.NoError(t, err)
		byteValue, err = yaml.Marshal(testCases)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
	}
}

func TestParser(t *testing.T) {
	testFileList := []string{
		"test-data/test_simple_mapper.yaml",
		"test-data/test_dynamic_sql_mapper.yaml",
	}
	for _, filepath := range testFileList {
		runTest(t, filepath, false)
	}
}

func TestRestoreWithLineMapping(t *testing.T) {
	testCases := []struct {
		xml         string
		sql         string
		lineMapping []*ast.MybatisSQLLineMapping
	}{
		{
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN" "http://mybatis.org/dtd/mybatis-3-mapper.dtd">
<mapper namespace="com.bytebase.test">
	<sql id="sometable">
		${prefix}Table
	</sql>
	<sql id="someinclude">
		from
		<include refid="${include_target}"/>
	</sql>
	<select id="select" resultType="map">
		select
		field1, field2, field3
		<include refid="someinclude">
			<property name="prefix" value="Some"/>
			<property name="include_target" value="sometable"/>
		</include>
	</select>
</mapper>
			`,
			sql: `select
		field1, field2, field3 from SomeTable;
`,
			lineMapping: []*ast.MybatisSQLLineMapping{
				{
					SQLLastLine:     2,
					OriginalEleLine: 11,
				},
			},
		},
	}
	for _, tc := range testCases {
		parser := NewParser(tc.xml)
		node, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, node)
		var sb strings.Builder
		lineMapping, err := node.RestoreSQLWithLineMapping(parser.GetRestoreContext(), &sb)
		require.NoError(t, err)
		require.Equal(t, tc.sql, sb.String())
		require.Equal(t, tc.lineMapping, lineMapping)
	}
}
