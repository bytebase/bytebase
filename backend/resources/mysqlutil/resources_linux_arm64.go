package mysqlutil

import "embed"

//go:generate ./build_mysqlutil.sh ../mysql/mysql-8.0.32-linux-glibc2.17-aarch64.tar.gz linux amd64 mysqlutil-8.0.32

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysqlutil ./...

//go:embed mysqlutil-8.0.32-linux-arm64.tar.gz
var resources embed.FS
