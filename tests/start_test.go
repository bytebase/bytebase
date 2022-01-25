package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bytebase/bytebase/bin/server/cmd"
)

func TestServiceStart(t *testing.T) {
	ctx := context.Background()
	logger, err := cmd.GetLogger()
	if err != nil {
		panic(fmt.Errorf("failed to create logger, %w", err))
	}
	defer logger.Sync()
	profile := cmd.GetTestProfile(t.TempDir())
	m := cmd.NewMain(profile, logger)

	errChan := make(chan error, 1)
	go func() {
		if err := m.Run(ctx); err != nil {
			errChan <- fmt.Errorf("m.Run() error %w", err)
		}
	}()

	if err := waitForServerStart(m, errChan); err != nil {
		t.Fatal("failed to wait for server to start, %w", err)
	}

	if err := m.Close(); err != nil {
		t.Fatal("m.Close() error %w", err)
	}
}

func waitForServerStart(m *cmd.Main, errChan <-chan error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.GetServer() == nil {
				continue
			}
			e := m.GetServer().GetEcho()
			if e == nil {
				continue
			}
			addr := e.ListenerAddr()
			if addr != nil && strings.Contains(addr.String(), ":") {
				return nil // was started
			}
		case err := <-errChan:
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		}
	}
}
