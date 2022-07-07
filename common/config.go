package common

// ReleaseMode is the mode for release, such as dev or release.
type ReleaseMode string

const (
	// ReleaseModeProd is the prod mode.
	ReleaseModeProd ReleaseMode = "prod"
	// ReleaseModeDev is the dev mode.
	ReleaseModeDev ReleaseMode = "dev"
)
