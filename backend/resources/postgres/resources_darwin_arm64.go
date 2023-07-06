package postgres

import "embed"

//go:embed postgres-darwin-amd64.txz
var resources embed.FS
