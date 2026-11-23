package trino

import (
	"context"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type fakeExecutorDriver struct {
	gotScript string
	gotOpts   omnidriver.ExecuteOptions
	commands  []omnidriver.Command
	affected  []int64
}

func (*fakeExecutorDriver) Ping(context.Context) error { return nil }
func (*fakeExecutorDriver) Close() error               { return nil }

func (f *fakeExecutorDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.gotScript = script
	f.gotOpts = opts
	var total int64
	for i, cmd := range f.commands {
		if opts.Log != nil {
			opts.Log.CommandExecute(cmd)
		}
		rows := f.affected[i]
		total += rows
		if opts.Log != nil {
			opts.Log.CommandResponse(rows, []int64{rows}, nil)
		}
	}
	return total, nil
}

// TestExecuteLogsTheCommandRangeWithItsSemicolon pins the adapter's log bridge:
// omni ends a command at its last token, while the task-run log's range covers
// the terminating semicolon when present.
func TestExecuteLogsTheCommandRangeWithItsSemicolon(t *testing.T) {
	t.Parallel()

	const (
		first  = "CREATE TABLE memory.default.t (id BIGINT)"
		second = "INSERT INTO memory.default.t VALUES (1)"
		third  = "SELECT 1"
	)
	script := first + ";\n" + second + ";\n" + third
	secondStart := len(first) + 2
	thirdStart := secondStart + len(second) + 2

	fake := &fakeExecutorDriver{
		commands: []omnidriver.Command{
			{Text: first, ByteStart: 0, ByteEnd: len(first)},
			{Text: second, ByteStart: secondStart, ByteEnd: secondStart + len(second)},
			{Text: third, ByteStart: thirdStart, ByteEnd: thirdStart + len(third)},
		},
		affected: []int64{0, 1, 0},
	}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

	var logs []*storepb.TaskRunLog
	affected, err := driver.Execute(context.Background(), script, db.ExecuteOptions{
		MaximumRetries: 2,
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
	require.Equal(t, script, fake.gotScript)
	require.Empty(t, fake.gotOpts.Transaction)
	require.Empty(t, fake.gotOpts.Isolation)
	require.Equal(t, 2, fake.gotOpts.MaxRetries)

	var ranges []string
	for _, log := range logs {
		if execute := log.GetCommandExecute(); execute != nil {
			require.NotNil(t, execute.Range)
			ranges = append(ranges, script[execute.Range.Start:execute.Range.End])
		}
	}
	require.Equal(t, []string{
		first + ";",
		second + ";",
		third,
	}, ranges)
}
