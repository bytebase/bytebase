//go:build darwin
// +build darwin

package mysqlbinlog

import "embed"

//go:embed mysqlbinlog-8.0.28-macos11-arm64.tar.gz
var resources embed.FS
