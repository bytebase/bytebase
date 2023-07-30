//go:build mysql
// +build mysql

package mysql

import "embed"

//go:generate ./fetch_mysql.sh mysql-8.0.33-macos13-arm64.tar.gz 3e1698ba3cd1283ede4426f062357e1f

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.33-macos13-arm64.tar.gz
var resources embed.FS
