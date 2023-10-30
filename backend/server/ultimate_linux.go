//go:build !minimal

package server

import (
	// Drivers under linux.
	_ "github.com/bytebase/bytebase/backend/plugin/db/obo"
)
