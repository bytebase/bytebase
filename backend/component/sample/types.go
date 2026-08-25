// Package sample defines the application contract shared by the self-host and
// SaaS sample implementations.
package sample

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/bytebase/bytebase/backend/store"
)

// SetupRequest identifies the workspace and database-owning project.
type SetupRequest struct {
	WorkspaceID string
	ProjectID   string
}

// SetupResult contains every instance created by the selected implementation.
type SetupResult struct {
	Instances []*store.InstanceMessage
}

// InstanceInfo describes one provisioned sample instance.
type InstanceInfo struct {
	Instance   string
	ExpireTime *time.Time
}

// Info describes availability and provisioned resources for one workspace.
type Info struct {
	Available bool
	Instances []InstanceInfo
}

// Manager is the lifecycle interface implemented independently by self-host
// and SaaS sample managers.
type Manager interface {
	SetupSample(context.Context, SetupRequest) (*SetupResult, error)
	Info(context.Context, string) (*Info, error)
	Start(context.Context, string) error
	Cleanup(context.Context) error
	ValidateInstanceRestore(context.Context, string, string) error
	HandleInstanceLifecycle(context.Context, string, string, bool) error
	Stop()
}

// ManagerOptions control manager identity and test seams.
type ManagerOptions struct {
	Clock     func() time.Time
	Random    io.Reader
	Logger    *slog.Logger
	ReplicaID string
}

// FailureKind classifies failures at the manager boundary.
type FailureKind string

const (
	FailureUnknown            FailureKind = "unknown"
	FailureFailedPrecondition FailureKind = "failed_precondition"
	FailureUnavailable        FailureKind = "unavailable"
	FailureDeadlineExceeded   FailureKind = "deadline_exceeded"
)

type failure struct {
	kind FailureKind
	err  error
}

func (e *failure) Error() string {
	if e.err == nil {
		return string(e.kind)
	}
	return e.err.Error()
}

func (e *failure) Unwrap() error { return e.err }

// FailureKindOf returns the manager failure classification.
func FailureKindOf(err error) FailureKind {
	var classified *failure
	if errors.As(err, &classified) {
		return classified.kind
	}
	return FailureUnknown
}

// NewFailure classifies err for transport adapters.
func NewFailure(kind FailureKind, err error) error {
	if kind == FailureUnknown {
		return err
	}
	return &failure{kind: kind, err: err}
}
