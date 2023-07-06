//go:build mysql
// +build mysql

package mysql

import "embed"

// 5538ac53ee979667dbf636d53023cc38 is the md5 hash.
//go:generate ./fetch_mysql.sh mysql-8.0.32-linux-glibc2.17-aarch64.tar.gz 5538ac53ee979667dbf636d53023cc38

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.32-linux-glibc2.17-aarch64.tar.gz
var resources embed.FS
