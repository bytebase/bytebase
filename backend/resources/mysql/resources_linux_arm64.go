//go:build mysql
// +build mysql

package mysql

import "embed"

//go:generate ./fetch_mysql.sh mysql-8.0.33-linux-glibc2.17-aarch64.tar.gz b8e3d807610074f0c9eddd55520baf7b

// To use this package in testing, download the MySQL binary first:
// go generate -tags mysql ./...
//
//go:embed mysql-8.0.33-linux-glibc2.17-aarch64.tar.gz
var resources embed.FS
