package server

import (
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// flight returns if the feature is enabled.
// By default, the feature is always enabled in dev mode, and you can decide if is available in the release mode.
func (s *Server) flight(feature api.FeatureType) bool {
	isEnabled, ok := api.FeatureFlight[feature]
	if !ok {
		return true
	}
	return isEnabled || s.profile.Mode == common.ReleaseModeDev
}
