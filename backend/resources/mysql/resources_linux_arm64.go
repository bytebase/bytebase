//go:build mysql
// +build mysql

package mysql

import "embed"

//go:generate ./fetch_mysql.sh mysql-8.0.33-linux-glibc2.17-aarch64-minimal.tar.gz f5e01242975453ec6db02727848975c9

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.33-linux-glibc2.17-aarch64-minimal.tar.gz
var resources embed.FS
