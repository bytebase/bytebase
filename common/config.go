package common

// ReleaseMode is the mode for release, such as dev or release.
type ReleaseMode string

const (
	// ReleaseModeRelease is the release mode.
	ReleaseModeRelease ReleaseMode = "release"
	// ReleaseModeDev is the dev mode.
	ReleaseModeDev ReleaseMode = "dev"
)
