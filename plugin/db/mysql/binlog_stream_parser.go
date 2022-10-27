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
	// XidEventType is the binlog event for Xid.
	// It is the last event of a transaction.
	XidEventType
)

// BinlogEvent contains the raw string of a single binlog event from the mysqlbinlog output stream.
type BinlogEvent struct {
	Type   BinlogEventType
	Header string
	Body   string
}

// BinlogTransaction is a list of events, starting with Query (BEGIN) and ending with Xid (COMMIT).
type BinlogTransaction []BinlogEvent

// ParseBinlogStream splits the mysqlbinlog output stream to a list of transactions.
func ParseBinlogStream(stream io.Reader) ([]BinlogTransaction, error) {
	reader := bufio.NewReader(stream)
	// prevLineType := unknownLineType
	var event BinlogEvent
	var txns []BinlogTransaction
	seenEvent := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, errors.Wrap(err, "failed to read line from the stream")
		}

		switch {
		case len(line) == 0 && err == io.EOF:
			// The last line is empty. Skip the state machine.
		case !seenEvent && !strings.HasPrefix(line, "# at "):
			// Skip the first non-binlog-event lines output of mysqlbinlog.
			continue
		case strings.HasPrefix(line, "# at "):
			seenEvent = true
		case strings.Contains(line, "server id"):
			// Parse the header line.
			// Examples:
			// - Query:       #221020 15:45:58 server id 1  end_log_pos 2828 CRC32 0x5445bc77 	Query	thread_id=62592	exec_time=0	error_code=0
			// - Write_rows:  #221017 14:25:24 server id 1  end_log_pos 1916 CRC32 0x896854fc 	Write_rows: table id 259 flags: STMT_END_F
			// - Update_rows: #221018 16:21:19 server id 1  end_log_pos 2044 CRC32 0x9dbbb766 	Update_rows: table id 259 flags: STMT_END_F
			// - Delete_rows: #221017 14:31:53 server id 1  end_log_pos 1685 CRC32 0x5ea4b2c4 	Delete_rows: table id 259 flags: STMT_END_F
			// - Xid:         #221026 15:35:51 server id 1  end_log_pos 1435 CRC32 0x3be8594f 	Xid = 46
			event.Type = getEventType(line)
			event.Header = line
			continue
		default:
			// Accumulate the body.
			event.Body += line
			continue
		}

		if event.Type != UnknownEventType {
			txns = appendBinlogEvent(txns, event)
		}
		event = BinlogEvent{}
		if err == io.EOF {
			break
		}
	}

	return txns, nil
}

func appendBinlogEvent(txns []BinlogTransaction, event BinlogEvent) []BinlogTransaction {
	if len(txns) == 0 {
		txns = append(txns, BinlogTransaction{event})
		return txns
	}

	lastTxn := txns[len(txns)-1]
	if len(lastTxn) == 1 && lastTxn[0].Type == QueryEventType && event.Type == QueryEventType {
		// A Query event without a corresponding Xid event is not a start of a transaction.
		// We should replace the existing Query event with the new one.
		txns[len(txns)-1] = BinlogTransaction{event}
	} else if len(lastTxn) > 1 && lastTxn[len(lastTxn)-1].Type == XidEventType {
		// The previous transaction ends with an Xid event, which means it's a complete transaction.
		// We should append a new transaction.
		txns = append(txns, BinlogTransaction{event})
	} else {
		// The event is a DML event. Append it to the last transaction.
		lastTxn = append(lastTxn, event)
		txns[len(txns)-1] = lastTxn
	}

	return txns
}

func getEventType(header string) BinlogEventType {
	if strings.Contains(header, "Query") {
		return QueryEventType
	} else if strings.Contains(header, "Xid") {
		return XidEventType
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
