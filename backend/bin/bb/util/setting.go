package util

import (
	"github.com/bytebase/bytebase/backend/bin/bb/config"
)

// Setting is passed to each command. This eases swapping implementation.
type Setting struct {
	Config *config.Config
}
