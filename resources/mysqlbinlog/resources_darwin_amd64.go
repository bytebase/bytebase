//go:build darwin && amd64
// +build darwin,amd64

package mysqlbinlog

import "embed"

//go:embed mysqlbinlog-8.0.28-macos11-x86_64.tar.gz
var resources embed.FS
