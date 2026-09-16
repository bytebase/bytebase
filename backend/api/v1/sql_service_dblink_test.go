package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/permission"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

func TestRemoteColumnRefusal(t *testing.T) {
	const connected = "oracle-dblink"
	local := parserbase.ColumnResource{Database: "ALLOWED_S", Table: "ALLOWED_T"}
	sameInstance := parserbase.ColumnResource{Instance: connected, Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "SECRET_T"}
	otherInstance := parserbase.ColumnResource{Instance: "oracle-remote", Server: "REMOTE3", Database: "APP", Table: "T"}
	unresolved := parserbase.ColumnResource{Server: "REMOTE2", Database: "SECRET_SCHEMA", Table: "SECRET_T"}
	unresolvedUnqualified := parserbase.ColumnResource{Server: "REMOTE2", Table: "SECRET_T"}
	unresolvedLocalName := parserbase.ColumnResource{Server: "REMOTE2", Database: "ALLOWED_S", Table: "ALLOWED_T"}
	linkedServerShaped := parserbase.ColumnResource{Server: "SRV", Database: "db", Schema: "dbo", Table: "t"}

	tests := []struct {
		name    string
		columns []parserbase.ColumnResource
		want    string
	}{
		{"local only", []parserbase.ColumnResource{local}, ""},
		{"link resolved to the connected instance", []parserbase.ColumnResource{local, sameInstance}, ""},
		{"link resolved to another instance", []parserbase.ColumnResource{local, otherInstance}, `table APP.T reached through database link "REMOTE3" is on instance "oracle-remote"`},
		{"unresolved link", []parserbase.ColumnResource{local, unresolved}, `table SECRET_SCHEMA.SECRET_T reached through remote server "REMOTE2" cannot be authorized: Bytebase could not establish which database it reaches`},
		{"unresolved link, unqualified table", []parserbase.ColumnResource{unresolvedUnqualified}, `table SECRET_T reached through remote server "REMOTE2"`},
		{"unresolved link naming a local database", []parserbase.ColumnResource{unresolvedLocalName}, `table ALLOWED_S.ALLOWED_T reached through remote server "REMOTE2"`},
		{"linked-server-shaped column", []parserbase.ColumnResource{linkedServerShaped}, `table db.dbo.t reached through remote server "SRV" cannot be authorized`},
	}
	for _, tc := range tests {
		columns := make(parserbase.SourceColumnSet, len(tc.columns))
		for _, column := range tc.columns {
			columns[column] = true
		}
		got := remoteColumnRefusal(columns, connected, storepb.Engine_ORACLE, permission.SQLSelect)
		if tc.want == "" {
			require.Nil(t, got, tc.name)
			continue
		}
		require.NotNil(t, got, tc.name)
		require.Contains(t, got.Error(), tc.want, tc.name)
		require.Nil(t, got.resources, "%s: the refusal must offer no resource to request access to", tc.name)
		require.Equal(t, permission.SQLSelect, got.permission, tc.name)
	}
}

func TestRemoteColumnRefusalAdviceIsOracleOnly(t *testing.T) {
	columns := parserbase.SourceColumnSet{parserbase.ColumnResource{Server: "SRV", Database: "db", Schema: "dbo", Table: "t"}: true}
	oracle := remoteColumnRefusal(columns, "inst", storepb.Engine_ORACLE, permission.SQLSelect)
	require.Contains(t, oracle.Error(), "admin data source")
	mssql := remoteColumnRefusal(columns, "inst", storepb.Engine_MSSQL, permission.SQLSelect)
	require.NotNil(t, mssql)
	require.NotContains(t, mssql.Error(), "admin data source")
	require.Contains(t, mssql.Error(), `table db.dbo.t reached through remote server "SRV" cannot be authorized`)
}

func TestLinkedTargetProjectRefusal(t *testing.T) {
	column := parserbase.ColumnResource{Instance: "oracle-dblink", Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "SECRET_T"}
	require.Empty(t, linkedTargetProjectRefusal(column, &store.DatabaseMessage{ProjectID: "default"}, "default"))
	require.Contains(t, linkedTargetProjectRefusal(column, nil, "default"), `SECRET_SCHEMA.SECRET_T reached through database link "REMOTE" is in a database Bytebase does not track`)
	require.Contains(t, linkedTargetProjectRefusal(column, &store.DatabaseMessage{ProjectID: "other"}, "default"), `is in project "other"`)
}
