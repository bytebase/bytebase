package utils

import "runtime"

// These paths must be consistent with the Dockerfile where decompressing the txz files.
// These paths are used to access the resources from the docker image.
var MongoUtilsDir string
var MySQLUtilsDir string
var PostgresUtilSource string

func init() {
	switch {
	case runtime.GOARCH == "amd64":
		MongoUtilsDir = "/var/opt/bytebase/resources/mongoutil-1.6.1-linux-amd64"
		MySQLUtilsDir = "/var/opt/bytebase/resources/mysqlutil-8.0.33-linux-amd64"
		PostgresUtilSource = "/var/opt/bytebase/resources/postgres-linux-amd64-16"
	case runtime.GOARCH == "arm64":
		MongoUtilsDir = "/var/opt/bytebase/resources/mongoutil-1.6.1-linux-arm64"
		MySQLUtilsDir = "/var/opt/bytebase/resources/mysqlutil-8.0.33-linux-arm64"
		PostgresUtilSource = "/var/opt/bytebase/resources/postgres-linux-arm64-16"
	}
}
