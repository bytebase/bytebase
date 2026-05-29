package v1

import "time"

const (
	vcsProviderUserActiveWindowDays = 90
	vcsProviderUserActiveWindow     = vcsProviderUserActiveWindowDays * 24 * time.Hour
)
