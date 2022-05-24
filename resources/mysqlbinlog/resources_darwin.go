//go:build darwin
// +build darwin

package mysqlbinlog

import "embed"

//TODO(zp): need test for darwin amd64
//go:embed mysqlbinlog-8.0.28-macos11-arm64.tar.gz
var resources embed.FS
