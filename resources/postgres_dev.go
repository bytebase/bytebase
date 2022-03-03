//go:build !release
// +build !release

package resources

import "embed"

//go:embed postgres-darwin-x86_64.txz postgres-linux-x86_64-alpine_linux.txz postgres-linux-x86_64.txz
var postgresResources embed.FS
