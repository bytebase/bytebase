package world

import (
	"log/slog"
	"time"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// World is the world environment for bytebase-action.
type World struct {
	Logger      *slog.Logger
	Platform    JobPlatform
	CurrentTime time.Time

	// Whether it is the rollout subcommand.
	IsRollout bool

	// bytebase-action flags
	Output               string
	URL                  string
	ServiceAccount       string
	ServiceAccountSecret string
	Project              string // projects/{project}
	Targets              []string
	FilePattern          string
	// Whether to use declarative mode.
	Declarative bool

	// bytebase-action check flags
	// An enum to determine should we fail on warning or error.
	// Valid values:
	// - SKIP
	// - FAIL_ON_WARNING
	// - FAIL_ON_ERROR
	CheckRelease string
	// Custom linting rules in natural language for AI-powered validation.
	CustomRules string

	// bytebase-action rollout flags
	ReleaseTitle string // The title of the release
	// Rollout up to the target-stage.
	// Format: environments/{environment}
	TargetStage string
	Plan        string

	// Outputs
	OutputMap struct {
		Release      string                     `json:"release,omitempty"`
		Plan         string                     `json:"plan,omitempty"`
		Rollout      string                     `json:"rollout,omitempty"`
		CheckResults *v1pb.CheckReleaseResponse `json:"checkResults,omitempty"`
	}
	Rollout *v1pb.Rollout
}

func NewWorld() *World {
	return &World{
		CurrentTime: time.Now(),
		Logger:      slog.Default(),
	}
}
