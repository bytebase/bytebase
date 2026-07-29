package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestGetMongoDBConnectionURL(t *testing.T) {
	tests := []struct {
		connConfig db.ConnectionConfig
		want       string
	}{
		{
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "localhost",
					Port:     "27017",
					Username: "",
					Password: "",
				},
			},
			want: "mongodb://localhost:27017/?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:             "localhost",
					Port:             "27017",
					Username:         "",
					Password:         "",
					DirectConnection: true,
				},
				Password: "",
			},
			want: "mongodb://localhost:27017/?appName=bytebase&authSource=admin&directConnection=true",
		},
		{
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "localhost",
					Port:     "27017",
					Username: "",
					Password: "",
				},
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "sampleDB",
				},
				Password: "",
			},
			want: "mongodb://localhost:27017/sampleDB?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "cluster0.sample.mongodb.net",
					Port:     "",
					Username: "bytebase",
					Password: "passwd",
					Srv:      true,
				},
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "sampleDB",
				},
				Password: "passwd",
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "cluster0.sample.mongodb.net",
					Port:     "",
					Username: "bytebase",
					Password: "passwd",
					Srv:      true,
					ExtraConnectionParameters: map[string]string{
						"readPreference":     "secondary",
						"readPreferenceTags": "dc:ny",
					},
				},
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "sampleDB",
				},
				Password: "passwd",
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?appName=bytebase&authSource=admin&readPreference=secondary&readPreferenceTags=dc%3Any",
		},
		{
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                   "cluster0.sample.mongodb.net",
					Port:                   "",
					Username:               "bytebase",
					Password:               "passwd",
					AuthenticationDatabase: "admin",
					Srv:                    true,
				},
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "sampleDB",
				},
				Password: "passwd",
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?appName=bytebase&authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "node1.cluster0.sample.mongodb.net",
					Port:     "27017",
					Username: "bytebase",
					Password: "passwd",
					AdditionalAddresses: []*storepb.DataSource_Address{
						{Host: "node2.cluster0.sample.mongodb.net", Port: "27017"},
						{Host: "node3.cluster0.sample.mongodb.net", Port: "27017"},
					},
					ReplicaSet:             "rs0",
					AuthenticationDatabase: "admin",
				},
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "sampleDB",
				},
				Password: "passwd",
			},
			want: "mongodb://bytebase:passwd@node1.cluster0.sample.mongodb.net:27017,node2.cluster0.sample.mongodb.net:27017,node3.cluster0.sample.mongodb.net:27017/sampleDB?appName=bytebase&authSource=admin&replicaSet=rs0",
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		got := getBasicMongoDBConnectionURI(tt.connConfig)
		a.Equal(tt.want, got)
	}
}

func TestIsSystemCollection(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{
			name: "users",
			want: false,
		},
		{
			name: "system.namespaces",
			want: true,
		},
		{
			name: "system.users",
			want: true,
		},
		{
			name: "system.buckets.events",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isSystemCollection(tt.name))
		})
	}
}

// TestExecuteLogsParseFailure pins BYT-9950: when the statement fails to
// split/parse, Execute must still emit a COMMAND_EXECUTE / COMMAND_RESPONSE
// pair carrying the parse error so the failure is visible in task run logs,
// and must report failure.
func TestExecuteLogsParseFailure(t *testing.T) {
	const statement = "db.users.find();\nthis is not mongosh"

	run := func(t *testing.T, logStatement bool) []*storepb.TaskRunLog {
		var logs []*storepb.TaskRunLog
		opts := db.ExecuteOptions{
			CreateTaskRunLog: func(_ time.Time, l *storepb.TaskRunLog) error {
				logs = append(logs, l)
				return nil
			},
			LogCommandStatement: logStatement,
		}

		d := &Driver{}
		_, err := d.Execute(context.Background(), statement, opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "syntax error")

		require.Len(t, logs, 2)
		require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
		require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)
		require.Contains(t, logs[1].CommandResponse.Error, "syntax error")
		return logs
	}

	t.Run("range logging", func(t *testing.T) {
		logs := run(t, false)
		require.Empty(t, logs[0].CommandExecute.Statement)
		require.Equal(t, int32(0), logs[0].CommandExecute.Range.GetStart())
		require.Equal(t, int32(len(statement)), logs[0].CommandExecute.Range.GetEnd())
	})

	t.Run("statement logging", func(t *testing.T) {
		logs := run(t, true)
		require.Nil(t, logs[0].CommandExecute.Range)
		require.Equal(t, statement, logs[0].CommandExecute.GetStatement())
	})
}
