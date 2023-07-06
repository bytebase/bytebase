//go:build mysql
// +build mysql

package mysql

import "embed"

// 0b98f999e8e6630a8e0966e8f867fc9d is the md5 hash.
//go:generate ./fetch_mysql.sh mysql-8.0.32-macos13-arm64.tar.gz 0b98f999e8e6630a8e0966e8f867fc9d

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.32-macos13-arm64.tar.gz
var resources embed.FS
