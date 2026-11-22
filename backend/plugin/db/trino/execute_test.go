package trino

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestExecuteLogsTheCommandRangeWithItsSemicolon pins the adapter's log bridge:
// the contract ends a command at its last token, while the task-run log's range
// covers the terminating semicolon as the splitter Bytebase used to read
// statements with did.
func TestExecuteLogsTheCommandRangeWithItsSemicolon(t *testing.T) {
	ctx := context.Background()
	c := testcontainer.SharedTrinoContainer(t)
	driver := openTestDriver(t, &storepb.DataSource{
		Host:     c.GetHost(),
		Port:     c.GetPort(),
		Username: c.GetUsername(),
		Database: "memory",
	})

	script := "CREATE TABLE memory.default.cutover_log_t (id BIGINT);\nINSERT INTO memory.default.cutover_log_t VALUES (1);"
	var logs []*storepb.TaskRunLog
	affected, err := driver.Execute(ctx, script, db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	var ranges []string
	for _, log := range logs {
		if execute := log.GetCommandExecute(); execute != nil {
			require.NotNil(t, execute.Range)
			ranges = append(ranges, script[execute.Range.Start:execute.Range.End])
			require.Equal(t, ";", script[execute.Range.End-1:execute.Range.End])
		}
	}
	require.Equal(t, []string{
		"CREATE TABLE memory.default.cutover_log_t (id BIGINT);",
		"INSERT INTO memory.default.cutover_log_t VALUES (1);",
	}, ranges)
}
