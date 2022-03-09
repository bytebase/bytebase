package mysql

import "embed"

//go:generate ./fetch_mysql.sh mysql-8.0.28-macos11-arm64.tar.gz f1943053b12428e4c0e4ed309a636fd0

//go:embed mysql-8.0.28-macos11-arm64.tar.gz
// To use this package in testing, download the MySQL binary first:
// go generate ./...
var resources embed.FS
