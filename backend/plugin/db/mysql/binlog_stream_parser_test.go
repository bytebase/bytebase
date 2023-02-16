package mysql

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestErrExceedSizeLimit(t *testing.T) {
	a := require.New(t)
	err := errors.New("err1")
	a.False(IsErrExceedSizeLimit(err))
	err = ErrExceedSizeLimit{err: err}
	a.True(IsErrExceedSizeLimit(err))
	err = errors.Wrap(err, "err2")
	a.True(IsErrExceedSizeLimit(err))
	err = errors.Wrap(err, "err3")
	a.True(IsErrExceedSizeLimit(err))
}

func TestParseBinlogStream(t *testing.T) {
	tests := []struct {
		name      string
		sizeLimit int
		stream    string
		want      []BinlogTransaction
		err       bool
	}{
		{
			name:      "empty",
			sizeLimit: 8 * 1024 * 1024,
			stream:    "",
			want:      nil,
			err:       false,
		},
		{
			name:      "select 1",
			sizeLimit: 8 * 1024 * 1024,
			stream: `# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
DELIMITER /*!*/;
# at 44161
#230115 21:31:43 server id 1  end_log_pos 0 CRC32 0x59ab35ab 	Start: binlog v 4, server v 8.0.27 created 230115 21:31:43
# at 44161
#230210 15:08:45 server id 1  end_log_pos 44240 CRC32 0x5843f3bd 	Anonymous_GTID	last_committed=39	sequence_number=rbr_only=yes	original_committed_timestamp=1676012925154284immediate_commit_timestamp=16760129251542transaction_length=1333
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1676012925154284 (2023-02-10 15:08:45.154284 CST)
# immediate_commit_timestamp=1676012925154284 (2023-02-10 15:08:45.154284 CST)
/*!80001 SET @@session.original_commit_timestamp=1676012925154284*//*!*/;
/*!80014 SET @@session.original_server_version=80027*//*!*/;
/*!80014 SET @@session.immediate_server_version=80027*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 44240
#230210 15:08:45 server id 1  end_log_pos 44315 CRC32 0xba444d9b 	Query	thread_id=8265	exec_time=0	error_code=0
SET TIMESTAMP=1676012925/*!*/;
SET @@session.pseudo_thread_id=8265/*!*/;
SET @@session.foreign_key_checks=1, @@session.sql_auto_is_null=0, @@session.unique_checks=1, @@session.autocommit=1/*!*/;
SET @@session.sql_mode=1168113696/*!*/;
SET @@session.auto_increment_increment=1, @@session.auto_increment_offset=1/*!*/;
/*!\C utf8mb4 *//*!*/;
SET @@session.character_set_client=45,@@session.collation_connection=45,@@session.collation_server=255/*!*/;
SET @@session.lc_time_names=0/*!*/;
SET @@session.collation_database=DEFAULT/*!*/;
/*!80011 SET @@session.default_collation_for_utf8mb4=255*//*!*/;
BEGIN
/*!*/;
# at 44315
#230210 15:08:45 server id 1  end_log_pos 44419 CRC32 0x58fc3a39 	Table_map: ` + "`bytebase`.`migration_history`" + ` mapped to numb114
# at 44419
#230210 15:08:45 server id 1  end_log_pos 45463 CRC32 0xc479b87e 	Write_rows: table id 114 flags: STMT_END_F
### INSERT INTO ` + "`bytebase`.`migration_history`" + `
### SET
###   @1=12
###   @2='Me'
###   @3=1676012925
###   @4='Me'
###   @5=1676012925
###   @6='development'
###   @7='test'
###   @8=11
###   @9='UI'
###   @10='DATA'
###   @11='PENDING'
###   @12='0000.0000.0000-20230210150841'
###   @13='[test] Change data @02-10 15:06 UTC+0800 - DML(data) for database "test"'
###   @14='select 1'
###   @15='SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKFOREIGN_KEY_CHECKS=0;\n--\n-- Table structure for ` + "`tbl`" + `\n--\nCREATE TABLE ` + "`tbl`" + ` (\n  ` + "`id`" + ` int NOT NULL COMMENT \'ID\',\PRIMARY KEY (` + "`id`" + `)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\nSFOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n'
###   @16='SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKFOREIGN_KEY_CHECKS=0;\n--\n-- Table structure for ` + "`tbl`" + `\n--\nCREATE TABLE ` + "`tbl`" + ` (\n  ` + "`id`" + ` int NOT NULL COMMENT \'ID\',\PRIMARY KEY (` + "`id`" + `)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\nSFOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n'
###   @17=0
###   @18='119'
###   @19=''
# at 45463
#230210 15:08:45 server id 1  end_log_pos 45494 CRC32 0x1ed8297c 	Xid = 51728
COMMIT/*!*/;
# at 45494
#230210 15:08:45 server id 1  end_log_pos 45573 CRC32 0x92dbc581 	Anonymous_GTID	last_committed=40	sequence_number=rbr_only=yes	original_committed_timestamp=1676012925164545immediate_commit_timestamp=16760129251645transaction_length=2349
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1676012925164545 (2023-02-10 15:08:45.164545 CST)
# immediate_commit_timestamp=1676012925164545 (2023-02-10 15:08:45.164545 CST)
/*!80001 SET @@session.original_commit_timestamp=1676012925164545*//*!*/;
/*!80014 SET @@session.original_server_version=80027*//*!*/;
/*!80014 SET @@session.immediate_server_version=80027*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 45573
#230210 15:08:45 server id 1  end_log_pos 45657 CRC32 0x5e18ab81 	Query	thread_id=8265	exec_time=0	error_code=0
SET TIMESTAMP=1676012925/*!*/;
BEGIN
/*!*/;
# at 45657
#230210 15:08:45 server id 1  end_log_pos 45761 CRC32 0x7304d3c4 	Table_map: ` + "`bytebase`" + `.` + "`migration_history`" + ` mapped to numb114
# at 45761
#230210 15:08:45 server id 1  end_log_pos 47812 CRC32 0x130a8a04 	Update_rows: table id 114 flags: STMT_END_F
### UPDATE ` + "`bytebase`" + `.` + "`migration_history`" + `
### WHERE
###   @1=12
###   @2='Me'
###   @3=1676012925
###   @4='Me'
###   @5=1676012925
###   @6='development'
###   @7='test'
###   @8=11
###   @9='UI'
###   @10='DATA'
###   @11='PENDING'
###   @12='0000.0000.0000-20230210150841'
###   @13='[test] Change data @02-10 15:06 UTC+0800 - DML(data) for database "test"'
###   @14='select 1'
###   @15='SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKFOREIGN_KEY_CHECKS=0;\n--\n-- Table structure for ` + "`tbl`" + `\n--\nCREATE TABLE ` + "`tbl`" + ` (\n  ` + "`id`" + ` int NOT NULL COMMENT \'ID\',\PRIMARY KEY (` + "`id`" + `)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\nSFOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n'
###   @16='SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKFOREIGN_KEY_CHECKS=0;\n--\n-- Table structure for ` + "`tbl`" + `\n--\nCREATE TABLE ` + "`tbl`" + ` (\n  ` + "`id`" + ` int NOT NULL COMMENT \'ID\',\PRIMARY KEY (` + "`id`" + `)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\nSFOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n'
###   @17=0
###   @18='119'
###   @19=''
### SET
###   @1=12
###   @2='Me'
###   @3=1676012925
###   @4='Me'
###   @5=1676012925
###   @6='development'
###   @7='test'
###   @8=11
###   @9='UI'
###   @10='DATA'
###   @11='DONE'
###   @12='0000.0000.0000-20230210150841'
###   @13='[test] Change data @02-10 15:06 UTC+0800 - DML(data) for database "test"'
###   @14='select 1'
###   @15='SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKFOREIGN_KEY_CHECKS=0;\n--\n-- Table structure for ` + "`tbl`" + `\n--\nCREATE TABLE ` + "`tbl`" + ` (\n  ` + "`id`" + ` int NOT NULL COMMENT \'ID\',\PRIMARY KEY (` + "`id`" + `)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\nSFOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n'
###   @16='SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKFOREIGN_KEY_CHECKS=0;\n--\n-- Table structure for ` + "`tbl`" + `\n--\nCREATE TABLE ` + "`tbl`" + ` (\n  ` + "`id`" + ` int NOT NULL COMMENT \'ID\',\PRIMARY KEY (` + "`id`" + `)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;\n\nSFOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n'
###   @17=7351000
###   @18='119'
###   @19=''
# at 47812
#230210 15:08:45 server id 1  end_log_pos 47843 CRC32 0xcafe7260 	Xid = 51758
COMMIT/*!*/;
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
DELIMITER ;
# End of log file
/*!50003 SET COMPLETION_TYPE=@OLD_COMPLETION_TYPE*/;
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=0*/;`,
			want: nil,
			err:  false,
		},
		{
			name:      "exceeds limit",
			sizeLimit: 1024,
			// This is generated by the following SQL:
			//   CREATE DATABASE binlog_test;
			//   USE binlog_test;
			//   CREATE TABLE user (id INT PRIMARY KEY, name VARCHAR(20), balance INT);
			//   INSERT INTO user VALUES (1, 'alice', 100), (2, 'bob', 100), (3, 'cindy', 100);
			//   BEGIN; UPDATE user SET balance=90 WHERE id=1; UPDATE user SET balance=110 WHERE id=2; COMMIT;
			//   DELETE FROM user WHERE id=3;
			//   UPDATE user SET balance=0;
			//   DELETE FROM user;
			stream: `# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
DELIMITER /*!*/;
# at 4
#221017 11:58:17 server id 1  end_log_pos 126 CRC32 0x05c5e116 	Start: binlog v 4, server v 8.0.30 created 221017 11:58:17
# at 126
#221017 11:58:17 server id 1  end_log_pos 157 CRC32 0x1ed79cf3 	Previous-GTIDs
# [empty]
# at 157
#221017 11:59:35 server id 1  end_log_pos 234 CRC32 0x1ac93ff6 	Anonymous_GTID	last_committed=0	sequence_number=1	rbr_only=no	original_committed_timestamp=1665979175330105	immediate_commit_timestamp=1665979175330105	transaction_length=206
# original_commit_timestamp=1665979175330105 (2022-10-17 11:59:35.330105 CST)
# immediate_commit_timestamp=1665979175330105 (2022-10-17 11:59:35.330105 CST)
/*!80001 SET @@session.original_commit_timestamp=1665979175330105*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 234
#221017 11:59:35 server id 1  end_log_pos 363 CRC32 0x88a0af23 	Query	thread_id=53771	exec_time=0	error_code=0	Xid = 327575
SET TIMESTAMP=1665979175/*!*/;
SET @@session.pseudo_thread_id=53771/*!*/;
SET @@session.foreign_key_checks=1, @@session.sql_auto_is_null=0, @@session.unique_checks=1, @@session.autocommit=1/*!*/;
SET @@session.sql_mode=1168113696/*!*/;
SET @@session.auto_increment_increment=1, @@session.auto_increment_offset=1/*!*/;
/*!\C utf8mb3 *//*!*/;
SET @@session.character_set_client=33,@@session.collation_connection=33,@@session.collation_server=255/*!*/;
SET @@session.lc_time_names=0/*!*/;
SET @@session.collation_database=DEFAULT/*!*/;
/*!80011 SET @@session.default_collation_for_utf8mb4=255*//*!*/;
/*!80016 SET @@session.default_table_encryption=0*//*!*/;
create database binlog_test
/*!*/;
# at 363
#221017 14:20:07 server id 1  end_log_pos 440 CRC32 0x0b45e716 	Anonymous_GTID	last_committed=1	sequence_number=2	rbr_only=no	original_committed_timestamp=1665987607990021	immediate_commit_timestamp=1665987607990021	transaction_length=248
# original_commit_timestamp=1665987607990021 (2022-10-17 14:20:07.990021 CST)
# immediate_commit_timestamp=1665987607990021 (2022-10-17 14:20:07.990021 CST)
/*!80001 SET @@session.original_commit_timestamp=1665987607990021*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 440
#221017 14:20:07 server id 1  end_log_pos 611 CRC32 0x7a17ec03 	Query	thread_id=53771	exec_time=0	error_code=0	Xid = 327594
use ` + "`binlog_test`" + `/*!*/;
SET TIMESTAMP=1665987607/*!*/;
/*!80013 SET @@session.sql_require_primary_key=0*//*!*/;
create table user (id int primary key, name varchar(20), balance int)
/*!*/;
# at 611
#221017 14:25:24 server id 1  end_log_pos 690 CRC32 0x9a0a39e8 	Anonymous_GTID	last_committed=2	sequence_number=3	rbr_only=yes	original_committed_timestamp=1665987924680671	immediate_commit_timestamp=1665987924680671	transaction_length=336
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1665987924680671 (2022-10-17 14:25:24.680671 CST)
# immediate_commit_timestamp=1665987924680671 (2022-10-17 14:25:24.680671 CST)
/*!80001 SET @@session.original_commit_timestamp=1665987924680671*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 690
#221017 14:25:24 server id 1  end_log_pos 772 CRC32 0x37cb53f6 	Query	thread_id=53771	exec_time=0	error_code=0
SET TIMESTAMP=1665987924/*!*/;
BEGIN
/*!*/;
# at 772
#221017 14:25:24 server id 1  end_log_pos 838 CRC32 0x449e64c7 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 838
#221017 14:25:24 server id 1  end_log_pos 916 CRC32 0x896854fc 	Write_rows: table id 259 flags: STMT_END_F
### INSERT INTO ` + "`binlog_test`.`user`" + `
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
###   @3=100
# at 916
#221017 14:25:24 server id 1  end_log_pos 947 CRC32 0xaf8e8303 	Xid = 327602
COMMIT/*!*/;
# at 947
#221017 14:26:12 server id 1  end_log_pos 1026 CRC32 0xeb36a09d 	Anonymous_GTID	last_committed=3	sequence_number=4	rbr_only=yes	original_committed_timestamp=1665987972230125	immediate_commit_timestamp=1665987972230125	transaction_length=461
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1665987972230125 (2022-10-17 14:26:12.230125 CST)
# immediate_commit_timestamp=1665987972230125 (2022-10-17 14:26:12.230125 CST)
/*!80001 SET @@session.original_commit_timestamp=1665987972230125*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 1026
#221017 14:25:53 server id 1  end_log_pos 1117 CRC32 0x5842528e 	Query	thread_id=53771	exec_time=0	error_code=0
SET TIMESTAMP=1665987953/*!*/;
BEGIN
/*!*/;
# at 1117
#221017 14:25:53 server id 1  end_log_pos 1183 CRC32 0xff6bd156 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 1183
#221017 14:25:53 server id 1  end_log_pos 1249 CRC32 0x3d8fa43e 	Update_rows: table id 259 flags: STMT_END_F
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=100
### SET
###   @1=1
###   @2='alice'
###   @3=90
# at 1249
#221017 14:26:08 server id 1  end_log_pos 1315 CRC32 0xab274b93 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 1315
#221017 14:26:08 server id 1  end_log_pos 1377 CRC32 0xd7bb3662 	Update_rows: table id 259 flags: STMT_END_F
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=100
### SET
###   @1=2
###   @2='bob'
###   @3=110
# at 1377
#221017 14:26:12 server id 1  end_log_pos 1408 CRC32 0xf2dd63fe 	Xid = 327607
COMMIT/*!*/;
# at 1408
#221017 14:31:58 server id 1  end_log_pos 1487 CRC32 0x875da2b3 	Anonymous_GTID	last_committed=4	sequence_number=5	rbr_only=yes	original_committed_timestamp=1665988318242416	immediate_commit_timestamp=1665988318242416	transaction_length=308
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1665988318242416 (2022-10-17 14:31:58.242416 CST)
# immediate_commit_timestamp=1665988318242416 (2022-10-17 14:31:58.242416 CST)
/*!80001 SET @@session.original_commit_timestamp=1665988318242416*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 1487
#221017 14:31:53 server id 1  end_log_pos 1569 CRC32 0x04ff75ee 	Query	thread_id=53771	exec_time=0	error_code=0
SET TIMESTAMP=1665988313/*!*/;
BEGIN
/*!*/;
# at 1569
#221017 14:31:53 server id 1  end_log_pos 1635 CRC32 0x5c2b9586 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 1635
#221017 14:31:53 server id 1  end_log_pos 1685 CRC32 0x5ea4b2c4 	Delete_rows: table id 259 flags: STMT_END_F
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=3
###   @2='cindy'
###   @3=100
# at 1685
#221017 14:31:58 server id 1  end_log_pos 1716 CRC32 0x55ad4a8d 	Xid = 327625
COMMIT/*!*/;
# at 1716
#221018 16:21:19 server id 1  end_log_pos 1795 CRC32 0x74cfc812 	Anonymous_GTID	last_committed=5	sequence_number=6	rbr_only=yes	original_committed_timestamp=1666081279389079	immediate_commit_timestamp=1666081279389079	transaction_length=359
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1666081279389079 (2022-10-18 16:21:19.389079 CST)
# immediate_commit_timestamp=1666081279389079 (2022-10-18 16:21:19.389079 CST)
/*!80001 SET @@session.original_commit_timestamp=1666081279389079*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 1795
#221018 16:21:19 server id 1  end_log_pos 1886 CRC32 0xd95a1592 	Query	thread_id=58599	exec_time=0	error_code=0
SET TIMESTAMP=1666081279/*!*/;
BEGIN
/*!*/;
# at 1886
#221018 16:21:19 server id 1  end_log_pos 1952 CRC32 0x4abaf53a 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 1952
#221018 16:21:19 server id 1  end_log_pos 2044 CRC32 0x9dbbb766 	Update_rows: table id 259 flags: STMT_END_F
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=90
### SET
###   @1=1
###   @2='alice'
###   @3=0
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=110
### SET
###   @1=2
###   @2='bob'
###   @3=0
# at 2044
#221018 16:21:19 server id 1  end_log_pos 2075 CRC32 0x64944ac3 	Xid = 349592
COMMIT/*!*/;
# at 2075
#221018 16:21:45 server id 1  end_log_pos 2154 CRC32 0xe3151316 	Anonymous_GTID	last_committed=6	sequence_number=7	rbr_only=yes	original_committed_timestamp=1666081305115676	immediate_commit_timestamp=1666081305115676	transaction_length=321
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1666081305115676 (2022-10-18 16:21:45.115676 CST)
# immediate_commit_timestamp=1666081305115676 (2022-10-18 16:21:45.115676 CST)
/*!80001 SET @@session.original_commit_timestamp=1666081305115676*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 2154
#221018 16:21:45 server id 1  end_log_pos 2236 CRC32 0x965db1d1 	Query	thread_id=58599	exec_time=0	error_code=0
SET TIMESTAMP=1666081305/*!*/;
BEGIN
/*!*/;
# at 2236
#221018 16:21:45 server id 1  end_log_pos 2302 CRC32 0x1340524f 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 2302
#221018 16:21:45 server id 1  end_log_pos 2365 CRC32 0xf759c90c 	Delete_rows: table id 259 flags: STMT_END_F
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=0
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=0
# at 2365
#221018 16:21:45 server id 1  end_log_pos 2396 CRC32 0x816695ae 	Xid = 349604
COMMIT/*!*/;
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
DELIMITER ;
# End of log file
/*!50003 SET COMPLETION_TYPE=@OLD_COMPLETION_TYPE*/;
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=0*/;`,
			want: []BinlogTransaction{
				{
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
###   @3=100
`,
					},
					{
						Type:   XidEventType,
						Header: "#221017 14:25:24 server id 1  end_log_pos 947 CRC32 0xaf8e8303 	Xid = 327602\n",
						Body: `COMMIT/*!*/;
`,
					},
				},
				{
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
###   @3=90
`,
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
###   @3=110
`,
					},
					{
						Type:   XidEventType,
						Header: "#221017 14:26:12 server id 1  end_log_pos 1408 CRC32 0xf2dd63fe 	Xid = 327607\n",
						Body: `COMMIT/*!*/;
`,
					},
				},
				{
					{
						Type:   QueryEventType,
						Header: "#221017 14:31:53 server id 1  end_log_pos 1569 CRC32 0x04ff75ee 	Query	thread_id=53771	exec_time=0	error_code=0\n",
						Body: `SET TIMESTAMP=1665988313/*!*/;
BEGIN
/*!*/;
`,
					},
					{
						Type:   DeleteRowsEventType,
						Header: "#221017 14:31:53 server id 1  end_log_pos 1685 CRC32 0x5ea4b2c4 	Delete_rows: table id 259 flags: STMT_END_F\n",
						Body: `### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=3
###   @2='cindy'
###   @3=100
`,
					},
					{
						Type:   XidEventType,
						Header: "#221017 14:31:58 server id 1  end_log_pos 1716 CRC32 0x55ad4a8d 	Xid = 327625\n",
						Body: `COMMIT/*!*/;
`,
					},
				},
			},
			err: true,
		},
		{
			name:      "real world",
			sizeLimit: 8 * 1024 * 1024,
			// This is generated by the following SQL:
			//   CREATE DATABASE binlog_test;
			//   USE binlog_test;
			//   CREATE TABLE user (id INT PRIMARY KEY, name VARCHAR(20), balance INT);
			//   INSERT INTO user VALUES (1, 'alice', 100), (2, 'bob', 100), (3, 'cindy', 100);
			//   BEGIN; UPDATE user SET balance=90 WHERE id=1; UPDATE user SET balance=110 WHERE id=2; COMMIT;
			//   DELETE FROM user WHERE id=3;
			//   UPDATE user SET balance=0;
			//   DELETE FROM user;
			stream: `# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
DELIMITER /*!*/;
# at 4
#221017 11:58:17 server id 1  end_log_pos 126 CRC32 0x05c5e116 	Start: binlog v 4, server v 8.0.30 created 221017 11:58:17
# at 126
#221017 11:58:17 server id 1  end_log_pos 157 CRC32 0x1ed79cf3 	Previous-GTIDs
# [empty]
# at 157
#221017 11:59:35 server id 1  end_log_pos 234 CRC32 0x1ac93ff6 	Anonymous_GTID	last_committed=0	sequence_number=1	rbr_only=no	original_committed_timestamp=1665979175330105	immediate_commit_timestamp=1665979175330105	transaction_length=206
# original_commit_timestamp=1665979175330105 (2022-10-17 11:59:35.330105 CST)
# immediate_commit_timestamp=1665979175330105 (2022-10-17 11:59:35.330105 CST)
/*!80001 SET @@session.original_commit_timestamp=1665979175330105*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 234
#221017 11:59:35 server id 1  end_log_pos 363 CRC32 0x88a0af23 	Query	thread_id=53771	exec_time=0	error_code=0	Xid = 327575
SET TIMESTAMP=1665979175/*!*/;
SET @@session.pseudo_thread_id=53771/*!*/;
SET @@session.foreign_key_checks=1, @@session.sql_auto_is_null=0, @@session.unique_checks=1, @@session.autocommit=1/*!*/;
SET @@session.sql_mode=1168113696/*!*/;
SET @@session.auto_increment_increment=1, @@session.auto_increment_offset=1/*!*/;
/*!\C utf8mb3 *//*!*/;
SET @@session.character_set_client=33,@@session.collation_connection=33,@@session.collation_server=255/*!*/;
SET @@session.lc_time_names=0/*!*/;
SET @@session.collation_database=DEFAULT/*!*/;
/*!80011 SET @@session.default_collation_for_utf8mb4=255*//*!*/;
/*!80016 SET @@session.default_table_encryption=0*//*!*/;
create database binlog_test
/*!*/;
# at 363
#221017 14:20:07 server id 1  end_log_pos 440 CRC32 0x0b45e716 	Anonymous_GTID	last_committed=1	sequence_number=2	rbr_only=no	original_committed_timestamp=1665987607990021	immediate_commit_timestamp=1665987607990021	transaction_length=248
# original_commit_timestamp=1665987607990021 (2022-10-17 14:20:07.990021 CST)
# immediate_commit_timestamp=1665987607990021 (2022-10-17 14:20:07.990021 CST)
/*!80001 SET @@session.original_commit_timestamp=1665987607990021*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 440
#221017 14:20:07 server id 1  end_log_pos 611 CRC32 0x7a17ec03 	Query	thread_id=53771	exec_time=0	error_code=0	Xid = 327594
use ` + "`binlog_test`" + `/*!*/;
SET TIMESTAMP=1665987607/*!*/;
/*!80013 SET @@session.sql_require_primary_key=0*//*!*/;
create table user (id int primary key, name varchar(20), balance int)
/*!*/;
# at 611
#221017 14:25:24 server id 1  end_log_pos 690 CRC32 0x9a0a39e8 	Anonymous_GTID	last_committed=2	sequence_number=3	rbr_only=yes	original_committed_timestamp=1665987924680671	immediate_commit_timestamp=1665987924680671	transaction_length=336
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1665987924680671 (2022-10-17 14:25:24.680671 CST)
# immediate_commit_timestamp=1665987924680671 (2022-10-17 14:25:24.680671 CST)
/*!80001 SET @@session.original_commit_timestamp=1665987924680671*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 690
#221017 14:25:24 server id 1  end_log_pos 772 CRC32 0x37cb53f6 	Query	thread_id=53771	exec_time=0	error_code=0
SET TIMESTAMP=1665987924/*!*/;
BEGIN
/*!*/;
# at 772
#221017 14:25:24 server id 1  end_log_pos 838 CRC32 0x449e64c7 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 838
#221017 14:25:24 server id 1  end_log_pos 916 CRC32 0x896854fc 	Write_rows: table id 259 flags: STMT_END_F
### INSERT INTO ` + "`binlog_test`.`user`" + `
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
###   @3=100
# at 916
#221017 14:25:24 server id 1  end_log_pos 947 CRC32 0xaf8e8303 	Xid = 327602
COMMIT/*!*/;
# at 947
#221017 14:26:12 server id 1  end_log_pos 1026 CRC32 0xeb36a09d 	Anonymous_GTID	last_committed=3	sequence_number=4	rbr_only=yes	original_committed_timestamp=1665987972230125	immediate_commit_timestamp=1665987972230125	transaction_length=461
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1665987972230125 (2022-10-17 14:26:12.230125 CST)
# immediate_commit_timestamp=1665987972230125 (2022-10-17 14:26:12.230125 CST)
/*!80001 SET @@session.original_commit_timestamp=1665987972230125*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 1026
#221017 14:25:53 server id 1  end_log_pos 1117 CRC32 0x5842528e 	Query	thread_id=53771	exec_time=0	error_code=0
SET TIMESTAMP=1665987953/*!*/;
BEGIN
/*!*/;
# at 1117
#221017 14:25:53 server id 1  end_log_pos 1183 CRC32 0xff6bd156 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 1183
#221017 14:25:53 server id 1  end_log_pos 1249 CRC32 0x3d8fa43e 	Update_rows: table id 259 flags: STMT_END_F
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=100
### SET
###   @1=1
###   @2='alice'
###   @3=90
# at 1249
#221017 14:26:08 server id 1  end_log_pos 1315 CRC32 0xab274b93 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 1315
#221017 14:26:08 server id 1  end_log_pos 1377 CRC32 0xd7bb3662 	Update_rows: table id 259 flags: STMT_END_F
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=100
### SET
###   @1=2
###   @2='bob'
###   @3=110
# at 1377
#221017 14:26:12 server id 1  end_log_pos 1408 CRC32 0xf2dd63fe 	Xid = 327607
COMMIT/*!*/;
# at 1408
#221017 14:31:58 server id 1  end_log_pos 1487 CRC32 0x875da2b3 	Anonymous_GTID	last_committed=4	sequence_number=5	rbr_only=yes	original_committed_timestamp=1665988318242416	immediate_commit_timestamp=1665988318242416	transaction_length=308
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1665988318242416 (2022-10-17 14:31:58.242416 CST)
# immediate_commit_timestamp=1665988318242416 (2022-10-17 14:31:58.242416 CST)
/*!80001 SET @@session.original_commit_timestamp=1665988318242416*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 1487
#221017 14:31:53 server id 1  end_log_pos 1569 CRC32 0x04ff75ee 	Query	thread_id=53771	exec_time=0	error_code=0
SET TIMESTAMP=1665988313/*!*/;
BEGIN
/*!*/;
# at 1569
#221017 14:31:53 server id 1  end_log_pos 1635 CRC32 0x5c2b9586 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 1635
#221017 14:31:53 server id 1  end_log_pos 1685 CRC32 0x5ea4b2c4 	Delete_rows: table id 259 flags: STMT_END_F
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=3
###   @2='cindy'
###   @3=100
# at 1685
#221017 14:31:58 server id 1  end_log_pos 1716 CRC32 0x55ad4a8d 	Xid = 327625
COMMIT/*!*/;
# at 1716
#221018 16:21:19 server id 1  end_log_pos 1795 CRC32 0x74cfc812 	Anonymous_GTID	last_committed=5	sequence_number=6	rbr_only=yes	original_committed_timestamp=1666081279389079	immediate_commit_timestamp=1666081279389079	transaction_length=359
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1666081279389079 (2022-10-18 16:21:19.389079 CST)
# immediate_commit_timestamp=1666081279389079 (2022-10-18 16:21:19.389079 CST)
/*!80001 SET @@session.original_commit_timestamp=1666081279389079*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 1795
#221018 16:21:19 server id 1  end_log_pos 1886 CRC32 0xd95a1592 	Query	thread_id=58599	exec_time=0	error_code=0
SET TIMESTAMP=1666081279/*!*/;
BEGIN
/*!*/;
# at 1886
#221018 16:21:19 server id 1  end_log_pos 1952 CRC32 0x4abaf53a 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 1952
#221018 16:21:19 server id 1  end_log_pos 2044 CRC32 0x9dbbb766 	Update_rows: table id 259 flags: STMT_END_F
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=90
### SET
###   @1=1
###   @2='alice'
###   @3=0
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=110
### SET
###   @1=2
###   @2='bob'
###   @3=0
# at 2044
#221018 16:21:19 server id 1  end_log_pos 2075 CRC32 0x64944ac3 	Xid = 349592
COMMIT/*!*/;
# at 2075
#221018 16:21:45 server id 1  end_log_pos 2154 CRC32 0xe3151316 	Anonymous_GTID	last_committed=6	sequence_number=7	rbr_only=yes	original_committed_timestamp=1666081305115676	immediate_commit_timestamp=1666081305115676	transaction_length=321
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1666081305115676 (2022-10-18 16:21:45.115676 CST)
# immediate_commit_timestamp=1666081305115676 (2022-10-18 16:21:45.115676 CST)
/*!80001 SET @@session.original_commit_timestamp=1666081305115676*//*!*/;
/*!80014 SET @@session.original_server_version=80030*//*!*/;
/*!80014 SET @@session.immediate_server_version=80030*//*!*/;
SET @@SESSION.GTID_NEXT= 'ANONYMOUS'/*!*/;
# at 2154
#221018 16:21:45 server id 1  end_log_pos 2236 CRC32 0x965db1d1 	Query	thread_id=58599	exec_time=0	error_code=0
SET TIMESTAMP=1666081305/*!*/;
BEGIN
/*!*/;
# at 2236
#221018 16:21:45 server id 1  end_log_pos 2302 CRC32 0x1340524f 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259
# at 2302
#221018 16:21:45 server id 1  end_log_pos 2365 CRC32 0xf759c90c 	Delete_rows: table id 259 flags: STMT_END_F
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=0
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=2
###   @2='bob'
###   @3=0
# at 2365
#221018 16:21:45 server id 1  end_log_pos 2396 CRC32 0x816695ae 	Xid = 349604
COMMIT/*!*/;
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
DELIMITER ;
# End of log file
/*!50003 SET COMPLETION_TYPE=@OLD_COMPLETION_TYPE*/;
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=0*/;`,
			want: []BinlogTransaction{
				{
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
###   @3=100
`,
					},
					{
						Type:   XidEventType,
						Header: "#221017 14:25:24 server id 1  end_log_pos 947 CRC32 0xaf8e8303 	Xid = 327602\n",
						Body: `COMMIT/*!*/;
`,
					},
				},
				{
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
###   @3=90
`,
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
###   @3=110
`,
					},
					{
						Type:   XidEventType,
						Header: "#221017 14:26:12 server id 1  end_log_pos 1408 CRC32 0xf2dd63fe 	Xid = 327607\n",
						Body: `COMMIT/*!*/;
`,
					},
				},
				{
					{
						Type:   QueryEventType,
						Header: "#221017 14:31:53 server id 1  end_log_pos 1569 CRC32 0x04ff75ee 	Query	thread_id=53771	exec_time=0	error_code=0\n",
						Body: `SET TIMESTAMP=1665988313/*!*/;
BEGIN
/*!*/;
`,
					},
					{
						Type:   DeleteRowsEventType,
						Header: "#221017 14:31:53 server id 1  end_log_pos 1685 CRC32 0x5ea4b2c4 	Delete_rows: table id 259 flags: STMT_END_F\n",
						Body: `### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=3
###   @2='cindy'
###   @3=100
`,
					},
					{
						Type:   XidEventType,
						Header: "#221017 14:31:58 server id 1  end_log_pos 1716 CRC32 0x55ad4a8d 	Xid = 327625\n",
						Body: `COMMIT/*!*/;
`,
					},
				},
			},
			err: false,
		},
	}

	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			a := require.New(t)
			txns, err := ParseBinlogStream(context.Background(), strings.NewReader(test.stream), "53771", test.sizeLimit)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(test.want, txns)
			}
		})
	}
}

func TestFilterBinlogTransactionsByThreadID(t *testing.T) {
	tests := []struct {
		name     string
		txn      BinlogTransaction
		threadID string
		ok       bool
		err      bool
	}{
		{
			name: "empty",
			err:  false,
		},
		{
			name: "invalid transaction",
			txn: BinlogTransaction{
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
			},
			ok:  false,
			err: true,
		},
		{
			name: "real world, ok",
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
			threadID: "53771",
			ok:       true,
			err:      false,
		},
		{
			name: "real world, thread id not match",
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
			threadID: "53771",
			ok:       false,
			err:      false,
		},
	}

	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			a := require.New(t)
			ok, err := filterBinlogTransactionsByThreadID(test.txn, test.threadID)
			a.Equal(test.ok, ok)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
			}
		})
	}
}
