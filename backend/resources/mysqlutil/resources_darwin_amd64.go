package mysqlutil

import "embed"

//go:generate ./build_mysqlutil.sh ../mysql/mysql-8.0.32-macos13-x86_64.tar.gz darwin amd64 mysqlutil-8.0.32

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysqlutil ./...

//go:embed mysqlutil-8.0.32-darwin-amd64.tar.gz
var resources embed.FS
