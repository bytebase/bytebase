package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/permission"
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

	tests := []struct {
		name    string
		columns []parserbase.ColumnResource
		want    string
	}{
		{"local only", []parserbase.ColumnResource{local}, ""},
		{"link resolved to the connected instance", []parserbase.ColumnResource{local, sameInstance}, ""},
		{"link resolved to another instance", []parserbase.ColumnResource{local, otherInstance}, `table APP.T reached through database link "REMOTE3" is on instance "oracle-remote"`},
		{"unresolved link", []parserbase.ColumnResource{local, unresolved}, `table SECRET_SCHEMA.SECRET_T reached through database link "REMOTE2" cannot be authorized: Bytebase could not establish which database it reaches. In this release a linked table is authorized only for a query running under the admin data source, whose database links Bytebase synced, and only when the connect string of the link names the host, port and service of a data source of exactly one instance; on an instance with a read-only data source the query-data policy must allow the admin data source`},
		{"unresolved link, unqualified table", []parserbase.ColumnResource{unresolvedUnqualified}, `table SECRET_T reached through database link "REMOTE2"`},
		{"unresolved link naming a local database", []parserbase.ColumnResource{unresolvedLocalName}, `table ALLOWED_S.ALLOWED_T reached through database link "REMOTE2"`},
	}
	for _, tc := range tests {
		columns := make(parserbase.SourceColumnSet, len(tc.columns))
		for _, column := range tc.columns {
			columns[column] = true
		}
		got := remoteColumnRefusal(columns, connected, permission.SQLSelect)
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

func TestLinkedTargetProjectRefusal(t *testing.T) {
	column := parserbase.ColumnResource{Instance: "oracle-dblink", Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "SECRET_T"}
	require.Empty(t, linkedTargetProjectRefusal(column, &store.DatabaseMessage{ProjectID: "default"}, "default"))
	require.Contains(t, linkedTargetProjectRefusal(column, nil, "default"), `SECRET_SCHEMA.SECRET_T reached through database link "REMOTE" is in a database Bytebase does not track`)
	require.Contains(t, linkedTargetProjectRefusal(column, &store.DatabaseMessage{ProjectID: "other"}, "default"), `is in project "other"`)
}
