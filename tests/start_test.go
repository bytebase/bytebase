package tests

import (
	"context"
	"testing"
)

func TestServiceRestart(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
		t.Fatal(err)
	}

	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}

	projects, err := ctl.getProjects()
	if err != nil {
		t.Fatal(err)
	}
	// Test seed should have more than one project.
	if len(projects) <= 1 {
		t.Errorf("unexpected number of projects %v", len(projects))
	}

	// Restart the server.
	if err := ctl.Close(); err != nil {
		t.Fatal(err)
	}

	if err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name())); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()

	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}
}
