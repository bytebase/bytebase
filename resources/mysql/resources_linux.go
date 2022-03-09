package mysql

import "embed"

//go:generate ./fetch_mysql.sh mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz 55a7759e25cc527416150c8181ce3f6d

//go:embed mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz
// To use this package in testing, download the MySQL binary first:
// go generate ./...
var resources embed.FS
