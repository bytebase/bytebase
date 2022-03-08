package tests

import (
	"context"
	"fmt"
	"testing"
)

func TestServiceRestart(t *testing.T) {
	t.Parallel()
	err := func() error {
		ctx := context.Background()
		ctl := &controller{}
		dataDir := t.TempDir()
		if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
			return err
		}

		if err := ctl.Login(); err != nil {
			return err
		}

		projects, err := ctl.getProjects()
		if err != nil {
			return err
		}
		// Test seed should have more than one project.
		if len(projects) <= 1 {
			return fmt.Errorf("unexpected number of projects %v", len(projects))
		}

		// Restart the server.
		if err := ctl.Close(); err != nil {
			return err
		}

		if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
			return err
		}
		defer ctl.Close()

		if err := ctl.Login(); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		t.Error(err)
	}
}
