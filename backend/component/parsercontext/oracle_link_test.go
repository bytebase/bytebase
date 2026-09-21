package parsercontext

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestParseOracleLinkTarget(t *testing.T) {
	ep := func(host, port string) oracleEndpoint { return oracleEndpoint{host: host, port: port} }
	svc := func(service string, endpoints ...oracleEndpoint) oracleLinkTarget {
		return oracleLinkTarget{endpoints: endpoints, service: service}
	}
	sid := func(sid string, endpoints ...oracleEndpoint) oracleLinkTarget {
		return oracleLinkTarget{endpoints: endpoints, sid: sid}
	}
	tests := []struct {
		connectString string
		want          oracleLinkTarget
		ok            bool
	}{
		{"localhost:1521/FREEPDB1", svc("FREEPDB1", ep("localhost", "1521")), true},
		{"//db.example.com:1522/ORCL", svc("ORCL", ep("db.example.com", "1522")), true},
		{"DB.EXAMPLE.COM/orcl", svc("orcl", ep("db.example.com", "1521")), true},
		{"host:1521/svc:dedicated/inst1", svc("svc", ep("host", "1521")), true},
		{"[::1]:1521/svc", svc("svc", ep("::1", "1521")), true},
		{"[::1]/svc", svc("svc", ep("::1", "1521")), true},
		{"  host:1521/svc  ", svc("svc", ep("host", "1521")), true},
		// Easy Connect Plus: scheme, parameters, several addresses.
		{"tcps://adb.example.com:1522/abc_high.adb.oraclecloud.com", svc("abc_high.adb.oraclecloud.com", ep("adb.example.com", "1522")), true},
		{"//host:1521/svc?connect_timeout=5&retry_count=2", svc("svc", ep("host", "1521")), true},
		{"rac1:1521,rac2:1522/svc", svc("svc", ep("rac1", "1521"), ep("rac2", "1522")), true},
		// A port applies to the hosts listed before it.
		{"rac1,rac2:1522/svc", svc("svc", ep("rac1", "1522"), ep("rac2", "1522")), true},
		{"rac1,rac2:1522,rac3/svc", svc("svc", ep("rac1", "1522"), ep("rac2", "1522"), ep("rac3", "1521")), true},
		{"[::1],[::2]:1522/svc", svc("svc", ep("::1", "1522"), ep("::2", "1522")), true},
		{"(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=db.example.com)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=orcl)))", svc("orcl", ep("db.example.com", "1521")), true},
		{"(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=db)(PORT=1522))(CONNECT_DATA=(SID=orcl)))", sid("orcl", ep("db", "1522")), true},
		{"(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=db))(CONNECT_DATA=(SERVICE_NAME=orcl)))", svc("orcl", ep("db", "1521")), true},
		{"( DESCRIPTION = ( ADDRESS = ( HOST = db ) ( PORT = 1521 ) ) ( CONNECT_DATA = ( SERVICE_NAME = orcl ) ) )", svc("orcl", ep("db", "1521")), true},
		// RAC and Data Guard: several addresses, one service.
		{"(DESCRIPTION=(ADDRESS_LIST=(LOAD_BALANCE=on)(ADDRESS=(PROTOCOL=TCP)(HOST=rac1-vip)(PORT=1521))(ADDRESS=(PROTOCOL=TCP)(HOST=rac2-vip)(PORT=1521)))(CONNECT_DATA=(SERVICE_NAME=proddb)))", svc("proddb", ep("rac1-vip", "1521"), ep("rac2-vip", "1521")), true},
		// A TNS alias names nothing on its own.
		{"PRODDB", oracleLinkTarget{}, false},
		{"", oracleLinkTarget{}, false},
		// No service or SID.
		{"host:1521", oracleLinkTarget{}, false},
		{"host:1521/", oracleLinkTarget{}, false},
		{"host:1521/?connect_timeout=5", oracleLinkTarget{}, false},
		{"/svc", oracleLinkTarget{}, false},
		{"(DESCRIPTION=(ADDRESS=(HOST=a)(PORT=1521)))", oracleLinkTarget{}, false},
		// A DESCRIPTION_LIST repeating one service is one database (Data Guard failover).
		{"(DESCRIPTION_LIST=(DESCRIPTION=(ADDRESS=(HOST=h1)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=proddb)))(DESCRIPTION=(ADDRESS=(HOST=h2)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=PRODDB))))", svc("proddb", ep("h1", "1521"), ep("h2", "1521")), true},
		// A SOURCE_ROUTE chain passes through a proxy; its first address is not the database.
		{"(DESCRIPTION=(SOURCE_ROUTE=yes)(ADDRESS=(HOST=cman)(PORT=1521))(ADDRESS=(HOST=real)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=svc)))", oracleLinkTarget{}, false},
		// A DESCRIPTION_LIST that switches service between addresses names two databases.
		{"(DESCRIPTION_LIST=(DESCRIPTION=(ADDRESS=(HOST=h1)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=proddb)))(DESCRIPTION=(ADDRESS=(HOST=h2)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=reportdb))))", oracleLinkTarget{}, false},
		// A service and a SID are different identities even when spelled the same.
		{"(DESCRIPTION=(ADDRESS=(HOST=a)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=orcl)(SID=orcl)))", oracleLinkTarget{}, false},
		{"(DESCRIPTION=(ADDRESS=(HOST=a)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=orcl)(SID=other)))", oracleLinkTarget{}, false},
		{"(DESCRIPTION_LIST=(DESCRIPTION=(ADDRESS=(HOST=h1)(PORT=1521))(CONNECT_DATA=(SID=orcl)))(DESCRIPTION=(ADDRESS=(HOST=h2)(PORT=1521))(CONNECT_DATA=(SID=ORCL))))", sid("orcl", ep("h1", "1521"), ep("h2", "1521")), true},
		// Malformed.
		{"host1 host2:1521/svc", oracleLinkTarget{}, false},
		{"rac1:1521,/svc", oracleLinkTarget{}, false},
		{"rac1:/svc", oracleLinkTarget{}, false},
		{"[::1:1521/svc", oracleLinkTarget{}, false},
		{"[::1]1521/svc", oracleLinkTarget{}, false},
		{"(DESCRIPTION=(ADDRESS=(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=orcl)))", oracleLinkTarget{}, false},
		{"(DESCRIPTION=(CONNECT_DATA=(SERVICE_NAME=orcl)))", oracleLinkTarget{}, false},
	}
	for _, tc := range tests {
		got, ok := parseOracleLinkTarget(tc.connectString)
		require.Equal(t, tc.ok, ok, tc.connectString)
		require.Equal(t, tc.want, got, tc.connectString)
	}
}

func TestDataSourceReachesLinkTarget(t *testing.T) {
	endpoints := []oracleEndpoint{{host: "10.0.0.50", port: "1521"}, {host: "rac2-vip", port: "1522"}}
	byService := oracleLinkTarget{endpoints: endpoints, service: "FREEPDB1"}
	bySID := oracleLinkTarget{endpoints: endpoints, sid: "FREEPDB1"}
	tests := []struct {
		name       string
		target     oracleLinkTarget
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
	domained := oracleLinkTarget{endpoints: endpoints, service: "FREEPDB1.corp.com"}
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
		{"REMOTE", []*metadatapb.LinkedDatabaseMetadata{link("REMOTE.WORLD", "h2:1521/s", "U2"), remote}, remote, true},
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
