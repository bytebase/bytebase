package mysql

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	regexpDatabaseTable = regexp.MustCompile("`(.+)`\\.`(.+)`")
)

// GetRollbackSQL generates the rollback SQL for the list of binlog events in the reversed order.
func (txn BinlogTransaction) GetRollbackSQL(columnNames []string) (string, error) {
	if len(txn) == 0 {
		return "", nil
	}
	var sqlList []string
	// Generate rollback SQL for each statement of the transaction in reversed order.
	// Each statement may have multiple affected rows in a single binlog event. The order between them is irrelevant.
	for i := len(txn) - 1; i >= 0; i-- {
		e := txn[i]
		if e.Type != WriteRowsEventType && e.Type != DeleteRowsEventType && e.Type != UpdateRowsEventType {
			continue
		}
		sql, err := e.getRollbackSQL(columnNames)
		if err != nil {
			return "", err
		}
		sqlList = append(sqlList, sql)
	}
	return strings.Join(sqlList, "\n"), nil
}

func (e *BinlogEvent) getRollbackSQL(columnNames []string) (string, error) {
	body, err := e.parseDMLBody()
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse the body %q of %s event", e.Body, e.Type.String())
	}

	if len(body.oldDataList) > 0 && len(body.oldDataList[0]) != len(columnNames) {
		return "", errors.Errorf("provided %d columns, but got %d values in the old data list of the %s event", len(columnNames), len(body.oldDataList), e.Type.String())
	}
	if len(body.newDataList) > 0 && len(body.newDataList[0]) != len(columnNames) {
		return "", errors.Errorf("provided %d columns, but got %d values in the new data list of the %s event", len(columnNames), len(body.newDataList), e.Type.String())
	}

	var sqlList []string
	switch e.Type {
	case WriteRowsEventType:
		for _, row := range body.newDataList {
			var where []string
			for i := range columnNames {
				where = append(where, fmt.Sprintf("%s=%s", columnNames[i], row[i]))
			}
			sqlList = append(sqlList, fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s;", body.database, body.table, strings.Join(where, " AND ")))
		}
	case DeleteRowsEventType:
		cols := strings.Join(columnNames, ", ")
		for _, row := range body.oldDataList {
			vals := strings.Join(row, ", ")
			sqlList = append(sqlList, fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES (%s);", body.database, body.table, cols, vals))
		}
	case UpdateRowsEventType:
		for i := range body.oldDataList {
			var where []string
			var set []string
			for j, colName := range columnNames {
				where = append(where, fmt.Sprintf("%s=%s", colName, body.newDataList[i][j]))
				set = append(set, fmt.Sprintf("%s=%s", colName, body.oldDataList[i][j]))
			}
			sqlList = append(sqlList, fmt.Sprintf("UPDATE `%s`.`%s` SET %s WHERE %s;", body.database, body.table, strings.Join(set, ", "), strings.Join(where, " AND ")))
		}
	}

	return strings.Join(sqlList, "\n"), nil
}

type dmlBody struct {
	database    string
	table       string
	oldDataList [][]string
	newDataList [][]string
}

func (e *BinlogEvent) parseDMLBody() (*dmlBody, error) {
	body := strings.Split(e.Body, "\n")
	if len(body) < e.minBlockLen() {
		return nil, errors.Errorf("invalid %s event body, must be at least %d lines, but got %#v", e.Type.String(), e.minBlockLen(), body)
	}
	groups, err := e.splitDMLBody(body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split %s event body %#v", e.Type.String(), body)
	}

	var parsedBody dmlBody
	for _, block := range groups {
		if len(block) < e.minBlockLen() {
			return nil, errors.Errorf("binlog event block must be at least %d lines, but got %#v", e.minBlockLen(), block)
		}
		matches := regexpDatabaseTable.FindStringSubmatch(block[0])
		if len(matches) != 3 {
			return nil, errors.Errorf("failed to parse database and table from the %s event body block %q", e.Type.String(), strings.Join(block, "\n"))
		}
		parsedBody.database = matches[1]
		parsedBody.table = matches[2]
		dataOld, dataNew, err := e.parseDMLBlock(block)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse the %s event block", e.Type.String())
		}
		if dataOld != nil {
			parsedBody.oldDataList = append(parsedBody.oldDataList, dataOld)
		}
		if dataNew != nil {
			parsedBody.newDataList = append(parsedBody.newDataList, dataNew)
		}
	}

	if err := validateRows(parsedBody.oldDataList, e.Type.String(), e.Body); err != nil {
		return nil, err
	}
	if err := validateRows(parsedBody.newDataList, e.Type.String(), e.Body); err != nil {
		return nil, err
	}
	if len(parsedBody.oldDataList) > 0 &&
		len(parsedBody.newDataList) > 0 &&
		len(parsedBody.oldDataList) != len(parsedBody.newDataList) {
		return nil, errors.Errorf("invalid %s event block, the old data list has %d items, but the new data list has %d items", e.Type.String(), len(parsedBody.oldDataList), len(parsedBody.newDataList))
	}

	return &parsedBody, nil
}

// validateRows checks that all the rows have the same number of columns.
func validateRows(rows [][]string, eventType, body string) error {
	if len(rows) > 0 {
		cols := len(rows[0])
		for _, row := range rows[1:] {
			if len(row) != cols {
				return errors.Errorf("inconsistent value length found in the %s event body %q", eventType, body)
			}
		}
	}
	return nil
}

func (e *BinlogEvent) parseDMLBlock(block []string) (dataOld []string, dataNew []string, err error) {
	block = block[1:]
	switch e.Type {
	case DeleteRowsEventType:
		// Example block:
		// ### DELETE FROM `database`.`table`
		// ### WHERE
		// ###   @1=x
		//       ...
		where, err := parseDMLBlockSection(block, "WHERE")
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse the WHERE section of the DELETE event")
		}
		return where, nil, nil
	case UpdateRowsEventType:
		// Example block:
		// ### UPDATE `database`.`table`
		// ### WHERE
		// ###   @1=x
		//       ...
		// ### SET
		// ###   @1=y
		// 	     ...
		if len(block)%2 != 0 {
			return nil, nil, errors.Errorf("invalid UPDATE event block, WHERE clause length != SET clause length: %#v", block)
		}
		where, err := parseDMLBlockSection(block[:len(block)/2], "WHERE")
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse the WHERE section of the UPDATE event")
		}
		block = block[len(block)/2:]
		set, err := parseDMLBlockSection(block, "SET")
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse the SET section of the UPDATE event")
		}
		return where, set, nil
	case WriteRowsEventType:
		// Example block:
		// ### INSERT INTO `database`.`table`
		// ### SET
		// ###   @1=x
		//       ...
		set, err := parseDMLBlockSection(block, "SET")
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse the SET section of the INSERT event")
		}
		return nil, set, nil
	default:
		// This is to make the compiler happy.
		return nil, nil, errors.Errorf("invalid DML binlog event type %s", e.Type.String())
	}
}

func (e *BinlogEvent) splitDMLBody(lines []string) ([][]string, error) {
	var groups [][]string
	var group []string
	for _, line := range lines {
		if len(line) == 0 {
			// Skip empty lines.
			continue
		}
		if !strings.HasPrefix(line, "### ") {
			return nil, errors.Errorf("invalid event body line %q, must start with \"### \"", line)
		}
		// Starts of a new group.
		if strings.HasPrefix(line, fmt.Sprintf("### %s", e.Type.String())) {
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

func parseDMLBlockSection(lines []string, header string) ([]string, error) {
	if lines[0] != fmt.Sprintf("### %s", header) {
		return nil, errors.Errorf("failed to parse event body's first line, expecting \"### %s\", but got %q", header, lines[0])
	}
	var values []string
	for i, line := range lines[1:] {
		prefix := fmt.Sprintf("###   @%d=", i+1)
		if !strings.HasPrefix(line, prefix) {
			return nil, errors.Errorf("invalid binlog event body line %q, expecting prefix %q", line, prefix)
		}
		values = append(values, strings.TrimPrefix(line, prefix))
	}
	return values, nil
}

func (e *BinlogEvent) minBlockLen() int {
	switch e.Type {
	case DeleteRowsEventType:
		return 3
	case UpdateRowsEventType:
		return 5
	case WriteRowsEventType:
		return 3
	default:
		return 0
	}
}

func (t BinlogEventType) String() string {
	switch t {
	case DeleteRowsEventType:
		return "DELETE"
	case UpdateRowsEventType:
		return "UPDATE"
	case WriteRowsEventType:
		return "INSERT"
	case QueryEventType:
		return "QUERY"
	case XidEventType:
		return "XID"
	default:
		return "UNKNOWN"
	}
}
