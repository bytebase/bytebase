//go:build !release
// +build !release

package resources

import "embed"

//go:embed postgres-darwin-x86_64.txz postgres-linux-x86_64-alpine_linux.txz postgres-linux-x86_64.txz mysql-8.0.28-macos11-arm64.tar.gz mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz
var resources embed.FS
