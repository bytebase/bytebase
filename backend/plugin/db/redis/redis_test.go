package redis

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestSplitCommands(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantArgs  [][]any
		wantLines []string
	}{
		{
			name:      "one command per line",
			statement: "GET a\nGET b",
			wantArgs:  [][]any{{"GET", "a"}, {"GET", "b"}},
			wantLines: []string{"GET a", "GET b"},
		},
		{
			name:      "a blank line does not shift the commands after it",
			statement: "GET a\n\nGET b\n   \nGET c",
			wantArgs:  [][]any{{"GET", "a"}, {"GET", "b"}, {"GET", "c"}},
			wantLines: []string{"GET a", "GET b", "GET c"},
		},
		{
			name:      "quoted argument",
			statement: `SET a "two words"`,
			wantArgs:  [][]any{{"SET", "a", "two words"}},
			wantLines: []string{`SET a "two words"`},
		},
		{
			name:      "no commands",
			statement: "\n  \n",
			wantArgs:  nil,
			wantLines: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args, lines, err := splitCommands(tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantArgs, args)
			require.Equal(t, tc.wantLines, lines)
			require.Len(t, lines, len(args))
		})
	}
}

func TestSplitCommandsRejectsAnUnclosedQuote(t *testing.T) {
	_, _, err := splitCommands(`SET a "unclosed`)
	require.Error(t, err)
}

// TestQueryConnOnAllNodes covers the ALL_NODES query option, whose per-shard
// replies used to be collected by a goroutine the caller never waited for, so
// the results were read before any of them had been recorded and every command
// reported no rows.
func TestQueryConnOnAllNodes(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.SharedRedisClusterContainer(t)

	d, err := newDriver().Open(ctx, storepb.Engine_REDIS, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:      storepb.DataSourceType_ADMIN,
			Host:      container.GetHost(),
			Port:      container.GetPort(),
			RedisType: storepb.DataSource_CLUSTER,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, d.Close(ctx)) })

	_, err = d.Execute(ctx, "SET allnodes:key allnodes-value", db.ExecuteOptions{})
	require.NoError(t, err)

	results, err := d.QueryConn(ctx, nil, "GET allnodes:key\nDBSIZE", db.QueryContext{
		MaximumSQLResultSize: 1 << 20,
		Option:               &v1pb.QueryOption{RedisRunCommandsOn: v1pb.QueryOption_ALL_NODES},
	})
	require.NoError(t, err)
	require.Len(t, results, 2)

	for i, want := range []string{"allnodes-value", "1"} {
		require.Len(t, results[i].Rows, 1, "statement %q returned no row", results[i].Statement)
		require.Equal(t, want, results[i].Rows[0].Values[1].GetStringValue())
		require.Equal(t, int64(1), results[i].RowsCount)
	}
}
