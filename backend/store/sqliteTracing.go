// +build debug, sqlite_trace

package store

import (
	"database/sql"
	"fmt"
	"strings"

	sqlite3 "github.com/mattn/go-sqlite3"
)

var (
	whitelistTables = []string{"project", "environment", "instance", "issue"}
)

func traceCallback(info sqlite3.TraceInfo) int {
	// Not very readable but may be useful; uncomment next line in case of doubt:
	//fmt.Printf("Trace: %#v\n", info)

	var dbErrText string
	if info.DBError.Code != 0 || info.DBError.ExtendedCode != 0 {
		dbErrText = fmt.Sprintf("; DB error: %#v", info.DBError)
	} else {
		dbErrText = ""
	}

	expandedText := info.StmtOrTrigger
	if info.ExpandedSQL != "" {
		expandedText = info.ExpandedSQL
	}

	// Make sql on a single line and remove redundant whitespaces.
	cleanText := strings.Join(strings.Fields(strings.TrimSpace(strings.Replace(expandedText, "\n", " ", -1))), " ")

	if dbErrText == "" {

		for _, table := range whitelistTables {
			if strings.Contains(cleanText, fmt.Sprintf("FROM %s ", table)) {
				fmt.Printf("%s%s\n",
					cleanText,
					dbErrText)
			}
		}
	} else {
		fmt.Printf("%s%s\n",
			cleanText,
			dbErrText)
	}
	return 0
}

func init() {
	eventMask := sqlite3.TraceStmt | sqlite3.TraceClose

	sql.Register("sqlite3_tracing",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				err := conn.SetTrace(&sqlite3.TraceConfig{
					Callback:        traceCallback,
					EventMask:       eventMask,
					WantExpandedSQL: true,
				})
				return err
			},
		})
}
