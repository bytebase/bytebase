package tests

import (
	"context"
	"testing"
)

func TestServiceStart(t *testing.T) {
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	if err := ctl.StartMain(ctx, dataDir); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()

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
}
