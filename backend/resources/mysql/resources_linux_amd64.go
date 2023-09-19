//go:build mysql
// +build mysql

package mysql

import "embed"

//go:generate ./fetch_mysql.sh mysql-8.0.33-linux-glibc2.17-x86_64-minimal.tar.xz

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.33-linux-glibc2.17-x86_64-minimal.tar.xz
var resources embed.FS
