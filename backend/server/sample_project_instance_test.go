package server

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigureSampleProjectManagerDisablesInvalidTarget(t *testing.T) {
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	manager := configureSampleProjectManager(
		context.Background(),
		"postgresql://control:secret@127.0.0.1:5432/postgres?sslmode=prefer",
		nil,
		nil,
		"replica",
	)

	require.Nil(t, manager)
	require.Contains(t, output.String(), "Sample Project Instance is disabled")
	require.NotContains(t, output.String(), "secret")
}

func TestConfigureSampleProjectManagerIgnoresEmptyTarget(t *testing.T) {
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	manager := configureSampleProjectManager(context.Background(), "", nil, nil, "replica")

	require.Nil(t, manager)
	require.Empty(t, output.String())
}

func TestConfigureSampleProjectManagerRetainsTemporarilyUnavailableTarget(t *testing.T) {
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	manager := configureSampleProjectManager(
		context.Background(),
		"postgresql://control:secret@127.0.0.1:1/postgres",
		nil,
		nil,
		"replica",
	)

	require.NotNil(t, manager)
	require.Contains(t, output.String(), "temporarily unavailable")
	require.NotContains(t, output.String(), "secret")
}
