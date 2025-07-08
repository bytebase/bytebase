package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bytebase/bytebase/action/args"
	"github.com/bytebase/bytebase/action/command"
	"github.com/bytebase/bytebase/action/world"
	"github.com/bytebase/bytebase/backend/common/log"
)

func main() {
	slog.Info("bytebase-action version " + args.Version + " built at commit " + args.Gitcommit)
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	// Trigger graceful shutdown on SIGINT or SIGTERM.
	// The default signal sent by the `kill` command is SIGTERM,
	// which is taken as the graceful shutdown signal for many systems, eg., Kubernetes, Gunicorn.
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		slog.Info(fmt.Sprintf("%s received.", sig.String()))
		cancel()
	}()

	w := world.NewWorld()
	cmd := command.NewRootCommand(w)

	if err := cmd.ExecuteContext(ctx); err != nil {
		slog.Error("failed to execute command", log.BBError(err))
		os.Exit(1)
	}
}
