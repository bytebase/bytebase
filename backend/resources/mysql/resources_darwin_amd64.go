//go:build mysql
// +build mysql

package mysql

import "embed"

// 7742d6746bf7d3fdf73e625420ae1d23 is the md5 hash.
//go:generate ./fetch_mysql.sh mysql-8.0.32-macos13-x86_64.tar.gz 7742d6746bf7d3fdf73e625420ae1d23

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.32-macos13-x86_64.tar.gz
var resources embed.FS
