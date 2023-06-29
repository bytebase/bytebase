package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestGetMongoDBConnectionURL(t *testing.T) {
	tests := []struct {
		connConfig db.ConnectionConfig
		want       string
	}{
		{
			connConfig: db.ConnectionConfig{
				Host:     "localhost",
				Port:     "27017",
				Username: "",
				Password: "",
			},
			want: "mongodb://localhost:27017/?authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:     "localhost",
				Port:     "27017",
				Username: "",
				Password: "",
				Database: "sampleDB",
			},
			want: "mongodb://localhost:27017/sampleDB?authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:     "cluster0.sample.mongodb.net",
				Username: "bytebase",
				Password: "passwd",
				Database: "sampleDB",
				SRV:      true,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                   "cluster0.sample.mongodb.net",
				Username:               "bytebase",
				Password:               "passwd",
				Database:               "sampleDB",
				AuthenticationDatabase: "",
				SRV:                    true,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?authSource=admin",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:                   "cluster0.sample.mongodb.net",
				Username:               "bytebase",
				Password:               "passwd",
				Database:               "sampleDB",
				AuthenticationDatabase: "admin",
				SRV:                    true,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB?authSource=admin",
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		got := getMongoDBConnectionURI(tt.connConfig)
		a.Equal(tt.want, got)
	}
}
