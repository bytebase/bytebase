package parsercontext

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	oracledriver "github.com/bytebase/omni/oracle/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestParseOracleConnectStringAndDataSourceMatch(t *testing.T) {
	// Pure connect-string parsing edge cases live in omni/oracle/driver; this
	// verifies the seam from omni's ConnectTarget into Bytebase DataSource matching.
	tests := []struct {
		name          string
		connectString string
		dataSource    *storepb.DataSource
		wantParsed    bool
		wantMatch     bool
	}{
		{
			name:          "easy connect plus multi-host shared port",
			connectString: "rac1,rac2:1522/FREEPDB1?connect_timeout=5",
			dataSource:    &storepb.DataSource{Host: "RAC1", Port: "1522", ServiceName: "freepdb1"},
			wantParsed:    true,
			wantMatch:     true,
		},
		{
			name:          "tns descriptor with SID",
			connectString: "(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=db)(PORT=1522))(CONNECT_DATA=(SID=orcl)))",
			dataSource:    &storepb.DataSource{Host: "db", Port: "1522", Sid: "ORCL"},
			wantParsed:    true,
			wantMatch:     true,
		},
		{
			name:          "tns alias rejected",
			connectString: "PRODDB",
			dataSource:    &storepb.DataSource{Host: "db", Port: "1521", ServiceName: "PRODDB"},
			wantParsed:    false,
		},
		{
			name:          "source route rejected",
			connectString: "(DESCRIPTION=(SOURCE_ROUTE=yes)(ADDRESS=(HOST=cman)(PORT=1521))(ADDRESS=(HOST=real)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=svc)))",
			dataSource:    &storepb.DataSource{Host: "real", Port: "1521", ServiceName: "svc"},
			wantParsed:    false,
		},
	}
	for _, tc := range tests {
		target, ok := oracledriver.ParseConnectString(tc.connectString)
		require.Equal(t, tc.wantParsed, ok, tc.name)
		if !tc.wantParsed {
			continue
		}
		require.Equal(t, tc.wantMatch, dataSourceReachesLinkTarget(target, tc.dataSource), tc.name)
	}
}

func TestDataSourceReachesLinkTarget(t *testing.T) {
	endpoints := []oracledriver.Endpoint{{Host: "10.0.0.50", Port: "1521"}, {Host: "rac2-vip", Port: "1522"}}
	byService := oracledriver.ConnectTarget{Endpoints: endpoints, Service: "FREEPDB1"}
	bySID := oracledriver.ConnectTarget{Endpoints: endpoints, SID: "FREEPDB1"}
	tests := []struct {
		name       string
		target     oracledriver.ConnectTarget
		dataSource *storepb.DataSource
		want       bool
	}{
		{"exact", byService, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", ServiceName: "FREEPDB1"}, true},
		{"host case and service case", byService, &storepb.DataSource{Host: "RAC2-VIP", Port: "1522", ServiceName: "freepdb1"}, true},
		{"sid link, sid data source", bySID, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", Sid: "freepdb1"}, true},
		// go-ora connects by SID whenever one is set, so that is the identity compared.
		{"sid link, data source with both", bySID, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", ServiceName: "OTHER", Sid: "FREEPDB1"}, true},
		{"service link, data source with both", byService, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", ServiceName: "FREEPDB1", Sid: "FREEPDB1"}, false},
		{"service link, sid data source", byService, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", Sid: "FREEPDB1"}, false},
		{"sid link, service data source", bySID, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", ServiceName: "FREEPDB1"}, false},
		{"empty host", byService, &storepb.DataSource{Port: "1521", ServiceName: "FREEPDB1"}, false},
		{"empty port", byService, &storepb.DataSource{Host: "10.0.0.50", ServiceName: "FREEPDB1"}, false},
		{"host substring", byService, &storepb.DataSource{Host: "10.0.0.5", Port: "1521", ServiceName: "FREEPDB1"}, false},
		{"host superstring", byService, &storepb.DataSource{Host: "10.0.0.500", Port: "1521", ServiceName: "FREEPDB1"}, false},
		{"port of the other address", byService, &storepb.DataSource{Host: "10.0.0.50", Port: "1522", ServiceName: "FREEPDB1"}, false},
		{"service", byService, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", ServiceName: "OTHER"}, false},
		{"service with a domain the link lacks", byService, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", ServiceName: "FREEPDB1.corp.com"}, false},
		{"no service or sid", byService, &storepb.DataSource{Host: "10.0.0.50", Port: "1521"}, false},
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, dataSourceReachesLinkTarget(tc.target, tc.dataSource), tc.name)
	}
	domained := oracledriver.ConnectTarget{Endpoints: endpoints, Service: "FREEPDB1.corp.com"}
	require.False(t, dataSourceReachesLinkTarget(domained, &storepb.DataSource{Host: "10.0.0.50", Port: "1521", ServiceName: "FREEPDB1"}))
}

func TestSelectLinkDefinition(t *testing.T) {
	link := func(name, host, user string) *metadatapb.LinkedDatabaseMetadata {
		return &metadatapb.LinkedDatabaseMetadata{Name: name, Host: host, Username: user}
	}
	remote := link("REMOTE", "h1:1521/s", "U1")
	tests := []struct {
		name      string
		links     []*metadatapb.LinkedDatabaseMetadata
		want      *metadatapb.LinkedDatabaseMetadata
		wantFound bool
	}{
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{remote}, remote, true},
		{"remote", []*metadatapb.LinkedDatabaseMetadata{remote}, remote, true},
		{"REMOTE", nil, nil, false},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("OTHER", "h1:1521/s", "U1")}, nil, false},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("REMOTEX", "h1:1521/s", "U1")}, nil, false},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE.WORLD", "h1:1521/s", "U1")}, link("REMOTE.WORLD", "h1:1521/s", "U1"), true},
		// Oracle follows REMOTE when its global name has no domain and REMOTE.WORLD when the
		// domain is WORLD, so the two resolve only when they reach the same target.
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE.WORLD", "h2:1521/s", "U2"), remote}, nil, true},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{remote, link("REMOTE.WORLD", "h1:1521/s", "U1")}, remote, true},
		{"REMOTE@Q", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE@Q", "h1:1521/s", "U1"), link("REMOTE.WORLD@Q", "h2:1521/s", "U2")}, nil, true},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{remote, link("REMOTE", "h1:1521/s", "u1")}, remote, true},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE.WORLD", "h1:1521/s", "U1"), link("REMOTE.CORP", "h1:1521/s", "U1")}, link("REMOTE.WORLD", "h1:1521/s", "U1"), true},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{remote, link("REMOTE", "h2:1521/s", "U1")}, nil, true},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{remote, link("REMOTE", "h1:1521/s", "U2")}, nil, true},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE.WORLD", "h1:1521/s", "U1"), link("REMOTE.CORP", "h2:1521/s", "U1")}, nil, true},
		// REMOTE.WORLD@Q is the qualified link Q of REMOTE.WORLD, not a domained REMOTE.
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE.WORLD@Q", "h2:1521/s", "U2")}, nil, false},
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE.WORLD", "h1:1521/s", "U1"), link("REMOTE.WORLD@Q", "h2:1521/s", "U2")}, link("REMOTE.WORLD", "h1:1521/s", "U1"), true},
		{"REMOTE@Q", []*metadatapb.LinkedDatabaseMetadata{remote, link("REMOTE@Q", "h2:1521/s", "U2"), link("REMOTE@R", "h3:1521/s", "U3")}, link("REMOTE@Q", "h2:1521/s", "U2"), true},
		{"remote@q", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE@Q", "h2:1521/s", "U2")}, link("REMOTE@Q", "h2:1521/s", "U2"), true},
		{"REMOTE@Q", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE.WORLD@Q", "h2:1521/s", "U2")}, link("REMOTE.WORLD@Q", "h2:1521/s", "U2"), true},
		{"REMOTE@Q", []*metadatapb.LinkedDatabaseMetadata{remote, link("REMOTE.WORLD@R", "h3:1521/s", "U3")}, nil, false},
	}
	for _, tc := range tests {
		got, found := selectLinkDefinition(tc.name, tc.links)
		require.Equal(t, tc.wantFound, found, tc.name)
		if tc.want == nil {
			require.Nil(t, got, tc.name)
			continue
		}
		require.NotNil(t, got, tc.name)
		require.Equal(t, tc.want.GetName(), got.GetName())
		require.Equal(t, tc.want.GetHost(), got.GetHost())
	}
}
