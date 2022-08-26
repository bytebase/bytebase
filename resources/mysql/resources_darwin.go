//go:build mysql
// +build mysql

package mysql

import "embed"

// f1943053b12428e4c0e4ed309a636fd0 is the md5 hash.
//go:generate ./fetch_mysql.sh mysql-8.0.28-macos11-arm64.tar.gz f1943053b12428e4c0e4ed309a636fd0

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.28-macos11-arm64.tar.gz
var resources embed.FS
