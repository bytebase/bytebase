package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/db"
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
			want: "mongodb://localhost:27017",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:     "localhost",
				Port:     "27017",
				Username: "",
				Password: "",
				Database: "sampleDB",
			},
			want: "mongodb://localhost:27017/sampleDB",
		},
		{
			connConfig: db.ConnectionConfig{
				Host:     "cluster0.sample.mongodb.net",
				Username: "bytebase",
				Password: "passwd",
				Database: "sampleDB",
				SRV:      true,
			},
			want: "mongodb+srv://bytebase:passwd@cluster0.sample.mongodb.net/sampleDB",
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

func TestReplaceCharactersWithPercentEncoding(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{
			s:    "passw@rd",
			want: `passw%40rd`,
		},
		{
			s:    "passw@rd:/?#[]",
			want: `passw%40rd%3A%2F%3F%23%5B%5D`,
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		got := replaceCharacterWithPercentEncoding(tt.s)
		a.Equal(tt.want, got)
	}
}
