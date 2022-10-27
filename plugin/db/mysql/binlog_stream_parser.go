package mysql

import (
	"bufio"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// BinlogEventType is the enumeration of binlog event types.
type BinlogEventType int

const (
	// UnknownEventType represents other types of event that are ignored.
	UnknownEventType BinlogEventType = iota
	// WriteRowsEventType is the binlog event for INSERT.
	WriteRowsEventType
	// UpdateRowsEventType is the binlog event for UPDATE.
	UpdateRowsEventType
	// DeleteRowsEventType is the binlog event for DELETE.
	DeleteRowsEventType
	// QueryEventType is the binlog event for QUERY.
	// The thread ID is parsed from QUERY events.
	QueryEventType
)

// BinlogEvent contains the raw string of a single binlog event from the mysqlbinlog output stream.
type BinlogEvent struct {
	Type   BinlogEventType
	Header string
	Body   string
}

// BinlogTransaction is a list of events, starting with Query (BEGIN).
type BinlogTransaction []BinlogEvent

// ParseBinlogStream splits the mysqlbinlog output stream to a list of transactions.
func ParseBinlogStream(stream io.Reader) ([]BinlogTransaction, error) {
	reader := bufio.NewReader(stream)
	prevLineType := unknownLineType
	eventType := UnknownEventType
	var eventHeader string
	var bodyBuf strings.Builder
	var txns []BinlogTransaction
	var txn BinlogTransaction
	done := false
	for !done {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			done = true
		} else if err != nil {
			return nil, errors.Wrap(err, "failed to read line from the stream")
		}

		// Start of a new binlog event.
		if strings.HasPrefix(line, "# at ") {
			if eventType != UnknownEventType {
				txn = append(txn, BinlogEvent{
					Type:   eventType,
					Header: eventHeader,
					Body:   bodyBuf.String(),
				})
			}
			bodyBuf.Reset()
			prevLineType = posLineType
			continue
		}

		// Parse the header line.
		// Examples:
		// - Query:       #221020 15:45:58 server id 1  end_log_pos 2828 CRC32 0x5445bc77 	Query	thread_id=62592	exec_time=0	error_code=0
		// - Write_rows:  #221017 14:25:24 server id 1  end_log_pos 1916 CRC32 0x896854fc 	Write_rows: table id 259 flags: STMT_END_F
		// - Update_rows: #221018 16:21:19 server id 1  end_log_pos 2044 CRC32 0x9dbbb766 	Update_rows: table id 259 flags: STMT_END_F
		// - Delete_rows: #221017 14:31:53 server id 1  end_log_pos 1685 CRC32 0x5ea4b2c4 	Delete_rows: table id 259 flags: STMT_END_F
		if prevLineType == posLineType {
			eventType = getEventType(line)
			// Start of a new transaction.
			if eventType == QueryEventType {
				if len(txn) > 0 {
					txns = append(txns, txn)
					txn = nil
				}
			}
			eventHeader = line
			prevLineType = headerLineType
			continue
		}

		// Accumulate the body.
		if prevLineType == headerLineType || prevLineType == bodyLineType {
			if _, err := bodyBuf.WriteString(line); err != nil {
				return nil, errors.Wrapf(err, "failed to write line %q to the bodyBuf", line)
			}
			prevLineType = bodyLineType
			continue
		}
	}

	// Deal with the last binlog event and transaction.
	if eventType != UnknownEventType {
		txn = append(txn, BinlogEvent{
			Type:   eventType,
			Header: eventHeader,
			Body:   bodyBuf.String(),
		})
	}
	if len(txn) > 0 {
		txns = append(txns, txn)
	}

	return txns, nil
}

// binlogStreamLineType represents different line types in the process of parsing the binlog text stream.
type binlogStreamLineType int

const (
	// unknownLineType is other line types we ignore.
	// It's always at the beginning of a binlog file.
	unknownLineType binlogStreamLineType = iota
	// posLineType is the line containing "# at xxx".
	// It is always the first line of an event.
	posLineType
	// headerLineType contains the metadata of the event.
	// Example: #221026 15:35:51 server id 1  end_log_pos 311 CRC32 0x8d4b5a5e 	Query	thread_id=10	exec_time=0	error_code=0
	headerLineType
	// bodyLineType contains the body of the event.
	// Query events contain valid SQL in the body, such as "BEGIN".
	// INSERT/UPDATE/DELETE events contain the old (WHERE) and new (SET) data of the data change.
	bodyLineType
)

func getEventType(header string) BinlogEventType {
	if strings.Contains(header, "Query") {
		return QueryEventType
	} else if strings.Contains(header, "Write_rows") {
		return WriteRowsEventType
	} else if strings.Contains(header, "Update_rows") {
		return UpdateRowsEventType
	} else if strings.Contains(header, "Delete_rows") {
		return DeleteRowsEventType
	} else {
		// Ignore other events.
		return UnknownEventType
	}
}
