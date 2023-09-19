package postgres

import "embed"

//go:embed postgres-darwin-arm64.txz
var resources embed.FS
