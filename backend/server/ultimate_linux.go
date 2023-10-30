//go:build !minimal

package server

import (
	_ "github.com/bytebase/bytebase/backend/plugin/db/obo"
)
