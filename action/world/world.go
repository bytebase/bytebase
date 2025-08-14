package world

import (
	"log/slog"
	"time"
)

// World is the world environment for bytebase-action.
type World struct {
	Logger      *slog.Logger
	Platform    JobPlatform
	CurrentTime time.Time

	// bytebase-action flags
	Output               string
	URL                  string
	ServiceAccount       string
	ServiceAccountSecret string
	Project              string // projects/{project}
	Targets              []string
	FilePattern          string

	// bytebase-action check flags
	// An enum to determine should we fail on warning or error.
	// Valid values:
	// - SKIP
	// - FAIL_ON_WARNING
	// - FAIL_ON_ERROR
	CheckRelease string

	// bytebase-action rollout flags
	ReleaseTitle string // The title of the release
	// An enum to determine should we run plan checks and fail on warning or error.
	// Valid values:
	// - SKIP
	// - FAIL_ON_WARNING
	// - FAIL_ON_ERROR
	CheckPlan string
	// Rollout up to the target-stage.
	// Format: environments/{environment}
	TargetStage string
	Plan        string
	// Whether to use declarative mode.
	Declarative bool

	OutputMap map[string]string
}

func NewWorld() *World {
	return &World{
		CurrentTime: time.Now(),
		Logger:      slog.Default(),
		OutputMap:   make(map[string]string),
	}
}
