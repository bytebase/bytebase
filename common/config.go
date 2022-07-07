package common

import "github.com/bytebase/bytebase/plugin/advisor"

// ReleaseMode is the mode for release, such as dev or release.
type ReleaseMode string

const (
	// ReleaseModeProd is the prod mode.
	ReleaseModeProd ReleaseMode = "prod"
	// ReleaseModeDev is the dev mode.
	ReleaseModeDev ReleaseMode = "dev"
)

// ConvertToAdvisorReleaseMode convert to advisor release mode.
func (mode ReleaseMode) ConvertToAdvisorReleaseMode() advisor.ReleaseMode {
	switch mode {
	case ReleaseModeDev:
		return advisor.ReleaseModeDev
	case ReleaseModeProd:
		return advisor.ReleaseModeProd
	}
	return advisor.ReleaseModeDev
}
