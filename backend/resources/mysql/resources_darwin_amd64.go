//go:build mysql
// +build mysql

package mysql

import "embed"

//go:generate ./fetch_mysql.sh mysql-8.0.33-macos13-x86_64.tar.gz

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.33-macos13-x86_64.tar.gz
var resources embed.FS
