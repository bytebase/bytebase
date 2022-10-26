package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBinlogEventDelete(t *testing.T) {
	tests := []struct {
		name            string
		binlogTextLines []string
		want            *binlogEvent
		err             bool
	}{
		{
			name:            "empty",
			binlogTextLines: nil,
			want:            nil,
			err:             true,
		},
		{
			name:            "partial 1 line 1",
			binlogTextLines: []string{"### DELETE FROM"},
			want:            nil,
			err:             true,
		},
		{
			name:            "partial 1 line 2",
			binlogTextLines: []string{"### DELETE FROM `binlog_test`.`user`"},
			want:            nil,
			err:             true,
		},
		{
			name: "partial 2 lines",
			binlogTextLines: strings.Split(`### DELETE FROM `+"`binlog_test`.`user`"+`
### WHERE`, "\n"),
			want: nil,
			err:  true,
		},
		{
			name: "min block length",
			binlogTextLines: strings.Split(`### DELETE FROM `+"`binlog_test`.`user`"+`
### WHERE
###   @1=1`, "\n"),
			want: &binlogEvent{
				Type:    DeleteRowsEvent,
				DataOld: [][]string{{"1"}},
			},
			err: false,
		},
		{
			name: "multiple rows",
			binlogTextLines: strings.Split(`### DELETE FROM `+"`binlog_test`.`user`"+`
### WHERE
###   @1=1
###   @2='alice'
###   @3=0
### DELETE FROM `+"`binlog_test`.`user`"+`
### WHERE
###   @1=2
###   @2='bob'
###   @3=0`, "\n"),
			want: &binlogEvent{
				Type: DeleteRowsEvent,
				DataOld: [][]string{
					{"1", "'alice'", "0"},
					{"2", "'bob'", "0"},
				},
			},
			err: false,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, err := parseBinlogEventDML(DeleteRowsEvent, test.binlogTextLines)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(test.want, event)
			}
		})
	}
}

func TestParseBinlogEventUpdate(t *testing.T) {
	tests := []struct {
		name            string
		binlogTextLines []string
		want            *binlogEvent
		err             bool
	}{
		{
			name:            "empty",
			binlogTextLines: nil,
			want:            nil,
			err:             true,
		},
		{
			name:            "partial 1 line 1",
			binlogTextLines: []string{"### UPDATE"},
			want:            nil,
			err:             true,
		},
		{
			name:            "partial 1 line 2",
			binlogTextLines: []string{"### UPDATE `binlog_test`.`user`"},
			want:            nil,
			err:             true,
		},
		{
			name: "partial 2 lines",
			binlogTextLines: strings.Split(`### UPDATE `+"`binlog_test`.`user`"+`
### WHERE`, "\n"),
			want: nil,
			err:  true,
		},
		{
			name: "no SET clause",
			binlogTextLines: strings.Split(`### UPDATE `+"`binlog_test`.`user`"+`
### WHERE
###   @1=1
###   @2='alice'
###   @3=90`, "\n"),
			want: nil,
			err:  true,
		},
		{
			name: "WHERE and SET clause not match",
			binlogTextLines: strings.Split(`### UPDATE `+"`binlog_test`.`user`"+`
### WHERE
###   @1=1
###   @2='alice'
###   @3=90
### SET
###   @1=1`, "\n"),
			want: nil,
			err:  true,
		},
		{
			name: "multiple rows",
			binlogTextLines: strings.Split(`### UPDATE `+"`binlog_test`.`user`"+`
### WHERE
###   @1=1
###   @2='alice'
###   @3=90
### SET
###   @1=1
###   @2='alice'
###   @3=0
### UPDATE `+"`binlog_test`.`user`"+`
### WHERE
###   @1=2
###   @2='bob'
###   @3=110
### SET
###   @1=2
###   @2='bob'
###   @3=0`, "\n"),
			want: &binlogEvent{
				Type: UpdateRowsEvent,
				DataOld: [][]string{
					{"1", "'alice'", "90"},
					{"2", "'bob'", "110"},
				},
				DataNew: [][]string{
					{"1", "'alice'", "0"},
					{"2", "'bob'", "0"},
				},
			},
			err: false,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, err := parseBinlogEventDML(UpdateRowsEvent, test.binlogTextLines)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(test.want, event)
			}
		})
	}
}

func TestParseBinlogEventInsert(t *testing.T) {
	tests := []struct {
		name            string
		binlogTextLines []string
		want            *binlogEvent
		err             bool
	}{
		{
			name:            "empty",
			binlogTextLines: nil,
			want:            nil,
			err:             true,
		},
		{
			name:            "partial 1 line 1",
			binlogTextLines: []string{"### INSERT INTO"},
			want:            nil,
			err:             true,
		},
		{
			name:            "partial 1 line 2",
			binlogTextLines: []string{"### INSERT INTO `binlog_test`.`user`"},
			want:            nil,
			err:             true,
		},
		{
			name: "partial 2 lines",
			binlogTextLines: strings.Split(`### INSERT INTO `+"`binlog_test`.`user`"+`
### SET`, "\n"),
			want: nil,
			err:  true,
		},
		{
			name: "min block length",
			binlogTextLines: strings.Split(`### INSERT INTO `+"`binlog_test`.`user`"+`
### SET
###   @1=1`, "\n"),
			want: &binlogEvent{
				Type:    WriteRowsEvent,
				DataNew: [][]string{{"1"}},
			},
			err: false,
		},
		{
			name: "multiple rows",
			binlogTextLines: strings.Split(`### INSERT INTO `+"`binlog_test`.`user`"+`
### SET
###   @1=1
###   @2='alice'
###   @3=100
### INSERT INTO `+"`binlog_test`.`user`"+`
### SET
###   @1=2
###   @2='bob'
###   @3=100
### INSERT INTO `+"`binlog_test`.`user`"+`
### SET
###   @1=3
###   @2='cindy'
###   @3=100`, "\n"),
			want: &binlogEvent{
				Type: WriteRowsEvent,
				DataNew: [][]string{
					{"1", "'alice'", "100"},
					{"2", "'bob'", "100"},
					{"3", "'cindy'", "100"},
				},
			},
			err: false,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, err := parseBinlogEventDML(WriteRowsEvent, test.binlogTextLines)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(test.want, event)
			}
		})
	}
}

func TestParseBinlogEventQuery(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   *binlogEvent
		err    bool
	}{
		{
			name:   "empty",
			header: "",
			want:   nil,
			err:    true,
		},
		{
			name:   "invalid",
			header: "#221017 11:59:35 server id 1  end_log_pos 363 CRC32 0x88a0af23 	Query	thread_id=",
			want:   nil,
			err:    true,
		},
		{
			name:   "valid",
			header: "#221017 11:59:35 server id 1  end_log_pos 363 CRC32 0x88a0af23 	Query	thread_id=53771	exec_time=0	error_code=0	Xid = 327575",
			want: &binlogEvent{
				Type:     QueryEvent,
				ThreadID: "53771",
			},
		},
	}

	a := require.New(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, err := parseBinlogEventQuery(test.header)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(test.want, event)
			}
		})
	}
}

// The body parsing are tested in the above cases for INSERT/UPDATE/DELETE/QUERY.
// So we only test for the whole process and the header parsing errors here.
func TestParseBinlogEvent(t *testing.T) {
	tests := []struct {
		name       string
		binlogText string
		want       *binlogEvent
		err        bool
	}{
		{
			name:       "empty",
			binlogText: "",
			want:       nil,
			err:        true,
		},
		{
			name:       "partial line",
			binlogText: "# at",
			want:       nil,
			err:        true,
		},
		{
			name:       "one line",
			binlogText: "# at 123",
			want:       nil,
			err:        true,
		},
		{
			name: "ignore other events",
			binlogText: `# at 1886
#221018 16:21:19 server id 1  end_log_pos 1952 CRC32 0x4abaf53a 	Table_map: ` + "`binlog_test`.`user`" + ` mapped to number 259`,
			want: nil,
			err:  false,
		},
		{
			name: "INSERT event",
			binlogText: `# at 838
#221017 14:25:24 server id 1  end_log_pos 916 CRC32 0x896854fc 	Write_rows: table id 259 flags: STMT_END_F
### INSERT INTO ` + "`binlog_test`.`user`" + `
### SET
###   @1=1
###   @2='alice'
###   @3=100`,
			want: &binlogEvent{
				Type:    WriteRowsEvent,
				DataNew: [][]string{{"1", "'alice'", "100"}},
			},
		},
		{
			name: "UPDATE event",
			binlogText: `# at 1183
#221017 14:25:53 server id 1  end_log_pos 1249 CRC32 0x3d8fa43e 	Update_rows: table id 259 flags: STMT_END_F
### UPDATE ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=1
###   @2='alice'
###   @3=100
### SET
###   @1=1
###   @2='alice'
###   @3=90`,
			want: &binlogEvent{
				Type:    UpdateRowsEvent,
				DataOld: [][]string{{"1", "'alice'", "100"}},
				DataNew: [][]string{{"1", "'alice'", "90"}},
			},
		},
		{
			name: "DELETE event",
			binlogText: `# at 1635
#221017 14:31:53 server id 1  end_log_pos 1685 CRC32 0x5ea4b2c4 	Delete_rows: table id 259 flags: STMT_END_F
### DELETE FROM ` + "`binlog_test`.`user`" + `
### WHERE
###   @1=3
###   @2='cindy'
###   @3=100`,
			want: &binlogEvent{
				Type:    DeleteRowsEvent,
				DataOld: [][]string{{"3", "'cindy'", "100"}},
			},
		},
		{
			name: "QUERY event",
			binlogText: `# at 234
#221017 11:59:35 server id 1  end_log_pos 363 CRC32 0x88a0af23 	Query	thread_id=53771	exec_time=0	error_code=0	Xid = 327575`,
			want: &binlogEvent{
				Type:     QueryEvent,
				ThreadID: "53771",
			},
		},
	}

	a := require.New(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, err := parseBinlogEvent(test.binlogText)
			if test.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(test.want, event)
			}
		})
	}
}
