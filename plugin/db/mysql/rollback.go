package mysql

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
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
	// 1. Remove the "### " prefix of each line.
	// Prefix a "\n" to make the replacement easier.
	body := strings.ReplaceAll("\n"+e.Body, "\n### ", "\n")

	// 2. Switch "DELETE FROM" and "INSERT INTO".
	// 3. Replace "WHERE" and "SET" with each other.
	// 4. Replace "@i" with the column names.
	// 5. Add a ";" at the end of each row.
	var err error
	switch e.Type {
	case WriteRowsEventType:
		body = strings.ReplaceAll(body, "\nINSERT INTO", "\nDELETE FROM")
		// Trim the "\n" prefix we added at the first step.
		body = strings.TrimPrefix(body, "\n")
		body = strings.ReplaceAll(body, "\nSET", "\nWHERE")
		body, err = replaceColumns(columnNames, body, "\nWHERE", " AND")
		if err != nil {
			return "", err
		}
		return addSemicolon(body, "\nDELETE FROM"), nil
	case DeleteRowsEventType:
		body = strings.ReplaceAll(body, "\nDELETE FROM", "\nINSERT INTO")
		// Trim the "\n" prefix we added at the first step.
		body = strings.TrimPrefix(body, "\n")
		body = strings.ReplaceAll(body, "\nWHERE", "\nSET")
		body, err = replaceColumns(columnNames, body, "\nSET", ",")
		if err != nil {
			return "", err
		}
		return addSemicolon(body, "\nINSERT INTO"), nil
	case UpdateRowsEventType:
		// Trim the "\n" prefix we added at the first step.
		body = strings.TrimPrefix(body, "\n")
		body = strings.ReplaceAll(body, "\nWHERE", "\nOLDWHERE")
		body = strings.ReplaceAll(body, "\nSET", "\nWHERE")
		body = strings.ReplaceAll(body, "\nOLDWHERE", "\nSET")
		body, err = replaceColumns(columnNames, body, "\nWHERE", " AND")
		if err != nil {
			return "", err
		}
		body, err = replaceColumns(columnNames, body, "\nSET", ",")
		if err != nil {
			return "", err
		}
		return addSemicolon(body, "\nUPDATE"), nil
	}
	return "", errors.Errorf("invalid binlog event type %s", e.Type.String())
}

func addSemicolon(sql, delimiter string) string {
	rows := strings.Split(sql, delimiter)
	for i := range rows {
		rows[i] = strings.TrimSuffix(rows[i], "\n")
	}
	return strings.Join(rows, ";"+delimiter) + ";"
}

func replaceColumns(columnNames []string, body, delimBlock, delimCol string) (string, error) {
	var rows []string
	// Split the block with "\nWHERE" or "\nSET".
	blocks := strings.Split(body, delimBlock)
	rows = append(rows, blocks[0])
	for _, block := range blocks[1:] {
		// prefixLocs is the index of all column placeholders "@i", which uses the same format as strings.FindAllStringIndex.
		var prefixLocs [][]int
		for i := range columnNames {
			prefix := fmt.Sprintf("\n  @%d=", i+1)
			colIndex := strings.Index(block, prefix)
			if colIndex == -1 {
				return "", errors.Errorf("the block has fewer values than columns, which means the schema has changed after executing the original task")
			}
			prefixLocs = append(prefixLocs, []int{colIndex, colIndex + len(prefix)})
		}
		var row []string
		for i, name := range columnNames {
			left := prefixLocs[i][1]
			var right int
			if i < len(columnNames)-1 {
				// For columns except the last one.
				right = prefixLocs[i+1][0]
			} else {
				// Write to the end of the block.
				right = len(block)
			}
			row = append(row, fmt.Sprintf("\n  `%s`=%s", name, block[left:right]))
		}
		rows = append(rows, strings.Join(row, delimCol))
	}
	return strings.Join(rows, delimBlock), nil
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
