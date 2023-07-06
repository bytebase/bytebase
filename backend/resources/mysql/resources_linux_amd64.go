//go:build mysql
// +build mysql

package mysql

import "embed"

// b553a35e0457e9414137adf78e67827d is the md5 hash.
//go:generate ./fetch_mysql.sh mysql-8.0.32-linux-glibc2.17-x86_64-minimal.tar.xz b553a35e0457e9414137adf78e67827d

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.32-linux-glibc2.17-x86_64-minimal.tar.xz
var resources embed.FS
