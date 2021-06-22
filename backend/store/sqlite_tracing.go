// +build debug, sqlite_trace

package store

import (
	"database/sql"
	"fmt"
	"strings"

	sqlite3 "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

var (
	blackListTables = []string{"principal", "environment", "member", "project_member", "task_run"}
	blackListStmt   = []string{"SELECT"}
)

func traceCallback(info sqlite3.TraceInfo, logger *zap.Logger) int {
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
		if cleanText != "BEGIN" && cleanText != "COMMIT" && cleanText != "ROLLBACK" {
			shouldLog := true
			for _, table := range blackListTables {
				if strings.Contains(cleanText, fmt.Sprintf("FROM %s ", table)) {
					shouldLog = false
					break
				}
			}

			for _, stmt := range blackListStmt {
				if strings.Contains(cleanText, stmt) {
					shouldLog = false
					break
				}
			}

			if shouldLog {
				logger.Info(fmt.Sprintf("[trace.sql]%s%s\n",
					cleanText,
					dbErrText))
			}
		}
	} else {
		logger.Info(fmt.Sprintf("[trace.sql]%s%s\n",
			cleanText,
			dbErrText))
	}
	return 0
}

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("Failed to create logger.")
	}

	eventMask := sqlite3.TraceStmt | sqlite3.TraceClose

	sql.Register("sqlite3_tracing",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				err := conn.SetTrace(&sqlite3.TraceConfig{
					Callback:        func(info sqlite3.TraceInfo) int { return traceCallback(info, logger) },
					EventMask:       eventMask,
					WantExpandedSQL: true,
				})
				return err
			},
		})
}
