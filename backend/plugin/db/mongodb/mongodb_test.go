package mongodb

import (
	"testing"

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
