package base

import storepb "github.com/bytebase/bytebase/proto/generated-go/store"

func BackupDatabaseNameOfEngine(e storepb.Engine) string {
	//exhaustive:enforce
	switch e {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB,
		storepb.Engine_MSSQL,
		storepb.Engine_POSTGRES:
		return "bbdataarchive"
	case
		storepb.Engine_ORACLE:
		return "BBDATAARCHIVE"
	default:
		return ""
	}
}
