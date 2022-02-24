// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"os"

	"go.uber.org/zap"
)

var (
	databaseType string
	username     string
	password     string
	hostname     string
	port         string
	database     string
	file         string

	// SSL flags.
	sslCA   string // server-ca.pem
	sslCert string // client-cert.pem
	sslKey  string // client-key.pem

	// Dump options.
	schemaOnly bool

	logger *zap.Logger
)

var envKeyToFlag = map[string]*string{
	"BB_USERNAME": &username,
	"BB_PASSWORD": &password,
}

func init() {
	for env, v := range envKeyToFlag {
		*v = os.Getenv(env)
	}
}
