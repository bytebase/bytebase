package mysql

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type binlogEventType int

const (
	// DeleteRowsEvent is the binlog event for DELETE.
	DeleteRowsEvent binlogEventType = iota
	// UpdateRowsEvent is the binlog event for UPDATE.
	UpdateRowsEvent
	// WriteRowsEvent is the binlog event for INSERT.
	WriteRowsEvent
	// QueryEvent is the binlog event for QUERY.
	// The thread ID is parsed from QUERY events.
	QueryEvent
)

var (
	regexpDeleteRow = regexp.MustCompile("DELETE FROM `(.+)`\\.`(.+)`")
	regexpUpdateRow = regexp.MustCompile("UPDATE `(.+)`\\.`(.+)`")
	regexpWriteRow  = regexp.MustCompile("INSERT INTO `(.+)`\\.`(.+)`")
	regexpThreadID  = regexp.MustCompile(`thread_id=(\d+)`)
)

func (t binlogEventType) String() string {
	switch t {
	case DeleteRowsEvent:
		return "DELETE"
	case UpdateRowsEvent:
		return "UPDATE"
	case WriteRowsEvent:
		return "INSERT"
	default:
		return "UNKNOWN"
	}
}

func (t binlogEventType) Regexp() *regexp.Regexp {
	switch t {
	case DeleteRowsEvent:
		return regexpDeleteRow
	case UpdateRowsEvent:
		return regexpUpdateRow
	case WriteRowsEvent:
		return regexpWriteRow
	default:
		return nil
	}
}

func (t binlogEventType) MinBlockLen() int {
	switch t {
	case DeleteRowsEvent:
		return 3
	case UpdateRowsEvent:
		return 5
	case WriteRowsEvent:
		return 3
	default:
		return -1
	}
}

func (t binlogEventType) ParseDMLPayload(block []string) (beforeValue []string, afterValue []string, err error) {
	block = block[1:]
	switch t {
	case DeleteRowsEvent:
		// Example block:
		// ### DELETE FROM `database`.`table`
		// ### WHERE
		// ###   @1=x
		//       ...
		where, err := parseBinlogEventPayloadBlock(block, "WHERE")
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse the WHERE payload section of the DELETE event")
		}
		return where, nil, nil
	case UpdateRowsEvent:
		// Example block:
		// ### UPDATE `database`.`table`
		// ### WHERE
		// ###   @1=x
		//       ...
		// ### SET
		// ###   @1=y
		// 	     ...
		if len(block)%2 != 0 {
			return nil, nil, errors.Errorf("invalid UPDATE event block, WHERE clause length != SET clause length: %q", strings.Join(block, "\n"))
		}
		where, err := parseBinlogEventPayloadBlock(block[:len(block)/2], "WHERE")
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse the WHERE payload section of the UPDATE event")
		}
		block = block[len(block)/2:]
		set, err := parseBinlogEventPayloadBlock(block, "SET")
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse the SET payload section of the UPDATE event")
		}
		return where, set, nil
	case WriteRowsEvent:
		// Example block:
		// ## INSERT INTO `database`.`table`
		// ### SET
		// ###   @1=x
		//       ...
		set, err := parseBinlogEventPayloadBlock(block, "SET")
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse the SET payload section of the INSERT event")
		}
		return nil, set, nil
	default:
		return nil, nil, errors.Errorf("invalid DML binlog event type %s", t.String())
	}
}

type binlogEvent struct {
	Type       binlogEventType
	Database   string
	Table      string
	DataBefore [][]string
	DataAfter  [][]string
	ThreadID   string
}

func parseBinlogEvent(binlogText string) (*binlogEvent, error) {
	lines := strings.Split(binlogText, "\n")
	if len(lines) < 2 {
		return nil, errors.Errorf("invalid mysqlbinlog dump string: must be at least 2 lines")
	}
	if !strings.HasPrefix(lines[0], "# at") {
		return nil, errors.Errorf("invalid mysqlbinlog dump string: must starts with \"# at\"")
	}

	// The second line must contain the event header information.
	// E.g., "#221017 14:25:24 server id 1  end_log_pos 916 CRC32 0x896854fc 	Write_rows: table id 259 flags: STMT_END_F"
	header := lines[1]

	var rowEvent *binlogEvent
	var err error
	body := lines[2:]
	if strings.Contains(header, "Delete_rows") {
		rowEvent, err = parseBinlogEventDML(DeleteRowsEvent, body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse DELETE event")
		}
	} else if strings.Contains(header, "Update_rows") {
		rowEvent, err = parseBinlogEventDML(UpdateRowsEvent, body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse UPDATE event")
		}
	} else if strings.Contains(header, "Write_rows") {
		rowEvent, err = parseBinlogEventDML(WriteRowsEvent, body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse INSERT event")
		}
	} else if strings.Contains(header, "Query") {
		rowEvent, err = parseBinlogEventQuery(header)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse QUERY event")
		}
	} else {
		// The binlog event type is none of DELETE/UPDATE/INSERT. Skip for now.
		rowEvent = nil
	}

	return rowEvent, nil
}

func parseBinlogEventQuery(header string) (*binlogEvent, error) {
	matches := regexpThreadID.FindStringSubmatch(header)
	if len(matches) != 2 {
		return nil, errors.Errorf("failed to parse thread ID from the QUERY event header: %q", header)
	}
	return &binlogEvent{
		Type:     QueryEvent,
		ThreadID: matches[1],
	}, nil
}

func parseBinlogEventDML(eventType binlogEventType, body []string) (*binlogEvent, error) {
	if len(body) < eventType.MinBlockLen() {
		return nil, errors.Errorf("invalid %s event body, must be at least %d lines", eventType.String(), eventType.MinBlockLen())
	}
	groups, err := splitBinlogEventBody(body, eventType.String())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split %s event body", eventType.String())
	}

	rowEvent := &binlogEvent{
		Type: eventType,
	}
	re := eventType.Regexp()
	if re == nil {
		// Should not reach here.
		return nil, errors.Errorf("invalid DML binlog event type %s", eventType.String())
	}
	for i, block := range groups {
		if len(block) < eventType.MinBlockLen() {
			return nil, errors.Errorf("binlog event payload must be at least %d lines, bot got %q", eventType.MinBlockLen(), strings.Join(block, "\n"))
		}
		if i == 0 {
			matches := re.FindStringSubmatch(block[0])
			if len(matches) != 3 {
				return nil, errors.Wrapf(err, "failed to parse database and table names from the binlog event %q with regexp %s", block[0], re.String())
			}
			rowEvent.Database = matches[1]
			rowEvent.Table = matches[2]
		}
		beforeValue, afterValue, err := eventType.ParseDMLPayload(block)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse the DML binlog event payload")
		}
		if beforeValue != nil {
			rowEvent.DataBefore = append(rowEvent.DataBefore, beforeValue)
		}
		if afterValue != nil {
			rowEvent.DataAfter = append(rowEvent.DataAfter, afterValue)
		}
	}

	return rowEvent, nil
}

func parseBinlogEventPayloadBlock(lines []string, header string) ([]string, error) {
	if lines[0] != fmt.Sprintf("### %s", header) {
		return nil, errors.Errorf("failed to parse event payload head line, expecting \"### %s\", but got %q", header, lines[0])
	}
	var values []string
	for i, line := range lines[1:] {
		prefix := fmt.Sprintf("###   @%d=", i+1)
		if !strings.HasPrefix(line, prefix) {
			return nil, errors.Errorf("invalid binlog event payload line %q, expecting prefix %q", line, prefix)
		}
		values = append(values, strings.TrimPrefix(line, prefix))
	}
	return values, nil
}

func splitBinlogEventBody(lines []string, prefix string) ([][]string, error) {
	var groups [][]string
	var group []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "### ") {
			return nil, errors.Errorf("invalid event payload line %q, must starts with \"### \"", line)
		}
		// Starts of a new group.
		if strings.HasPrefix(line, fmt.Sprintf("### %s", prefix)) {
			if len(group) > 0 {
				groups = append(groups, group)
				group = nil
			}
		}
		group = append(group, line)
	}
	// The last group.
	groups = append(groups, group)
	return groups, nil
}
