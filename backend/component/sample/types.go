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

// PrepareRequest identifies the workspace and owning project.
type PrepareRequest struct {
	WorkspaceID string
	ProjectID   string
}

// Instance describes one provisioned sample instance.
type Instance struct {
	Name       string
	ExpireTime *time.Time
}

// Manager is the lifecycle interface implemented independently by self-host
// and SaaS sample managers.
type Manager interface {
	CheckAvailable(context.Context) error
	PrepareSampleProjectInstance(context.Context, PrepareRequest) (*store.InstanceMessage, error)
	ListInstances(context.Context, string) ([]*Instance, error)
	Start(context.Context, string) error
	Cleanup(context.Context) error
	ValidateInstanceRestore(context.Context, string, string) error
	HandleInstanceLifecycle(context.Context, string, string, bool) error
	HandleProjectPurge(context.Context, string, string) error
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
