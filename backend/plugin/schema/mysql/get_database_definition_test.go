package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGetTableDefinitionWithCheckConstraint(t *testing.T) {
	table := &storepb.TableMetadata{
		Name: "t1",
		Columns: []*storepb.ColumnMetadata{
			{Name: "type", Type: "varchar(10)", Nullable: true, Default: "NULL"},
			{Name: "amount", Type: "decimal(10,2)", Nullable: true, Default: "NULL"},
		},
		CheckConstraints: []*storepb.CheckConstraintMetadata{
			{
				Name: "c1",
				// The expression as synced from MySQL after unescaping the
				// information_schema escaped form ((`type` = _utf8mb4\'A\')...).
				Expression: "(((`type` = _utf8mb4'A') and (`amount` is not null)))",
			},
		},
		Engine:    "InnoDB",
		Charset:   "utf8mb4",
		Collation: "utf8mb4_general_ci",
	}

	got, err := GetTableDefinition("", table, nil)
	require.NoError(t, err)

	want := "--\n" +
		"-- Table structure for `t1`\n" +
		"--\n" +
		"CREATE TABLE `t1` (\n" +
		"  `type` varchar(10) DEFAULT NULL,\n" +
		"  `amount` decimal(10,2) DEFAULT NULL,\n" +
		"  CONSTRAINT `c1` CHECK (((`type` = _utf8mb4'A') and (`amount` is not null)))\n" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\n"
	require.Equal(t, want, got)
	require.NotContains(t, got, `\'`)
}

func TestGetFunctionDefinitionWithMultilineParameters(t *testing.T) {
	function := &storepb.FunctionMetadata{
		Name: "f1",
		// The definition as synced from SHOW CREATE FUNCTION: the parameter
		// list keeps its original multi-line formatting, and the CHARSET and
		// COLLATE attributes of the RETURNS clause have been stripped.
		Definition: "CREATE FUNCTION `f1`(\n" +
			"    p_a BIGINT,\n" +
			"    p_b VARCHAR(36)\n" +
			") RETURNS char(36)\n" +
			"    NO SQL\n" +
			"    DETERMINISTIC\n" +
			"BEGIN\n" +
			"    RETURN p_b;\n" +
			"END",
		CharacterSetClient:  "utf8mb4",
		CollationConnection: "utf8mb4_unicode_ci",
		SqlMode:             "ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES",
	}

	got, err := GetFunctionDefinition("", function)
	require.NoError(t, err)

	want := "--\n" +
		"-- Function structure for `f1`\n" +
		"--\n" +
		"SET character_set_client = utf8mb4;\n" +
		"SET character_set_results = utf8mb4;\n" +
		"SET collation_connection = utf8mb4_unicode_ci;\n" +
		"SET sql_mode = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES';\n" +
		"CREATE FUNCTION `f1`(\n" +
		"    p_a BIGINT,\n" +
		"    p_b VARCHAR(36)\n" +
		") RETURNS char(36)\n" +
		"    NO SQL\n" +
		"    DETERMINISTIC\n" +
		"BEGIN\n" +
		"    RETURN p_b;\n" +
		"END;;\n" +
		"DELIMITER ;\n\n"
	require.Equal(t, want, got)
}
