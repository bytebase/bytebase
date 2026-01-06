package store

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

// SignalChannel is the PostgreSQL NOTIFY channel for HA coordination.
const SignalChannel = "bytebase_signal"

// Signal represents a notification payload sent via PostgreSQL NOTIFY.
type Signal struct {
	Type string `json:"type"`
	UID  int    `json:"uid"`
}

// Signal types.
const (
	SignalTypeCancelPlanCheckRun = "cancel_plan_check_run"
	SignalTypeCancelTaskRun      = "cancel_task_run"
)

// SendSignal sends a notification to the bytebase_signal channel.
func (s *Store) SendSignal(ctx context.Context, signalType string, uid int) error {
	payload, err := json.Marshal(&Signal{
		Type: signalType,
		UID:  uid,
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal signal payload")
	}
	_, err = s.GetDB().ExecContext(ctx, "SELECT pg_notify($1, $2)", SignalChannel, string(payload))
	return errors.Wrap(err, "failed to send signal")
}
