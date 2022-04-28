// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import "go.uber.org/zap"

var (
	logger *zap.Logger
)

const dsnUsage = `Database connection string.

DSN format:
  DRIVER://USERNAME[:PASSWORD]@HOST[:PORT]/[DATABASE][?PARAM=VALUE&...&PARAM=VALUE]

Drivers:
  mysql
  postgresql

Examples:
  mysql://root@localhost:3306/
  mysql://user:pass@localhost:3306/dbname
  postgresql://$(whoami)@localhost:5432/postgres
  postgresql://user:pass@localhost:5432/dbname?ssl-key=a&ssl-ca=b&ssl-cert=c
`
