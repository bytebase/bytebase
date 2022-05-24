//go:build darwin && x86_64
// +build darwin,x86_64

package mysqlbinlog

import "embed"

//go:embed mysqlbinlog-8.0.28-macos11-x86_64.tar.gz
var resources embed.FS
