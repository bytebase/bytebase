//go:build mysql
// +build mysql

package mysql

import "embed"

// 55a7759e25cc527416150c8181ce3f6d is the md5 hash.
//go:generate ./fetch_mysql.sh mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz 55a7759e25cc527416150c8181ce3f6d

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz
var resources embed.FS
