package postgres

import "embed"

//go:embed postgres-darwin-x86_64.txz
var resources embed.FS
