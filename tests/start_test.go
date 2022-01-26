package tests

import (
	"context"
	"testing"
)

func TestServiceStart(t *testing.T) {
	ctx := context.Background()
	ctl := &controller{}
	if err := ctl.StartMain(ctx, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()

	if err := ctl.Login(); err != nil {
		t.Fatal(err)
	}
}
