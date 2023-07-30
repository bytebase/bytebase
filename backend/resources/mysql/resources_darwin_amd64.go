//go:build mysql
// +build mysql

package mysql

import "embed"

//go:generate ./fetch_mysql.sh mysql-8.0.33-macos13-x86_64.tar.gz 4ac08ea50a719c099eee3acf7dfab211

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.33-macos13-x86_64.tar.gz
var resources embed.FS
