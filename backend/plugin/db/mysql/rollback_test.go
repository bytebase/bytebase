package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRollbackSQL(t *testing.T) {
	tests := []struct {
		name         string
		txn          BinlogTransaction
		tableCatalog map[string][]string
		rollbackSQL  string
		err          bool
	}{
		{
			name:         "empty",
			txn:          BinlogTransaction{},
			tableCatalog: map[string][]string{},
			rollbackSQL:  "",
			err:          false,
		},
		{
			name: "INSERT",
			txn: BinlogTransaction{
				{
					Type:   QueryEventType,
					Header: "#221017 14:25:24 server id 1  end_log_pos 772 CRC32 0x37cb53f6 	Query	thread_id=53771	exec_time=0	error_code=0\n",
					Body: `SET TIMESTAMP=1665987924/*!*/;
BEGIN
/*!*/;
`,
				},
				{
					Type:   WriteRowsEventType,
					Header: "#221017 14:25:24 server id 1  end_log_pos 916 CRC32 0x896854fc 	Write_rows: table id 259 flags: STMT_END_F\n",
					Body: `### INSERT INTO ` + "`binlog_test`.`user`" + `
### SET
###   @1=1
###   @2='alice'
###   @3=100
### INSERT INTO ` + "`binlog_test`.`user`" + `
### SET
###   @1=2
###   @2='bob'
###   @3=100
### INSERT INTO ` + "`binlog_test`.`user`" + `
### SET
###   @1=3
###   @2='cindy'
###   @3=100`,
				},
				{
					Type:   XidEventType,
					Header: "#221017 14:25:24 server id 1  end_log_pos 947 CRC32 0xaf8e8303 	Xid = 327602\n",
					Body: `COMMIT/*!*/;
`,
				},
			},
			tableCatalog: map[string][]string{
				"user": {"id", "name", "balance"},
			},
			rollbackSQL: `DELETE FROM ` + "`binlog_test`.`user`" + `
WHERE
  ` + "`id`" + `<=>1 AND
  ` + "`name`" + `<=>'alice' AND
  ` + "`balance`" + `<=>100;
DELETE FROM ` + "`binlog_test`.`user`" + `
WHERE
  ` + "`id`" + `<=>2 AND
  ` + "`name`" + `<=>'bob' AND
  ` + "`balance`" + `<=>100;
DELETE FROM ` + "`binlog_test`.`user`" + `
WHERE
  ` + "`id`" + `<=>3 AND
  ` + "`name`" + `<=>'cindy' AND
  ` + "`balance`" + `<=>100;`,
			err: false,
		},
		{
			name: "UPDATE",
			txn: BinlogTransaction{
				{
					Type:   QueryEventType,
					Header: "#221017 14:25:53 server id 1  end_log_pos 1117 CRC32 0x5842528e 	Query	thread_id=53771	exec_time=0	error_code=0\n",
					Body: `SET TIMESTAMP=1665987953/*!*/;
BEGIN
/*!*/;
`,
				},
				{
					Type:   UpdateRowsEventType,
					Header: "#221017 14:25:53 server id 1  end_log_pos 1249 CRC32 0x3d8fa43e 	Update_rows: table id 259 flags: STMT_END_F\n",
					Body: `### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=100
### SET
###   @1=1
###   @2='alice'
###   @3=90`,
				},
				{
					Type:   UpdateRowsEventType,
					Header: "#221017 14:26:08 server id 1  end_log_pos 1377 CRC32 0xd7bb3662 	Update_rows: table id 259 flags: STMT_END_F\n",
					Body: `### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=100
### SET
###   @1=2
###   @2='bob'
###   @3=110`,
				},
				{
					Type:   XidEventType,
					Header: "#221017 14:26:12 server id 1  end_log_pos 1408 CRC32 0xf2dd63fe 	Xid = 327607\n",
					Body: `COMMIT/*!*/;
`,
				},
			},
			tableCatalog: map[string][]string{
				"user": {"id", "name", "balance"},
			},
			rollbackSQL: `UPDATE ` + "`binlog_test`.`user`" + `
SET
  ` + "`id`" + `=2,
  ` + "`name`" + `='bob',
  ` + "`balance`" + `=100
WHERE
  ` + "`id`" + `<=>2 AND
  ` + "`name`" + `<=>'bob' AND
  ` + "`balance`" + `<=>110;
UPDATE ` + "`binlog_test`.`user`" + `
SET
  ` + "`id`" + `=1,
  ` + "`name`" + `='alice',
  ` + "`balance`" + `=100
WHERE
  ` + "`id`" + `<=>1 AND
  ` + "`name`" + `<=>'alice' AND
  ` + "`balance`" + `<=>90;`,
			err: false,
		},
		{
			name: "DELETE",
			txn: BinlogTransaction{
				{
					Type:   QueryEventType,
					Header: "#221018 16:21:45 server id 1  end_log_pos 2236 CRC32 0x965db1d1 	Query	thread_id=58599	exec_time=0	error_code=0\n",
					Body: `SET TIMESTAMP=1666081305/*!*/;
BEGIN
/*!*/;
`,
				},
				{
					Type:   DeleteRowsEventType,
					Header: "#221018 16:21:45 server id 1  end_log_pos 2365 CRC32 0xf759c90c 	Delete_rows: table id 259 flags: STMT_END_F\n",
					Body: `### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=0
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=0`,
				},
				{
					Type:   XidEventType,
					Header: "#221018 16:21:45 server id 1  end_log_pos 2396 CRC32 0x816695ae 	Xid = 349604\n",
					Body: `COMMIT/*!*/;
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
DELIMITER ;
# End of log file
/*!50003 SET COMPLETION_TYPE=@OLD_COMPLETION_TYPE*/;
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=0*/;`,
				},
			},
			tableCatalog: map[string][]string{
				"user": {"id", "name", "balance"},
			},
			rollbackSQL: `INSERT INTO ` + "`binlog_test`.`user`" + `
SET
  ` + "`id`" + `=1,
  ` + "`name`" + `='alice',
  ` + "`balance`" + `=0;
INSERT INTO ` + "`binlog_test`.`user`" + `
SET
  ` + "`id`" + `=2,
  ` + "`name`" + `='bob',
  ` + "`balance`" + `=0;`,
			err: false,
		},
		{
			name: "schema changed",
			txn: BinlogTransaction{
				{
					Type:   DeleteRowsEventType,
					Header: "#221018 16:21:45 server id 1  end_log_pos 2365 CRC32 0xf759c90c 	Delete_rows: table id 259 flags: STMT_END_F\n",
					Body: `### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=0`,
				},
			},
			tableCatalog: map[string][]string{
				"user": {"id", "name", "balance", "new_column"},
			},
			rollbackSQL: "",
			err:         true,
		},
		{
			name: "hand-crafted DELETE with an event having empty body",
			txn: BinlogTransaction{
				{
					Type:   QueryEventType,
					Header: "#221018 16:21:45 server id 1  end_log_pos 2236 CRC32 0x965db1d1 	Query	thread_id=58599	exec_time=0	error_code=0\n",
					Body: `SET TIMESTAMP=1666081305/*!*/;
BEGIN
/*!*/;
`,
				},
				{
					Type:   DeleteRowsEventType,
					Header: "#221018 16:21:45 server id 1  end_log_pos 2365 CRC32 0xf759c90c 	Delete_rows: table id 259\n",
					Body:   "",
				},
				{
					Type:   DeleteRowsEventType,
					Header: "#221018 16:21:45 server id 1  end_log_pos 2365 CRC32 0xf759c90c 	Delete_rows: table id 259 flags: STMT_END_F\n",
					Body: `### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=0
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=0`,
				},
				{
					Type:   XidEventType,
					Header: "#221018 16:21:45 server id 1  end_log_pos 2396 CRC32 0x816695ae 	Xid = 349604\n",
					Body: `COMMIT/*!*/;
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
DELIMITER ;
# End of log file
/*!50003 SET COMPLETION_TYPE=@OLD_COMPLETION_TYPE*/;
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=0*/;`,
				},
			},
			tableCatalog: map[string][]string{
				"user": {"id", "name", "balance"},
			},
			rollbackSQL: `INSERT INTO ` + "`binlog_test`.`user`" + `
SET
  ` + "`id`" + `=1,
  ` + "`name`" + `='alice',
  ` + "`balance`" + `=0;
INSERT INTO ` + "`binlog_test`.`user`" + `
SET
  ` + "`id`" + `=2,
  ` + "`name`" + `='bob',
  ` + "`balance`" + `=0;`,
			err: false,
		},
	}

	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			a := require.New(t)
			sql, err := test.txn.GetRollbackSQL(test.tableCatalog)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(test.rollbackSQL, sql)
			}
		})
	}
}

func TestGetTableColumns(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		tableMap map[string][]string
		err      bool
	}{
		{
			name: "multiple tables",
			schema: `
CREATE TABLE user (
	id INT PRIMARY KEY,
	name VARCHAR(20)
);
CREATE TABLE balance (
	id INT PRIMARY KEY,
	user_id INT REFERENCES user(id),
	balance INT
);`,
			tableMap: map[string][]string{
				"user":    {"id", "name"},
				"balance": {"id", "user_id", "balance"},
			},
			err: false,
		},
	}
	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			a := require.New(t)
			tableMap, err := GetTableColumns(test.schema)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(test.tableMap, tableMap)
			}
		})
	}
}
