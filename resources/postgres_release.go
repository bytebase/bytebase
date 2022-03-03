//go:build release
// +build release

package resources

import "embed"

//go:embed postgres-linux-x86_64-alpine_linux.txz
var postgresResources embed.FS
