package redis

import (
	"context"
	"testing"

	"github.com/pkg/errors"
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

// serverError stands in for an error reply from the server, which go-redis
// marks with the RedisError method.
type serverError string

func (e serverError) Error() string { return string(e) }
func (serverError) RedisError()     {}

func TestDatabasesFromConfigReply(t *testing.T) {
	tests := []struct {
		name         string
		val          map[string]string
		err          error
		want         []int
		wantFailopen bool
		wantErr      bool
	}{
		{
			name: "configured count",
			val:  map[string]string{"databases": "3"},
			want: []int{0, 1, 2},
		},
		{
			name:         "reply without the databases setting",
			val:          map[string]string{},
			wantFailopen: true,
		},
		{
			name:         "command removed by the vendor",
			err:          serverError("ERR unknown command 'CONFIG', with args beginning with: 'GET' 'databases' "),
			wantFailopen: true,
		},
		{
			name:         "user without permission to run CONFIG",
			err:          serverError("NOPERM User limited has no permissions to run the 'config|get' command"),
			wantFailopen: true,
		},
		{
			name:    "connection error",
			err:     errors.New("dial tcp 127.0.0.1:6379: connect: connection refused"),
			wantErr: true,
		},
		{
			name:    "count that is not a number",
			val:     map[string]string{"databases": "many"},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, failopen, err := databasesFromConfigReply(tc.val, tc.err)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.wantFailopen, failopen)
		})
	}
}

func TestSyncInstanceWithoutConfigPermission(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.SharedRedisClusterContainer(t)
	open := func(username, password string) db.Driver {
		d, err := newDriver().Open(ctx, storepb.Engine_REDIS, db.ConnectionConfig{
			DataSource: &storepb.DataSource{
				Type:      storepb.DataSourceType_ADMIN,
				Host:      container.GetHost(),
				Port:      container.GetPort(),
				Username:  username,
				RedisType: storepb.DataSource_CLUSTER,
			},
			Password: password,
		})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, d.Close(ctx)) })
		return d
	}

	_, err := open("", "").Execute(ctx, "ACL SETUSER noconfig on >noconfig-pw ~* +@all -config", db.ExecuteOptions{})
	require.NoError(t, err)

	limited := open("noconfig", "noconfig-pw")
	_, err = limited.Execute(ctx, "CONFIG GET databases", db.ExecuteOptions{})
	require.ErrorContains(t, err, "NOPERM")

	instance, err := limited.SyncInstance(ctx)
	require.NoError(t, err)
	require.Len(t, instance.Databases, 1)
	require.Equal(t, "0", instance.Databases[0].Name)
}
