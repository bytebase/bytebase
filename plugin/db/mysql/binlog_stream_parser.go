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
	prevLineType := otherLine
	eventType := UnknownEventType
	var eventHeader string
	var bodyBuf strings.Builder
	var txns []BinlogTransaction
	var txn BinlogTransaction
	done := false
	for !done {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				done = true
			} else {
				return nil, errors.Wrap(err, "failed to read line from the stream")
			}
		}

		// Start of a new binlog event.
		if strings.HasPrefix(line, "# at ") {
			if eventType != UnknownEventType {
				event := BinlogEvent{
					Type:   eventType,
					Header: strings.TrimSuffix(eventHeader, "\n"),
					Body:   strings.TrimSuffix(bodyBuf.String(), "\n"),
				}
				txn = append(txn, event)
			}
			bodyBuf.Reset()
			prevLineType = posLine
			continue
		}

		// Parse the header line.
		// Examples:
		// - Query:       #221020 15:45:58 server id 1  end_log_pos 2828 CRC32 0x5445bc77 	Query	thread_id=62592	exec_time=0	error_code=0
		// - Write_rows:  #221017 14:25:24 server id 1  end_log_pos 1916 CRC32 0x896854fc 	Write_rows: table id 259 flags: STMT_END_F
		// - Update_rows: #221018 16:21:19 server id 1  end_log_pos 2044 CRC32 0x9dbbb766 	Update_rows: table id 259 flags: STMT_END_F
		// - Delete_rows: #221017 14:31:53 server id 1  end_log_pos 1685 CRC32 0x5ea4b2c4 	Delete_rows: table id 259 flags: STMT_END_F
		if prevLineType == posLine {
			eventType = getEventType(line)
			// Start of a new transaction.
			if eventType == QueryEventType {
				if len(txn) > 0 {
					txns = append(txns, txn)
					txn = nil
				}
			}
			eventHeader = line
			prevLineType = headerLine
			continue
		}

		// Accumulate the body.
		if prevLineType == headerLine || prevLineType == bodyLine {
			if _, err := bodyBuf.WriteString(line); err != nil {
				return nil, errors.Wrapf(err, "failed to write line %q to the bodyBuf", line)
			}
			prevLineType = bodyLine
			continue
		}
	}

	// Deal with the last binlog event and transaction.
	if eventType != UnknownEventType {
		event := BinlogEvent{
			Type:   eventType,
			Header: strings.TrimSuffix(eventHeader, "\n"),
			Body:   strings.TrimSuffix(bodyBuf.String(), "\n"),
		}
		txn = append(txn, event)
	}
	if len(txn) > 0 {
		txns = append(txns, txn)
	}

	return txns, nil
}

type binlogStreamLineType int

const (
	posLine binlogStreamLineType = iota
	headerLine
	bodyLine
	otherLine
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
