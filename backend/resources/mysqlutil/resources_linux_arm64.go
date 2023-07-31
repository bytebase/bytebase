package mysqlutil

import "embed"

////go:generate ./build_mysqlutil.sh ../mysql/mysql-8.0.33-linux-glibc2.17-aarch64.tar.gz linux arm64 mysqlutil-8.0.33

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysqlutil ./...

//go:embed mysqlutil-8.0.33-linux-arm64.tar.gz
var resources embed.FS
